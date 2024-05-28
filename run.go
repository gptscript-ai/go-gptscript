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
	"os/exec"
	"strconv"
	"sync"
)

var errAbortRun = errors.New("run aborted")

type Run struct {
	url, requestPath, toolPath, content string
	opts                                Options
	state                               RunState
	chatState                           string
	cancel                              context.CancelCauseFunc
	err                                 error
	stdout, stderr                      io.Reader
	wait                                func() error

	rawOutput      map[string]any
	output, errput []byte
	events         chan Frame
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

// Err returns the error that caused the gptscript to fail, if any.
func (r *Run) Err() error {
	return r.err
}

// ErrorOutput returns the stderr output of the gptscript.
// Should only be called after Bytes or Text has returned an error.
func (r *Run) ErrorOutput() string {
	return string(r.errput)
}

// Events returns a channel that streams the gptscript events as they occur as Frames.
func (r *Run) Events() <-chan Frame {
	return r.events
}

// Close will stop the gptscript run, if it is running.
func (r *Run) Close() error {
	// If the command was not started, then report error.
	if r.cancel == nil {
		return fmt.Errorf("run not started")
	}

	r.cancel(errAbortRun)
	if r.wait == nil {
		return nil
	}

	if err := r.wait(); !errors.Is(err, errAbortRun) && !errors.Is(err, context.Canceled) && !errors.As(err, new(*exec.ExitError)) {
		return err
	}

	return nil
}

// RawOutput returns the raw output of the gptscript. Most users should use Text or Bytes instead.
func (r *Run) RawOutput() (map[string]any, error) {
	if _, err := r.Bytes(); err != nil {
		return nil, err
	}
	return r.rawOutput, nil
}

// ChatState returns the current chat state of the Run.
func (r *Run) ChatState() string {
	return r.chatState
}

// NextChat will pass input and create the next run in a chat.
// The new Run will be returned.
func (r *Run) NextChat(ctx context.Context, input string) (*Run, error) {
	if r.state != Creating && r.state != Continue {
		return nil, fmt.Errorf("run must be in creating or continue state not %q", r.state)
	}

	run := &Run{
		url:         r.url,
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

	var payload any
	if r.content != "" {
		payload = requestPayload{
			Content: run.content,
			Input:   input,
			Options: run.opts,
		}
	} else if run.toolPath != "" {
		payload = requestPayload{
			File:    run.toolPath,
			Input:   input,
			Options: run.opts,
		}
	}

	return run, run.request(ctx, payload)
}

func (r *Run) readEvents(ctx context.Context, events io.Reader) {
	defer close(r.events)

	var (
		err  error
		frag []byte

		b = make([]byte, 64*1024)
	)
	for n := 0; n != 0 || err == nil; n, err = events.Read(b) {
		if !r.opts.IncludeEvents {
			continue
		}

		for _, line := range bytes.Split(append(frag, b[:n]...), []byte("\n\n")) {
			if line = bytes.TrimSpace(line); len(line) == 0 {
				frag = frag[:0]
				continue
			}

			var event Frame
			if err := json.Unmarshal(line, &event); err != nil {
				slog.Debug("failed to unmarshal event", "error", err, "event", string(b))
				frag = line[:]
				continue
			}

			select {
			case <-ctx.Done():
				return
			case r.events <- event:
				frag = frag[:0]
			}
		}
	}

	if err != nil && !errors.Is(err, io.EOF) {
		slog.Debug("failed to read events", "error", err)
		r.err = fmt.Errorf("failed to read events: error: %w, stderr: %s", err, string(r.errput))
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

	r.cancel = cancel
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

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusBadRequest {
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

	r.events = make(chan Frame, 100)
	go r.readEvents(cancelCtx, eventsRead)

	go func() {
		var (
			err  error
			frag []byte
			buf  = make([]byte, 64*1024)
		)
		bufferedStdout := bufio.NewWriter(stdoutWriter)
		bufferedStderr := bufio.NewWriter(stderrWriter)
		defer func() {
			eventsWrite.Close()

			bufferedStderr.Flush()
			stderrWriter.Close()

			bufferedStdout.Flush()
			stdoutWriter.Close()

			cancel(r.err)

			resp.Body.Close()
		}()

		for n := 0; n != 0 || err == nil; n, err = resp.Body.Read(buf) {
			for _, line := range bytes.Split(bytes.TrimSpace(append(frag, buf[:n]...)), []byte("\n\n")) {
				line = bytes.TrimSpace(bytes.TrimPrefix(line, []byte("data: ")))
				if len(line) == 0 {
					frag = frag[:0]
					continue
				}
				if bytes.Equal(line, []byte("[DONE]")) {
					return
				}

				// Is this a JSON object?
				if err := json.Unmarshal(line, &[]map[string]any{make(map[string]any)}[0]); err != nil {
					// If not, then wait until we get the rest of the output.
					frag = line[:]
					continue
				}

				frag = frag[:0]

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
					_, err = eventsWrite.Write(append(line, '\n', '\n'))
					if err != nil {
						r.state = Error
						r.err = fmt.Errorf("failed to write events: %w", err)
						return
					}
				}
			}
		}

		if err != nil && !errors.Is(err, io.EOF) {
			slog.Debug("failed to read events from response", "error", err)
			r.err = fmt.Errorf("failed to read events: %w", err)
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
		} else if r.state != Continue {
			r.state = Finished
		}
		return r.err
	}

	return nil
}

type RunState string

func (rs RunState) IsTerminal() bool {
	return rs == Finished || rs == Error
}

const (
	Creating RunState = "creating"
	Running  RunState = "running"
	Continue RunState = "continue"
	Finished RunState = "finished"
	Error    RunState = "error"
)

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
	Options `json:",inline"`
}

func isObject(b []byte) bool {
	return len(b) > 0 && b[0] == '{'
}
