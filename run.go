package gptscript

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
)

type Run struct {
	url, binPath, requestPath, toolPath, content string
	opts                                         Opts
	state                                        RunState
	chatState                                    string
	cmd                                          *exec.Cmd
	err                                          error
	stdout, stderr                               io.Reader
	wait                                         func() error

	rawOutput      map[string]any
	output, errput []byte
	events         chan Event
	lock           sync.Mutex
	complete       bool
}

// Text returns the text output of the gptscript. It blocks until the output is ready.
func (r *Run) Text() (string, error) {
	out, err := r.Bytes()
	if err != nil {
		return "", err
	}

	return string(out), nil
}

// Bytes returns the output of the gptscript in bytes. It blocks until the output is ready.
func (r *Run) Bytes() ([]byte, error) {
	if err := r.readAllOutput(); err != nil {
		return nil, err
	}
	if r.err != nil {
		return nil, r.err
	}

	return r.output, nil
}

// State returns the current state of the gptscript.
func (r *Run) State() RunState {
	return r.state
}

// ErrorOutput returns the stderr output of the gptscript.
// Should only be called after Bytes or Text has returned an error.
func (r *Run) ErrorOutput() string {
	return string(r.errput)
}

// Events returns a channel that streams the gptscript events as they occur.
func (r *Run) Events() <-chan Event {
	return r.events
}

// Close will stop the gptscript run, if it is running.
func (r *Run) Close() error {
	// If the command was not started, then report error.
	if r.cmd == nil || r.cmd.Process == nil {
		return fmt.Errorf("run not started")
	}

	// If the command has already exited, then nothing to do.
	if r.cmd.ProcessState == nil {
		return nil
	}

	if err := r.cmd.Process.Signal(os.Kill); err != nil {
		return err
	}

	return r.wait()
}

// RawOutput returns the raw output of the gptscript. Most users should use Text or Bytes instead.
func (r *Run) RawOutput() (map[string]any, error) {
	if _, err := r.Bytes(); err != nil {
		return nil, err
	}
	return r.rawOutput, nil
}

// NextChat will pass input and create the next run in a chat.
// The new Run will be returned.
func (r *Run) NextChat(ctx context.Context, input string) (*Run, error) {
	if r.state != Creating && r.state != Continue {
		return nil, fmt.Errorf("run must be in creating or continue state not %q", r.state)
	}

	run := &Run{
		url:         r.url,
		binPath:     r.binPath,
		requestPath: r.requestPath,
		state:       Creating,
		chatState:   r.chatState,
		toolPath:    r.toolPath,
		content:     r.content,
		opts:        r.opts,
	}
	run.opts.Input = input
	if run.chatState != "" {
		run.opts.ChatState = run.chatState
	}

	if run.url != "" {
		var payload any
		if r.content != "" {
			payload = requestPayload{
				Content: run.content,
				Input:   input,
				Opts:    run.opts,
			}
		} else if run.toolPath != "" {
			payload = requestPayload{
				File:  run.toolPath,
				Input: input,
				Opts:  run.opts,
			}
		}

		return run, run.request(ctx, payload)
	}

	return run, run.exec(ctx)
}

func (r *Run) exec(ctx context.Context, extraArgs ...string) error {
	eventsRead, eventsWrite, err := os.Pipe()
	if err != nil {
		r.state = Error
		r.err = fmt.Errorf("failed to create events reader: %w", err)
		return r.err
	}

	// Close the parent pipe after starting the child process
	defer eventsWrite.Close()

	chatState := r.chatState
	if chatState == "" {
		chatState = "null"
	}
	args := append(r.opts.toArgs(), "--chat-state="+chatState)
	args = append(args, extraArgs...)
	if r.toolPath != "" {
		args = append(args, r.toolPath)
	}

	cancelCtx, cancel := context.WithCancel(ctx)
	c, stdout, stderr, err := setupForkCommand(cancelCtx, r.binPath, r.content, r.opts.Input, args, eventsWrite)
	if err != nil {
		cancel()
		_ = eventsRead.Close()
		r.state = Error
		r.err = fmt.Errorf("failed to setup gptscript: %w", err)
		return r.err
	}

	if err = c.Start(); err != nil {
		cancel()
		_ = eventsRead.Close()
		r.state = Error
		r.err = fmt.Errorf("failed to start gptscript: %w", err)
		return r.err
	}

	r.state = Running
	r.cmd = c
	r.stdout = stdout
	r.stderr = stderr
	r.events = make(chan Event, 100)
	go r.readEvents(cancelCtx, eventsRead)

	r.wait = func() error {
		err := c.Wait()
		_ = eventsRead.Close()
		cancel()
		if err != nil {
			r.state = Error
			r.err = fmt.Errorf("failed to wait for gptscript: %w", err)
		} else {
			if r.state == Running {
				r.state = Finished
			}
		}
		return r.err
	}

	return nil
}

func (r *Run) readEvents(ctx context.Context, events io.Reader) {
	defer close(r.events)

	scan := bufio.NewScanner(events)
	for scan.Scan() {
		if !r.opts.IncludeEvents {
			continue
		}

		line := scan.Bytes()
		if len(line) == 0 {
			continue
		}

		var event Event
		if err := json.Unmarshal(line, &event); err != nil {
			slog.Debug("failed to unmarshal event", "error", err, "event", string(line))
			continue
		}

		select {
		case <-ctx.Done():
			go func() {
				for scan.Scan() {
					// Drain any remaining events
				}
			}()
			return
		case r.events <- event:
		}
	}
}

func (r *Run) readAllOutput() error {
	r.lock.Lock()
	defer r.lock.Unlock()
	if r.complete {
		return nil
	}
	r.complete = true

	done := true
	errChan := make(chan error)
	go func() {
		var err error
		r.errput, err = io.ReadAll(r.stderr)
		errChan <- err
	}()

	go func() {
		var err error
		r.output, err = io.ReadAll(r.stdout)
		errChan <- err
	}()

	for range 2 {
		err := <-errChan
		if err != nil {
			r.err = fmt.Errorf("failed to read output: %w", err)
		}
	}

	if isObject(r.output) {
		var chatOutput map[string]any
		if err := json.Unmarshal(r.output, &chatOutput); err != nil {
			r.state = Error
			r.err = fmt.Errorf("failed to parse chat output: %w", err)
		}

		chatState, err := json.Marshal(chatOutput["state"])
		if err != nil {
			r.state = Error
			r.err = fmt.Errorf("failed to process chat state: %w", err)
		}
		r.chatState = string(chatState)

		if content, ok := chatOutput["content"].(string); ok {
			r.output = []byte(content)
		}

		done, _ = chatOutput["done"].(bool)
		r.rawOutput = chatOutput
	} else {
		if unquoted, err := strconv.Unquote(string(r.output)); err == nil {
			r.output = []byte(unquoted)
		}
	}

	if r.err != nil {
		r.state = Error
	} else if done {
		r.state = Finished
	} else {
		r.state = Continue
	}

	return r.wait()
}

func (r *Run) request(ctx context.Context, payload any) (err error) {
	var (
		req               *http.Request
		url               = fmt.Sprintf("%s/%s", r.url, r.requestPath)
		cancelCtx, cancel = context.WithCancelCause(ctx)
	)

	defer func() {
		if err != nil {
			cancel(err)
		}
	}()

	if payload == nil {
		req, err = http.NewRequestWithContext(cancelCtx, http.MethodGet, url, nil)
	} else {
		var b []byte
		b, err = json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("failed to marshal payload: %w", err)
		}

		req, err = http.NewRequestWithContext(cancelCtx, http.MethodPost, url, bytes.NewReader(b))
	}
	if err != nil {
		r.state = Error
		r.err = fmt.Errorf("failed to create request: %w", err)
		return r.err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		r.state = Error
		r.err = fmt.Errorf("failed to make request: %w", err)
		return r.err
	}

	if resp.StatusCode != http.StatusOK {
		r.state = Error
		r.err = fmt.Errorf("unexpected response status: %s", resp.Status)
		return r.err
	}

	r.state = Running

	stdout, stdoutWriter := io.Pipe()
	stderr, stderrWriter := io.Pipe()
	eventsRead, eventsWrite := io.Pipe()

	r.stdout = stdout
	r.stderr = stderr

	r.events = make(chan Event, 100)
	go r.readEvents(cancelCtx, eventsRead)

	go func() {
		bufferedStdout := bufio.NewWriter(stdoutWriter)
		bufferedStderr := bufio.NewWriter(stderrWriter)
		scan := bufio.NewScanner(resp.Body)
		defer func() {
			go func() {
				for scan.Scan() {
					// Drain any remaining events
				}
			}()

			eventsWrite.Close()

			bufferedStderr.Flush()
			stderrWriter.Close()

			bufferedStdout.Flush()
			stdoutWriter.Close()

			cancel(r.err)

			resp.Body.Close()
		}()

		for scan.Scan() {
			line := bytes.TrimSpace(bytes.TrimPrefix(scan.Bytes(), []byte("data: ")))
			if len(line) == 0 {
				continue
			}
			if bytes.Equal(line, []byte("[DONE]")) {
				return
			}

			if bytes.HasPrefix(line, []byte(`{"stdout":`)) {
				_, err = bufferedStdout.Write(bytes.TrimSuffix(bytes.TrimPrefix(line, []byte(`{"stdout":`)), []byte("}")))
				if err != nil {
					r.state = Error
					r.err = fmt.Errorf("failed to write stdout: %w", err)
					return
				}
			} else if bytes.HasPrefix(line, []byte(`{"stderr":`)) {
				_, err = bufferedStderr.Write(bytes.TrimSuffix(bytes.TrimPrefix(line, []byte(`{"stderr":`)), []byte("}")))
				if err != nil {
					r.state = Error
					r.err = fmt.Errorf("failed to write stderr: %w", err)
					return
				}
			} else {
				_, err = eventsWrite.Write(append(line, '\n'))
				if err != nil {
					r.state = Error
					r.err = fmt.Errorf("failed to write events: %w", err)
					return
				}
			}
		}
	}()

	r.wait = func() error {
		<-cancelCtx.Done()
		stdout.Close()
		stderr.Close()
		eventsRead.Close()
		if err := context.Cause(cancelCtx); !errors.Is(err, context.Canceled) && r.err == nil {
			r.state = Error
			r.err = err
		}
		return r.err
	}

	return nil
}

type RunState string

const (
	Creating RunState = "creating"
	Running  RunState = "running"
	Continue RunState = "continue"
	Finished RunState = "finished"
	Error    RunState = "error"
)

func setupForkCommand(ctx context.Context, bin, content, input string, args []string, extraFiles ...*os.File) (*exec.Cmd, io.Reader, io.Reader, error) {
	var stdin io.Reader
	if content != "" {
		args = append(args, "-")
		stdin = strings.NewReader(content)
	}

	if input != "" {
		args = append(args, input)
	}

	c := exec.CommandContext(ctx, bin, args...)
	if len(extraFiles) > 0 {
		appendExtraFiles(c, extraFiles...)
	}

	if content != "" {
		c.Stdin = stdin
	}

	stdout, err := c.StdoutPipe()
	if err != nil {
		return nil, new(reader), new(reader), err
	}

	stderr, err := c.StderrPipe()
	if err != nil {
		return nil, stdout, new(reader), err
	}

	return c, stdout, stderr, nil
}

type runSubCommand struct {
	Run
}

func (r *runSubCommand) Bytes() ([]byte, error) {
	r.lock.Lock()
	defer r.lock.Unlock()
	if r.complete {
		return r.output, r.err
	}
	r.complete = true

	errChan := make(chan error)
	go func() {
		var err error
		r.errput, err = io.ReadAll(r.stderr)
		if unquoted, err := strconv.Unquote(string(r.errput)); err == nil {
			r.errput = []byte(unquoted)
		}
		errChan <- err
	}()

	go func() {
		var err error
		r.output, err = io.ReadAll(r.stdout)
		if unquoted, err := strconv.Unquote(string(r.output)); err == nil {
			r.output = []byte(unquoted)
		}
		errChan <- err
	}()

	for range 2 {
		err := <-errChan
		if err != nil {
			r.err = fmt.Errorf("failed to read output: %w", err)
		}
	}

	if r.err != nil {
		r.state = Error
	} else {
		r.state = Finished
	}

	if err := r.wait(); err != nil {
		return nil, err
	}

	return r.output, r.err
}

func (r *runSubCommand) Text() (string, error) {
	output, err := r.Bytes()
	return string(output), err
}

type requestPayload struct {
	Content string `json:"content"`
	File    string `json:"file"`
	Input   string `json:"input"`
	Opts    `json:",inline"`
}

func isObject(b []byte) bool {
	return len(b) > 0 && b[0] == '{'
}
