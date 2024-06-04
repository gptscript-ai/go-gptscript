package gptscript

import (
	"context"
	"runtime"
	"testing"
)

func TestRestartingErrorRun(t *testing.T) {
	instructions := "#!/bin/bash\nexit ${EXIT_CODE}"
	if runtime.GOOS == "windows" {
		instructions = "#!/usr/bin/env powershell.exe\n\n$e = $env:EXIT_CODE;\nif ($e) { Exit 1; }"
	}
	tool := &ToolDef{
		Context:      []string{"my-context"},
		Instructions: "Say hello",
	}
	contextTool := &ToolDef{
		Name:         "my-context",
		Instructions: instructions,
	}

	run, err := g.Evaluate(context.Background(), Options{Env: []string{"EXIT_CODE=1"}, IncludeEvents: true}, tool, contextTool)
	if err != nil {
		t.Errorf("Error executing tool: %v", err)
	}

	// Wait for the run to complete
	_, err = run.Text()
	if err == nil {
		t.Fatalf("no error returned from run")
	}

	run.opts.Env = nil
	run, err = run.NextChat(context.Background(), "")
	if err != nil {
		t.Errorf("Error executing next run: %v", err)
	}

	_, err = run.Text()
	if err != nil {
		t.Errorf("executing run with input of 0 should not fail: %v", err)
	}
}
