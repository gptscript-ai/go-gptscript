package gptscript

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

// Opts represents options for the gptscript tool or file.
type Opts struct {
	DisableCache bool   `json:"disableCache"`
	CacheDir     string `json:"cacheDir"`
	Quiet        bool   `json:"quiet"`
	Chdir        string `json:"chdir"`
	SubTool      string `json:"subTool"`
}

func (o Opts) toArgs() []string {
	var args []string
	if o.DisableCache {
		args = append(args, "--disable-cache")
	}
	if o.CacheDir != "" {
		args = append(args, "--cache-dir="+o.CacheDir)
	}
	if o.Chdir != "" {
		args = append(args, "--chdir="+o.Chdir)
	}
	if o.SubTool != "" {
		args = append(args, "--sub-tool="+o.SubTool)
	}
	return append(args, "--quiet="+fmt.Sprint(o.Quiet))
}

// Version will return the output of `gptscript --version`
func Version(ctx context.Context) (string, error) {
	out, err := exec.CommandContext(ctx, getCommand(), "--version").CombinedOutput()
	return string(bytes.TrimSpace(out)), err
}

// ListTools will list all the available tools.
func ListTools(ctx context.Context) (string, error) {
	out, err := exec.CommandContext(ctx, getCommand(), "--list-tools").CombinedOutput()
	return string(out), err
}

// ListModels will list all the available models.
func ListModels(ctx context.Context) ([]string, error) {
	out, err := exec.CommandContext(ctx, getCommand(), "--list-models").CombinedOutput()
	if err != nil {
		return nil, err
	}
	return strings.Split(strings.TrimSpace(string(out)), "\n"), nil
}

// ExecTool will execute a tool. The tool must be a fmt.Stringer, and the string should be a valid gptscript file.
func ExecTool(ctx context.Context, opts Opts, tools ...fmt.Stringer) (string, error) {
	c := exec.CommandContext(ctx, getCommand(), append(opts.toArgs(), "-")...)
	c.Stdin = strings.NewReader(concatTools(tools))

	stdout, err := c.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("failed to get stdout pipe: %w", err)
	}

	stderr, err := c.StderrPipe()
	if err != nil {
		return "", fmt.Errorf("failed to get stderr pipe: %w", err)
	}

	if err = c.Start(); err != nil {
		return "", fmt.Errorf("failed to start command: %w", err)
	}

	stdErr, err := io.ReadAll(stderr)
	if err != nil {
		return "", fmt.Errorf("failed to read stderr: %w", err)
	}

	stdOut, err := io.ReadAll(stdout)
	if err != nil {
		return "", fmt.Errorf("failed to read stdout: %w", err)
	}

	if err = c.Wait(); err != nil {
		return "", fmt.Errorf("failed to wait for command, stderr: %s: %w", stdErr, err)
	}

	return string(stdOut), err
}

// StreamExecTool will execute a tool. The tool must be a fmt.Stringer, and the string should be a valid gptscript file.
// This returns two io.ReadClosers, one for stdout and one for stderr, and a function to wait for the process to exit.
// Reading from stdOut and stdErr should be completed before calling the wait function.
func StreamExecTool(ctx context.Context, opts Opts, tools ...fmt.Stringer) (io.Reader, io.Reader, func() error) {
	c, stdout, stderr, err := setupForkCommand(ctx, "", append(opts.toArgs(), "-"))
	if err != nil {
		return stdout, stderr, func() error { return err }
	}

	c.Stdin = strings.NewReader(concatTools(tools))

	if err = c.Start(); err != nil {
		return stdout, stderr, func() error { return err }
	}

	return stdout, stderr, c.Wait
}

// StreamExecToolWithEvents will execute a tool. The tool must be a fmt.Stringer, and the string should be a valid gptscript file.
// This returns three io.ReadClosers, one for stdout, one for stderr, and one for events, and a function to wait for the process to exit.
// Reading from stdOut, stdErr, and events should be completed before calling the wait function.
func StreamExecToolWithEvents(ctx context.Context, opts Opts, tools ...fmt.Stringer) (io.Reader, io.Reader, io.Reader, func() error) {
	eventsRead, eventsWrite, err := os.Pipe()
	if err != nil {
		return new(reader), new(reader), new(reader), func() error { return err }
	}
	// Close the parent pipe after starting the child process
	defer eventsWrite.Close()

	c, stdout, stderr, err := setupForkCommand(ctx, "", append(opts.toArgs(), "-"))
	if err != nil {
		_ = eventsRead.Close()
		return stdout, stderr, new(reader), func() error { return err }
	}

	c.Stdin = strings.NewReader(concatTools(tools))

	appendExtraFiles(c, eventsWrite)

	if err = c.Start(); err != nil {
		_ = eventsRead.Close()
		return stdout, stderr, new(reader), func() error { return err }
	}

	wait := func() error {
		err := c.Wait()
		_ = eventsRead.Close()
		return err
	}
	return stdout, stderr, eventsRead, wait
}

// ExecFile will execute the file at the given path with the given input.
// The file at the path should be a valid gptscript file.
// The input should be command line arguments in the form of a string (i.e. "--arg1 value1 --arg2 value2").
func ExecFile(ctx context.Context, toolPath, input string, opts Opts) (string, error) {
	args := append(opts.toArgs(), toolPath)
	if input != "" {
		args = append(args, input)
	}

	c := exec.CommandContext(ctx, getCommand(), args...)

	stdout, err := c.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("failed to get stdout pipe: %w", err)
	}

	stderr, err := c.StderrPipe()
	if err != nil {
		return "", fmt.Errorf("failed to get stderr pipe: %w", err)
	}

	if err = c.Start(); err != nil {
		return "", fmt.Errorf("failed to start command: %w", err)
	}

	stdErr, err := io.ReadAll(stderr)
	if err != nil {
		return "", fmt.Errorf("failed to read stderr: %w", err)
	}

	stdOut, err := io.ReadAll(stdout)
	if err != nil {
		return "", fmt.Errorf("failed to read stdout: %w", err)
	}

	if err = c.Wait(); err != nil {
		return "", fmt.Errorf("failed to wait for command, stderr: %s: %w", stdErr, err)
	}

	return string(stdOut), err
}

// StreamExecFile will execute the file at the given path with the given input.
// The file at the path should be a valid gptscript file.
// The input should be command line arguments in the form of a string (i.e. "--arg1 value1 --arg2 value2").
// This returns two io.ReadClosers, one for stdout and one for stderr, and a function to wait for the process to exit.
// Reading from stdOut and stdErr should be completed before calling the wait function.
func StreamExecFile(ctx context.Context, toolPath, input string, opts Opts) (io.Reader, io.Reader, func() error) {
	args := append(opts.toArgs(), toolPath)
	c, stdout, stderr, err := setupForkCommand(ctx, input, args)
	if err != nil {
		return stdout, stderr, func() error { return err }
	}

	if err = c.Start(); err != nil {
		return stdout, stderr, func() error { return err }
	}

	return stdout, stderr, c.Wait
}

// StreamExecFileWithEvents will execute the file at the given path with the given input.
// The file at the path should be a valid gptscript file.
// The input should be command line arguments in the form of a string (i.e. "--arg1 value1 --arg2 value2").
// This returns three io.ReadClosers, one for stdout, one for stderr, and one for events, and a function to wait for the process to exit.
// Reading from stdOut, stdErr, and events should be completed before calling the wait function.
func StreamExecFileWithEvents(ctx context.Context, toolPath, input string, opts Opts) (io.Reader, io.Reader, io.Reader, func() error) {
	eventsRead, eventsWrite, err := os.Pipe()
	if err != nil {
		return new(reader), new(reader), new(reader), func() error { return err }
	}
	// Close the parent pipe after starting the child process
	defer eventsWrite.Close()

	args := append(opts.toArgs(), toolPath)

	c, stdout, stderr, err := setupForkCommand(ctx, input, args)
	if err != nil {
		_ = eventsRead.Close()
		return stdout, stderr, new(reader), func() error { return err }
	}

	appendExtraFiles(c, eventsWrite)

	if err = c.Start(); err != nil {
		_ = eventsRead.Close()
		return stdout, stderr, new(reader), func() error { return err }
	}

	wait := func() error {
		err := c.Wait()
		_ = eventsRead.Close()
		return err
	}

	return stdout, stderr, eventsRead, wait
}

// Parse will parse the given file into an array of Nodes.
func Parse(ctx context.Context, fileName string, opts Opts) ([]Node, error) {
	output, err := exec.CommandContext(ctx, getCommand(), append(opts.toArgs(), "parse", fileName)...).CombinedOutput()
	if err != nil {
		return nil, err
	}

	var doc Document
	if err = json.Unmarshal(output, &doc); err != nil {
		return nil, err
	}

	return doc.Nodes, nil
}

// ParseTool will parse the given string into a tool.
func ParseTool(ctx context.Context, input string) ([]Node, error) {
	c := exec.CommandContext(ctx, getCommand(), "parse", "-")
	c.Stdin = strings.NewReader(input)

	output, err := c.CombinedOutput()
	if err != nil {
		return nil, err
	}

	var doc Document
	if err = json.Unmarshal(output, &doc); err != nil {
		return nil, err
	}

	return doc.Nodes, nil
}

// Fmt will format the given nodes into a string.
func Fmt(ctx context.Context, nodes []Node) (string, error) {
	b, err := json.Marshal(Document{Nodes: nodes})
	if err != nil {
		return "", fmt.Errorf("failed to marshal nodes: %w", err)
	}

	c := exec.CommandContext(ctx, getCommand(), "fmt", "-")
	c.Stdin = bytes.NewReader(b)

	output, err := c.CombinedOutput()
	if err != nil {
		return "", err
	}

	return string(output), nil
}

func concatTools(tools []fmt.Stringer) string {
	var sb strings.Builder
	for i, tool := range tools {
		sb.WriteString(tool.String())
		if i < len(tools)-1 {
			sb.WriteString("\n---\n")
		}
	}
	return sb.String()
}

func getCommand() string {
	if gptScriptBin := os.Getenv("GPTSCRIPT_BIN"); gptScriptBin != "" {
		return gptScriptBin
	}

	return "gptscript"
}

func setupForkCommand(ctx context.Context, input string, args []string) (*exec.Cmd, io.Reader, io.Reader, error) {
	if input != "" {
		args = append(args, input)
	}

	c := exec.CommandContext(ctx, getCommand(), args...)

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
