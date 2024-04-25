package gptscript

import (
	"context"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"testing"
)

func TestMain(m *testing.M) {
	if os.Getenv("OPENAI_API_KEY") == "" {
		panic("OPENAI_API_KEY not set")
	}

	os.Exit(m.Run())
}

func TestVersion(t *testing.T) {
	out, err := Version(context.Background())
	if err != nil {
		t.Errorf("Error getting version: %v", err)
	}

	if !strings.HasPrefix(out, "gptscript version") {
		t.Errorf("Unexpected output: %s", out)
	}
}

func TestListTools(t *testing.T) {
	tools, err := ListTools(context.Background())
	if err != nil {
		t.Errorf("Error listing tools: %v", err)
	}

	if len(tools) == 0 {
		t.Error("No tools found")
	}
}

func TestListModels(t *testing.T) {
	models, err := ListModels(context.Background())
	if err != nil {
		t.Errorf("Error listing models: %v", err)
	}

	if len(models) == 0 {
		t.Error("No models found")
	}
}

func TestSimpleExec(t *testing.T) {
	tool := &FreeForm{Content: "What is the capital of the united states?"}

	out, err := ExecTool(context.Background(), Opts{}, tool)
	if err != nil {
		t.Errorf("Error executing tool: %v", err)
	}

	if !strings.Contains(out, "Washington") {
		t.Errorf("Unexpected output: %s", out)
	}
}

func TestExecFileChdir(t *testing.T) {
	// By changing the directory here, we should be able to find the test.gpt file without `./test` (see TestStreamExecFile)
	out, err := ExecFile(context.Background(), "test.gpt", "", Opts{Chdir: "./test"})
	if err != nil {
		t.Errorf("Error executing tool: %v", err)
	}

	if out == "" {
		t.Error("No output from tool")
	}
}

func TestExecComplexTool(t *testing.T) {
	tool := &Tool{
		JSONResponse: true,
		Instructions: `
Create three short graphic artist descriptions and their muses.
These should be descriptive and explain their point of view.
Also come up with a made up name, they each should be from different
backgrounds and approach art differently.
the response should be in JSON and match the format:
{
	artists: [{
		name: "name"
		description: "description"
	}]
}
`,
	}

	out, err := ExecTool(context.Background(), Opts{}, tool)
	if err != nil {
		t.Errorf("Error executing tool: %v", err)
	}

	if !strings.Contains(out, "\"artists\":") {
		t.Errorf("Unexpected output: %s", out)
	}
}

func TestExecWithToolList(t *testing.T) {
	shebang := "#!/bin/bash"
	if runtime.GOOS == "windows" {
		shebang = "#!/usr/bin/env powershell.exe"
	}
	tools := []fmt.Stringer{
		&Tool{
			Tools:        []string{"echo"},
			Instructions: "echo hello there",
		},
		&Tool{
			Name:        "echo",
			Tools:       []string{"sys.exec"},
			Description: "Echoes the input",
			Args: map[string]string{
				"input": "The string input to echo",
			},
			Instructions: shebang + "\n echo ${input}",
		},
	}

	out, err := ExecTool(context.Background(), Opts{}, tools...)
	if err != nil {
		t.Errorf("Error executing tool: %v", err)
	}

	if !strings.Contains(out, "hello there") {
		t.Errorf("Unexpected output: %s", out)
	}
}

func TestStreamExec(t *testing.T) {
	tool := &FreeForm{Content: "What is the capital of the united states?"}

	stdout, stderr, wait := StreamExecTool(context.Background(), Opts{}, tool)

	stdOut, err := io.ReadAll(stdout)
	if err != nil {
		t.Errorf("Error reading stdout: %v", err)
	}

	stdErr, err := io.ReadAll(stderr)
	if err != nil {
		t.Errorf("Error reading stderr: %v", err)
	}

	if err = wait(); err != nil {
		t.Errorf("Error waiting for process: %v", err)
	}

	if !strings.Contains(string(stdOut), "Washington") {
		t.Errorf("Unexpected output: %s", string(stdOut))
	}

	if len(stdErr) == 0 {
		t.Error("No stderr output")
	}
}

func TestStreamExecFile(t *testing.T) {
	stdout, stderr, wait := StreamExecFile(context.Background(), "./test/test.gpt", "", Opts{})

	stdOut, err := io.ReadAll(stdout)
	if err != nil {
		t.Errorf("Error reading stdout: %v", err)
	}

	stdErr, err := io.ReadAll(stderr)
	if err != nil {
		t.Errorf("Error reading stderr: %v", err)
	}

	if err = wait(); err != nil {
		t.Errorf("Error waiting for process: %v", err)
	}

	if len(stdOut) == 0 {
		t.Error("No stdout output")
	}
	if len(stdErr) == 0 {
		t.Error("No stderr output")
	}
}

func TestStreamExecToolWithEvents(t *testing.T) {
	tool := &FreeForm{Content: "What is the capital of the united states?"}

	stdout, stderr, events, wait := StreamExecToolWithEvents(context.Background(), Opts{}, tool)

	stdOut, err := io.ReadAll(stdout)
	if err != nil {
		t.Errorf("Error reading stdout: %v", err)
	}

	stdErr, err := io.ReadAll(stderr)
	if err != nil {
		t.Errorf("Error reading stderr: %v", err)
	}

	eventsOut, err := io.ReadAll(events)
	if err != nil {
		t.Errorf("Error reading events: %v", err)
	}

	if err = wait(); err != nil {
		t.Errorf("Error waiting for process: %v", err)
	}

	if !strings.Contains(string(stdOut), "Washington") {
		t.Errorf("Unexpected output: %s", string(stdOut))
	}

	if len(stdErr) == 0 {
		t.Error("No stderr output")
	}

	if len(eventsOut) == 0 {
		t.Error("No events output")
	}
}

func TestStreamExecFileWithEvents(t *testing.T) {
	stdout, stderr, events, wait := StreamExecFileWithEvents(context.Background(), "./test/test.gpt", "", Opts{})

	stdOut, err := io.ReadAll(stdout)
	if err != nil {
		t.Errorf("Error reading stdout: %v", err)
	}

	stdErr, err := io.ReadAll(stderr)
	if err != nil {
		t.Errorf("Error reading stderr: %v", err)
	}

	eventsOut, err := io.ReadAll(events)
	if err != nil {
		t.Errorf("Error reading events: %v", err)
	}

	if err = wait(); err != nil {
		fmt.Println(string(stdErr))
		t.Errorf("Error waiting for process: %v", err)
	}

	if len(stdOut) == 0 {
		t.Error("No stdout output")
	}

	if len(stdErr) == 0 {
		t.Error("No stderr output")
	}

	if len(eventsOut) == 0 {
		t.Error("No events output")
	}
}
