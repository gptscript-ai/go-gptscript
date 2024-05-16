package gptscript

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

const relativeToBinaryPath = "<me>"

type ClientOpts struct {
	GPTScriptURL string
	GPTScriptBin string
}

type Client struct {
	opts ClientOpts
}

func NewClient(opts ClientOpts) *Client {
	c := &Client{opts: opts}
	c.complete()
	return c
}

func (c *Client) complete() {
	if c.opts.GPTScriptBin == "" {
		c.opts.GPTScriptBin = getCommand()
	}
}

func (c *Client) Evaluate(ctx context.Context, opts Opts, tools ...fmt.Stringer) (*Run, error) {
	return (&Run{
		url:         c.opts.GPTScriptURL,
		binPath:     c.opts.GPTScriptBin,
		requestPath: "evaluate",
		state:       Creating,
		opts:        opts,
		content:     concatTools(tools),
		chatState:   opts.ChatState,
	}).NextChat(ctx, opts.Input)
}

func (c *Client) Run(ctx context.Context, toolPath string, opts Opts) (*Run, error) {
	return (&Run{
		url:         c.opts.GPTScriptURL,
		binPath:     c.opts.GPTScriptBin,
		requestPath: "run",
		state:       Creating,
		opts:        opts,
		toolPath:    toolPath,
		chatState:   opts.ChatState,
	}).NextChat(ctx, opts.Input)
}

// Parse will parse the given file into an array of Nodes.
func (c *Client) Parse(ctx context.Context, fileName string) ([]Node, error) {
	out, err := c.runBasicCommand(ctx, "parse", "parse", fileName, "")
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
func (c *Client) ParseTool(ctx context.Context, toolDef string) ([]Node, error) {
	out, err := c.runBasicCommand(ctx, "parse", "parse", "", toolDef)
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
func (c *Client) Fmt(ctx context.Context, nodes []Node) (string, error) {
	b, err := json.Marshal(Document{Nodes: nodes})
	if err != nil {
		return "", fmt.Errorf("failed to marshal nodes: %w", err)
	}

	run := &runSubCommand{
		Run: Run{
			url:         c.opts.GPTScriptURL,
			binPath:     c.opts.GPTScriptBin,
			requestPath: "fmt",
			state:       Creating,
			toolPath:    "",
			content:     string(b),
		},
	}

	if run.url != "" {
		err = run.request(ctx, Document{Nodes: nodes})
	} else {
		err = run.exec(ctx, "fmt")
	}
	if err != nil {
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

// Version will return the output of `gptscript --version`
func (c *Client) Version(ctx context.Context) (string, error) {
	out, err := c.runBasicCommand(ctx, "--version", "version", "", "")
	if err != nil {
		return "", err
	}

	return out, nil
}

// ListTools will list all the available tools.
func (c *Client) ListTools(ctx context.Context) (string, error) {
	out, err := c.runBasicCommand(ctx, "--list-tools", "list-tools", "", "")
	if err != nil {
		return "", err
	}

	return out, nil
}

// ListModels will list all the available models.
func (c *Client) ListModels(ctx context.Context) ([]string, error) {
	out, err := c.runBasicCommand(ctx, "--list-models", "list-models", "", "")
	if err != nil {
		return nil, err
	}

	return strings.Split(strings.TrimSpace(out), "\n"), nil
}

func (c *Client) runBasicCommand(ctx context.Context, command, requestPath, toolPath, content string) (string, error) {
	run := &runSubCommand{
		Run: Run{
			url:         c.opts.GPTScriptURL,
			binPath:     c.opts.GPTScriptBin,
			requestPath: requestPath,
			state:       Creating,
			toolPath:    toolPath,
			content:     content,
		},
	}

	var err error
	if run.url != "" {
		var m any
		if content != "" || toolPath != "" {
			m = map[string]any{"input": content, "file": toolPath}
		}
		err = run.request(ctx, m)
	} else {
		err = run.exec(ctx, command)
	}
	if err != nil {
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
