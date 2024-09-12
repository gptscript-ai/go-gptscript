package gptscript

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
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
	url        string
	globalOpts GlobalOptions
}

func NewGPTScript(opts ...GlobalOptions) (*GPTScript, error) {
	opt := completeGlobalOptions(opts...)
	lock.Lock()
	defer lock.Unlock()
	gptscriptCount++

	disableServer := os.Getenv("GPTSCRIPT_DISABLE_SERVER") == "true"

	if serverURL == "" {
		serverURL = os.Getenv("GPTSCRIPT_URL")
	}

	if opt.Env == nil {
		opt.Env = os.Environ()
	}

	opt.Env = append(opt.Env, opt.toEnv()...)

	if serverProcessCancel == nil && !disableServer {
		ctx, cancel := context.WithCancel(context.Background())
		in, _ := io.Pipe()

		serverProcess = exec.CommandContext(ctx, getCommand(), "sys.sdkserver", "--listen-address", serverURL)
		serverProcess.Env = opt.Env[:]

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
		url:        "http://" + serverURL,
		globalOpts: opt,
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
	opts.GlobalOptions = completeGlobalOptions(g.globalOpts, opts.GlobalOptions)
	return (&Run{
		url:         g.url,
		requestPath: "evaluate",
		state:       Creating,
		opts:        opts,
		tools:       tools,
	}).NextChat(ctx, opts.Input)
}

func (g *GPTScript) Run(ctx context.Context, toolPath string, opts Options) (*Run, error) {
	opts.GlobalOptions = completeGlobalOptions(g.globalOpts, opts.GlobalOptions)
	return (&Run{
		url:         g.url,
		requestPath: "run",
		state:       Creating,
		opts:        opts,
		toolPath:    toolPath,
	}).NextChat(ctx, opts.Input)
}

type ParseOptions struct {
	DisableCache bool
}

// Parse will parse the given file into an array of Nodes.
func (g *GPTScript) Parse(ctx context.Context, fileName string, opts ...ParseOptions) ([]Node, error) {
	var disableCache bool
	for _, opt := range opts {
		disableCache = disableCache || opt.DisableCache
	}

	out, err := g.runBasicCommand(ctx, "parse", map[string]any{"file": fileName, "disableCache": disableCache})
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

// ParseContent will parse the given string into a tool.
func (g *GPTScript) ParseContent(ctx context.Context, toolDef string) ([]Node, error) {
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

type LoadOptions struct {
	DisableCache bool
	SubTool      string
}

// LoadFile will load the given file into a Program.
func (g *GPTScript) LoadFile(ctx context.Context, fileName string, opts ...LoadOptions) (*Program, error) {
	return g.load(ctx, map[string]any{"file": fileName}, opts...)
}

// LoadContent will load the given content into a Program.
func (g *GPTScript) LoadContent(ctx context.Context, content string, opts ...LoadOptions) (*Program, error) {
	return g.load(ctx, map[string]any{"content": content}, opts...)
}

// LoadTools will load the given tools into a Program.
func (g *GPTScript) LoadTools(ctx context.Context, toolDefs []ToolDef, opts ...LoadOptions) (*Program, error) {
	return g.load(ctx, map[string]any{"toolDefs": toolDefs}, opts...)
}

func (g *GPTScript) load(ctx context.Context, payload map[string]any, opts ...LoadOptions) (*Program, error) {
	for _, opt := range opts {
		if opt.DisableCache {
			payload["disableCache"] = true
		}
		if opt.SubTool != "" {
			payload["subTool"] = opt.SubTool
		}
	}

	out, err := g.runBasicCommand(ctx, "load", payload)
	if err != nil {
		return nil, err
	}

	type loadResponse struct {
		Program *Program `json:"program"`
	}

	prg := new(loadResponse)
	if err = json.Unmarshal([]byte(out), prg); err != nil {
		return nil, err
	}

	return prg.Program, nil
}

// Version will return the output of `gptscript --version`
func (g *GPTScript) Version(ctx context.Context) (string, error) {
	out, err := g.runBasicCommand(ctx, "version", nil)
	if err != nil {
		return "", err
	}

	return out, nil
}

type ListModelsOptions struct {
	Providers           []string
	CredentialOverrides []string
}

// ListModels will list all the available models.
func (g *GPTScript) ListModels(ctx context.Context, opts ...ListModelsOptions) ([]string, error) {
	var o ListModelsOptions
	for _, opt := range opts {
		o.Providers = append(o.Providers, opt.Providers...)
		o.CredentialOverrides = append(o.CredentialOverrides, opt.CredentialOverrides...)
	}

	if g.globalOpts.DefaultModelProvider != "" {
		o.Providers = append(o.Providers, g.globalOpts.DefaultModelProvider)
	}

	out, err := g.runBasicCommand(ctx, "list-models", map[string]any{
		"providers":           o.Providers,
		"env":                 g.globalOpts.Env,
		"credentialOverrides": o.CredentialOverrides,
	})
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

func (g *GPTScript) ListCredentials(ctx context.Context, credCtx string, allContexts bool) ([]Credential, error) {
	req := CredentialRequest{}
	if allContexts {
		req.AllContexts = true
	} else {
		req.Context = credCtx
	}

	out, err := g.runBasicCommand(ctx, "credentials", req)
	if err != nil {
		return nil, err
	}

	var creds []Credential
	if err = json.Unmarshal([]byte(out), &creds); err != nil {
		return nil, err
	}
	return creds, nil
}

func (g *GPTScript) CreateCredential(ctx context.Context, cred Credential) error {
	credJson, err := json.Marshal(cred)
	if err != nil {
		return fmt.Errorf("failed to marshal credential: %w", err)
	}

	_, err = g.runBasicCommand(ctx, "credentials/create", CredentialRequest{Content: string(credJson)})
	return err
}

func (g *GPTScript) RevealCredential(ctx context.Context, credCtx, name string) (Credential, error) {
	out, err := g.runBasicCommand(ctx, "credentials/reveal", CredentialRequest{
		Context: credCtx,
		Name:    name,
	})
	if err != nil {
		return Credential{}, err
	}

	var cred Credential
	if err = json.Unmarshal([]byte(out), &cred); err != nil {
		return Credential{}, err
	}
	return cred, nil
}

func (g *GPTScript) DeleteCredential(ctx context.Context, credCtx, name string) error {
	_, err := g.runBasicCommand(ctx, "credentials/delete", CredentialRequest{
		Context: credCtx,
		Name:    name,
	})
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

func GetEnv(key, def string) string {
	v := os.Getenv(key)
	if v == "" {
		return def
	}

	if strings.HasPrefix(v, `{"_gz":"`) && strings.HasSuffix(v, `"}`) {
		data, err := base64.StdEncoding.DecodeString(v[8 : len(v)-2])
		if err != nil {
			return v
		}
		gz, err := gzip.NewReader(bytes.NewBuffer(data))
		if err != nil {
			return v
		}
		strBytes, err := io.ReadAll(gz)
		if err != nil {
			return v
		}
		return string(strBytes)
	}

	return v
}
