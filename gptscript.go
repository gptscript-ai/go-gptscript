package gptscript

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

var (
	serverProcess       *exec.Cmd
	serverProcessCancel context.CancelFunc
	gptscriptCount      int
	serverURL           string
	lock                sync.Mutex
)

const relativeToBinaryPath = "<me>"

type GPTScript struct {
	url       string
	globalEnv []string
}

func NewGPTScript(opts GlobalOptions) (*GPTScript, error) {
	lock.Lock()
	defer lock.Unlock()
	gptscriptCount++

	disableServer := os.Getenv("GPTSCRIPT_DISABLE_SERVER") == "true"

	if serverURL == "" && disableServer {
		serverURL = os.Getenv("GPTSCRIPT_URL")
	}

	if opts.Env == nil {
		opts.Env = os.Environ()
	}

	opts.Env = append(opts.Env, opts.toEnv()...)

	if serverProcessCancel == nil && !disableServer {
		ctx, cancel := context.WithCancel(context.Background())
		in, _ := io.Pipe()

		serverProcess = exec.CommandContext(ctx, getCommand(), "sys.sdkserver", "--listen-address", serverURL)
		serverProcess.Env = opts.Env[:]

		serverProcess.Stdin = in
		stdErr, err := serverProcess.StderrPipe()
		if err != nil {
			cancel()
			return nil, fmt.Errorf("failed to get stderr pipe: %w", err)
		}

		serverProcessCancel = func() {
			cancel()
			_ = in.Close()
			_ = serverProcess.Wait()
		}

		if err = serverProcess.Start(); err != nil {
			serverProcessCancel()
			return nil, fmt.Errorf("failed to start server: %w", err)
		}

		serverURL, err = readAddress(stdErr)
		if err != nil {
			serverProcessCancel()
			return nil, fmt.Errorf("failed to read server URL: %w", err)
		}

		go func() {
			for {
				// Ensure that stdErr is drained as logs come in
				_, _, _ = bufio.NewReader(stdErr).ReadLine()
			}
		}()

		if _, url, found := strings.Cut(serverURL, "addr="); found {
			// Ensure backwards compatibility with older versions of the SDK server
			serverURL = url
		}

		serverURL = strings.TrimSpace(serverURL)
	}
	g := &GPTScript{
		url: "http://" + serverURL,
	}

	if disableServer {
		g.globalEnv = opts.Env[:]
	}

	return g, nil
}

func readAddress(stdErr io.Reader) (string, error) {
	addr, err := bufio.NewReader(stdErr).ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read server address: %w", err)
	}

	if _, url, found := strings.Cut(addr, "addr="); found {
		// For backward compatibility: older versions of the SDK server print the address in a slightly different way.
		addr = url
	}

	return addr, nil
}

func (g *GPTScript) Close() {
	lock.Lock()
	defer lock.Unlock()
	gptscriptCount--

	if gptscriptCount == 0 && serverProcessCancel != nil {
		serverProcessCancel()
		_ = serverProcess.Wait()
	}
}

func (g *GPTScript) Evaluate(ctx context.Context, opts Options, tools ...ToolDef) (*Run, error) {
	opts.Env = append(g.globalEnv, opts.Env...)
	return (&Run{
		url:         g.url,
		requestPath: "evaluate",
		state:       Creating,
		opts:        opts,
		tools:       tools,
	}).NextChat(ctx, opts.Input)
}

func (g *GPTScript) Run(ctx context.Context, toolPath string, opts Options) (*Run, error) {
	opts.Env = append(g.globalEnv, opts.Env...)
	return (&Run{
		url:         g.url,
		requestPath: "run",
		state:       Creating,
		opts:        opts,
		toolPath:    toolPath,
	}).NextChat(ctx, opts.Input)
}

// Parse will parse the given file into an array of Nodes.
func (g *GPTScript) Parse(ctx context.Context, fileName string) ([]Node, error) {
	out, err := g.runBasicCommand(ctx, "parse", map[string]any{"file": fileName})
	if err != nil {
		return nil, err
	}

	var doc Document
	if err = json.Unmarshal([]byte(out), &doc); err != nil {
		return nil, err
	}

	for _, node := range doc.Nodes {
		node.TextNode.process()
	}

	return doc.Nodes, nil
}

// ParseTool will parse the given string into a tool.
func (g *GPTScript) ParseTool(ctx context.Context, toolDef string) ([]Node, error) {
	out, err := g.runBasicCommand(ctx, "parse", map[string]any{"content": toolDef})
	if err != nil {
		return nil, err
	}

	var doc Document
	if err = json.Unmarshal([]byte(out), &doc); err != nil {
		return nil, err
	}

	for _, node := range doc.Nodes {
		node.TextNode.process()
	}

	return doc.Nodes, nil
}

// Fmt will format the given nodes into a string.
func (g *GPTScript) Fmt(ctx context.Context, nodes []Node) (string, error) {
	for _, node := range nodes {
		node.TextNode.combine()
	}

	out, err := g.runBasicCommand(ctx, "fmt", Document{Nodes: nodes})
	if err != nil {
		return "", err
	}

	return out, nil
}

// Version will return the output of `gptscript --version`
func (g *GPTScript) Version(ctx context.Context) (string, error) {
	out, err := g.runBasicCommand(ctx, "version", nil)
	if err != nil {
		return "", err
	}

	return out, nil
}

// ListTools will list all the available tools.
func (g *GPTScript) ListTools(ctx context.Context) (string, error) {
	out, err := g.runBasicCommand(ctx, "list-tools", nil)
	if err != nil {
		return "", err
	}

	return out, nil
}

// ListModels will list all the available models.
func (g *GPTScript) ListModels(ctx context.Context) ([]string, error) {
	out, err := g.runBasicCommand(ctx, "list-models", nil)
	if err != nil {
		return nil, err
	}

	return strings.Split(strings.TrimSpace(out), "\n"), nil
}

func (g *GPTScript) Confirm(ctx context.Context, resp AuthResponse) error {
	_, err := g.runBasicCommand(ctx, "confirm/"+resp.ID, resp)
	return err
}

func (g *GPTScript) PromptResponse(ctx context.Context, resp PromptResponse) error {
	_, err := g.runBasicCommand(ctx, "prompt-response/"+resp.ID, resp.Responses)
	return err
}

func (g *GPTScript) runBasicCommand(ctx context.Context, requestPath string, body any) (string, error) {
	run := &Run{
		url:          g.url,
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
