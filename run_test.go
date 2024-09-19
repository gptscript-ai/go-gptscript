package gptscript

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"os"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRestartingErrorRun(t *testing.T) {
	instructions := "#!/bin/bash\nexit ${EXIT_CODE}"
	if runtime.GOOS == "windows" {
		instructions = "#!/usr/bin/env powershell.exe\n\n$e = $env:EXIT_CODE;\nif ($e) { Exit 1; }"
	}
	tool := ToolDef{
		Context:      []string{"my-context"},
		Instructions: "Say hello",
	}
	contextTool := ToolDef{
		Name:         "my-context",
		Instructions: instructions,
	}

	run, err := g.Evaluate(context.Background(), Options{GlobalOptions: GlobalOptions{Env: []string{"EXIT_CODE=1"}}, IncludeEvents: true}, tool, contextTool)
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

func TestStackedContexts(t *testing.T) {
	const name = "testcred"

	wd, err := os.Getwd()
	require.NoError(t, err)

	bytes := make([]byte, 32)
	_, err = rand.Read(bytes)
	require.NoError(t, err)

	context1 := hex.EncodeToString(bytes)[:16]
	context2 := hex.EncodeToString(bytes)[16:]

	run, err := g.Run(context.Background(), wd+"/test/credential.gpt", Options{
		CredentialContexts: []string{context1, context2},
	})
	require.NoError(t, err)

	_, err = run.Text()
	require.NoError(t, err)

	// The credential should exist in context1 now.
	cred, err := g.RevealCredential(context.Background(), []string{context1, context2}, name)
	require.NoError(t, err)
	require.Equal(t, cred.Context, context1)

	// Now change the context order and run the script again.
	run, err = g.Run(context.Background(), wd+"/test/credential.gpt", Options{
		CredentialContexts: []string{context2, context1},
	})
	require.NoError(t, err)

	_, err = run.Text()
	require.NoError(t, err)

	// Now make sure the credential exists in context1 still.
	cred, err = g.RevealCredential(context.Background(), []string{context2, context1}, name)
	require.NoError(t, err)
	require.Equal(t, cred.Context, context1)
}
