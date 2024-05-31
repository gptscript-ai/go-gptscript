package gptscript

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var (
	serverProcess       *exec.Cmd
	serverProcessCancel context.CancelFunc
	clientCount         int
	lock                sync.Mutex
)

const relativeToBinaryPath = "<me>"

type Client interface {
	Run(context.Context, string, Options) (*Run, error)
	Evaluate(context.Context, Options, ...fmt.Stringer) (*Run, error)
	Parse(context.Context, string) ([]Node, error)
	ParseTool(context.Context, string) ([]Node, error)
	Version(context.Context) (string, error)
	Fmt(context.Context, []Node) (string, error)
	ListTools(context.Context) (string, error)
	ListModels(context.Context) ([]string, error)
	Confirm(context.Context, AuthResponse) error
	PromptResponse(context.Context, PromptResponse) error
	Close()
}

type client struct {
	gptscriptURL string
}

func NewClient() (Client, error) {
	lock.Lock()
	defer lock.Unlock()
	clientCount++

	serverURL := os.Getenv("GPTSCRIPT_URL")
	if serverURL == "" {
		serverURL = "127.0.0.1:9090"
	}

	if serverProcessCancel == nil && os.Getenv("GPTSCRIPT_DISABLE_SERVER") != "true" {
		ctx, cancel := context.WithCancel(context.Background())

		in, _ := io.Pipe()
		serverProcess = exec.CommandContext(ctx, getCommand(), "--listen-address", serverURL, "sdkserver")
		serverProcess.Stdin = in

		serverProcessCancel = func() {
			cancel()
			_ = in.Close()
		}

		if err := serverProcess.Start(); err != nil {
			serverProcessCancel()
			return nil, fmt.Errorf("failed to start server: %w", err)
		}

		timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		if err := waitForServerReady(timeoutCtx, serverURL); err != nil {
			serverProcessCancel()
			_ = serverProcess.Wait()
			return nil, fmt.Errorf("failed to wait for gptscript to be ready: %w", err)
		}
	}
	return &client{gptscriptURL: "http://" + serverURL}, nil
}

func waitForServerReady(ctx context.Context, serverURL string) error {
	for {
		resp, err := http.Get("http://" + serverURL + "/healthz")
		if err != nil {
			slog.DebugContext(ctx, "waiting for server to become ready")
		} else {
			_ = resp.Body.Close()

			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Second):
		}
	}
}

func (c *client) Close() {
	lock.Lock()
	defer lock.Unlock()
	clientCount--

	if clientCount == 0 && serverProcessCancel != nil {
		serverProcessCancel()
		_ = serverProcess.Wait()
	}
}

func (c *client) Evaluate(ctx context.Context, opts Options, tools ...fmt.Stringer) (*Run, error) {
	return (&Run{
		url:         c.gptscriptURL,
		requestPath: "evaluate",
		state:       Creating,
		opts:        opts,
		content:     concatTools(tools),
		chatState:   opts.ChatState,
	}).NextChat(ctx, opts.Input)
}

func (c *client) Run(ctx context.Context, toolPath string, opts Options) (*Run, error) {
	return (&Run{
		url:         c.gptscriptURL,
		requestPath: "run",
		state:       Creating,
		opts:        opts,
		toolPath:    toolPath,
		chatState:   opts.ChatState,
	}).NextChat(ctx, opts.Input)
}

// Parse will parse the given file into an array of Nodes.
func (c *client) Parse(ctx context.Context, fileName string) ([]Node, error) {
	out, err := c.runBasicCommand(ctx, "parse", map[string]any{"file": fileName})
	if err != nil {
		return nil, err
	}

	var doc Document
	if err = json.Unmarshal([]byte(out), &doc); err != nil {
		return nil, err
	}

	return doc.Nodes, nil
}

// ParseTool will parse the given string into a tool.
func (c *client) ParseTool(ctx context.Context, toolDef string) ([]Node, error) {
	out, err := c.runBasicCommand(ctx, "parse", map[string]any{"content": toolDef})
	if err != nil {
		return nil, err
	}

	var doc Document
	if err = json.Unmarshal([]byte(out), &doc); err != nil {
		return nil, err
	}

	return doc.Nodes, nil
}

// Fmt will format the given nodes into a string.
func (c *client) Fmt(ctx context.Context, nodes []Node) (string, error) {
	out, err := c.runBasicCommand(ctx, "fmt", Document{Nodes: nodes})
	if err != nil {
		return "", err
	}

	return out, nil
}

// Version will return the output of `gptscript --version`
func (c *client) Version(ctx context.Context) (string, error) {
	out, err := c.runBasicCommand(ctx, "version", nil)
	if err != nil {
		return "", err
	}

	return out, nil
}

// ListTools will list all the available tools.
func (c *client) ListTools(ctx context.Context) (string, error) {
	out, err := c.runBasicCommand(ctx, "list-tools", nil)
	if err != nil {
		return "", err
	}

	return out, nil
}

// ListModels will list all the available models.
func (c *client) ListModels(ctx context.Context) ([]string, error) {
	out, err := c.runBasicCommand(ctx, "list-models", nil)
	if err != nil {
		return nil, err
	}

	return strings.Split(strings.TrimSpace(out), "\n"), nil
}

func (c *client) Confirm(ctx context.Context, resp AuthResponse) error {
	_, err := c.runBasicCommand(ctx, "confirm/"+resp.ID, resp)
	return err
}

func (c *client) PromptResponse(ctx context.Context, resp PromptResponse) error {
	_, err := c.runBasicCommand(ctx, "prompt-response/"+resp.ID, resp.Response)
	return err
}

func (c *client) runBasicCommand(ctx context.Context, requestPath string, body any) (string, error) {
	run := &Run{
		url:          c.gptscriptURL,
		requestPath:  requestPath,
		state:        Creating,
		basicCommand: true,
	}

	if err := run.request(ctx, body); err != nil {
		return "", err
	}

	out, err := run.Text()
	if err != nil {
		return "", err
	}
	if run.err != nil {
		return run.ErrorOutput(), run.err
	}

	return out, nil
}

func getCommand() string {
	if gptScriptBin := os.Getenv("GPTSCRIPT_BIN"); gptScriptBin != "" {
		if len(os.Args) == 0 {
			return gptScriptBin
		}
		return determineProperCommand(filepath.Dir(os.Args[0]), gptScriptBin)
	}

	return "gptscript"
}

// determineProperCommand is for testing purposes. Users should use getCommand instead.
func determineProperCommand(dir, bin string) string {
	if !strings.HasPrefix(bin, relativeToBinaryPath) {
		return bin
	}

	bin = filepath.Join(dir, strings.TrimPrefix(bin, relativeToBinaryPath))
	if !filepath.IsAbs(bin) {
		bin = "." + string(os.PathSeparator) + bin
	}

	slog.Debug("Using gptscript binary: " + bin)
	return bin
}
