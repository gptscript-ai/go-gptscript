package gptscript

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
)

var g *GPTScript

func TestMain(m *testing.M) {
	if os.Getenv("OPENAI_API_KEY") == "" && os.Getenv("GPTSCRIPT_URL") == "" {
		panic("OPENAI_API_KEY or GPTSCRIPT_URL environment variable must be set")
	}

	var err error
	g, err = NewGPTScript(GlobalOptions{OpenAIAPIKey: os.Getenv("OPENAI_API_KEY")})
	if err != nil {
		panic(fmt.Sprintf("error creating gptscript: %s", err))
	}

	exitCode := m.Run()
	g.Close()
	os.Exit(exitCode)
}

func TestCreateAnotherGPTScript(t *testing.T) {
	g, err := NewGPTScript(GlobalOptions{})
	if err != nil {
		t.Errorf("error creating gptscript: %s", err)
	}
	defer g.Close()

	version, err := g.Version(context.Background())
	if err != nil {
		t.Errorf("error getting version from second gptscript: %s", err)
	}

	if !strings.Contains(version, "gptscript version") {
		t.Errorf("unexpected gptscript version: %s", version)
	}
}

func TestVersion(t *testing.T) {
	out, err := g.Version(context.Background())
	if err != nil {
		t.Errorf("Error getting version: %v", err)
	}

	if !strings.HasPrefix(out, "gptscript version") {
		t.Errorf("Unexpected output: %s", out)
	}
}

func TestListTools(t *testing.T) {
	tools, err := g.ListTools(context.Background())
	if err != nil {
		t.Errorf("Error listing tools: %v", err)
	}

	if len(tools) == 0 {
		t.Error("No tools found")
	}
}

func TestListModels(t *testing.T) {
	models, err := g.ListModels(context.Background())
	if err != nil {
		t.Errorf("Error listing models: %v", err)
	}

	if len(models) == 0 {
		t.Error("No models found")
	}
}

func TestAbortRun(t *testing.T) {
	tool := ToolDef{Instructions: "What is the capital of the united states?"}

	run, err := g.Evaluate(context.Background(), Options{DisableCache: true, IncludeEvents: true}, tool)
	if err != nil {
		t.Errorf("Error executing tool: %v", err)
	}

	// Abort the run after the first event.
	<-run.Events()

	if err := run.Close(); err != nil {
		t.Errorf("Error aborting run: %v", err)
	}

	if run.State() != Error {
		t.Errorf("Unexpected run state: %s", run.State())
	}

	if run.Err() == nil {
		t.Error("Expected error but got nil")
	}
}

func TestSimpleEvaluate(t *testing.T) {
	tool := ToolDef{Instructions: "What is the capital of the united states?"}

	run, err := g.Evaluate(context.Background(), Options{
		GlobalOptions: GlobalOptions{},
	}, tool)
	if err != nil {
		t.Errorf("Error executing tool: %v", err)
	}

	out, err := run.Text()
	if err != nil {
		t.Errorf("Error reading output: %v", err)
	}

	if !strings.Contains(out, "Washington") {
		t.Errorf("Unexpected output: %s", out)
	}

	// This should be able to be called multiple times and produce the same answer.
	out, err = run.Text()
	if err != nil {
		t.Errorf("Error reading output: %v", err)
	}

	if !strings.Contains(out, "Washington") {
		t.Errorf("Unexpected output: %s", out)
	}

	if run.Program() == nil {
		t.Error("Run program not set")
	}
}

func TestEvaluateWithContext(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Error getting current working directory: %v", err)
	}

	tool := ToolDef{
		Instructions: "What is the capital of the united states?",
		Context: []string{
			wd + "/test/acorn-labs-context.gpt",
		},
	}

	run, err := g.Evaluate(context.Background(), Options{}, tool)
	if err != nil {
		t.Errorf("Error executing tool: %v", err)
	}

	out, err := run.Text()
	if err != nil {
		t.Errorf("Error reading output: %v", err)
	}

	if out != "Acorn Labs" {
		t.Errorf("Unexpected output: %s", out)
	}
}

func TestEvaluateComplexTool(t *testing.T) {
	tool := ToolDef{
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

	run, err := g.Evaluate(context.Background(), Options{DisableCache: true}, tool)
	if err != nil {
		t.Errorf("Error executing tool: %v", err)
	}

	out, err := run.Text()
	if err != nil {
		t.Errorf("Error reading output: %v", err)
	}

	if !strings.Contains(out, "\"artists\":") {
		t.Errorf("Unexpected output: %s", out)
	}
}

func TestEvaluateWithToolList(t *testing.T) {
	shebang := "#!/bin/bash"
	if runtime.GOOS == "windows" {
		shebang = "#!/usr/bin/env powershell.exe"
	}
	tools := []ToolDef{
		{
			Tools:        []string{"echo"},
			Instructions: "echo hello there",
		},
		{
			Name:         "echo",
			Tools:        []string{"sys.exec"},
			Description:  "Echoes the input",
			Arguments:    ObjectSchema("input", "The string input to echo"),
			Instructions: shebang + "\necho ${input}",
		},
	}

	run, err := g.Evaluate(context.Background(), Options{}, tools...)
	if err != nil {
		t.Errorf("Error executing tool: %v", err)
	}

	out, err := run.Text()
	if err != nil {
		t.Errorf("Error reading output: %v", err)
	}

	if !strings.Contains(out, "hello there") {
		t.Errorf("Unexpected output: %s", out)
	}
}

func TestEvaluateWithToolListAndSubTool(t *testing.T) {
	shebang := "#!/bin/bash"
	if runtime.GOOS == "windows" {
		shebang = "#!/usr/bin/env powershell.exe"
	}
	tools := []ToolDef{
		{
			Tools:        []string{"echo"},
			Instructions: "echo 'hello there'",
		},
		{
			Name:         "other",
			Tools:        []string{"echo"},
			Instructions: "echo 'hello somewhere else'",
		},
		{
			Name:         "echo",
			Tools:        []string{"sys.exec"},
			Description:  "Echoes the input",
			Arguments:    ObjectSchema("input", "The string input to echo"),
			Instructions: shebang + "\n echo ${input}",
		},
	}

	run, err := g.Evaluate(context.Background(), Options{SubTool: "other"}, tools...)
	if err != nil {
		t.Errorf("Error executing tool: %v", err)
	}

	out, err := run.Text()
	if err != nil {
		t.Errorf("Error reading output: %v", err)
	}

	if !strings.Contains(out, "hello somewhere else") {
		t.Errorf("Unexpected output: %s", out)
	}
}

func TestStreamEvaluate(t *testing.T) {
	var eventContent string
	tool := ToolDef{Instructions: "What is the capital of the united states?"}

	run, err := g.Evaluate(context.Background(), Options{IncludeEvents: true}, tool)
	if err != nil {
		t.Fatalf("Error executing tool: %v", err)
	}

	for e := range run.Events() {
		if e.Call != nil {
			for _, o := range e.Call.Output {
				eventContent += o.Content
			}
		}
	}

	out, err := run.Text()
	if err != nil {
		t.Errorf("Error reading output: %v", err)
	}

	if !strings.Contains(eventContent, "Washington") {
		t.Errorf("Unexpected event output: %s", eventContent)
	}

	if !strings.Contains(out, "Washington") {
		t.Errorf("Unexpected output: %s", out)
	}

	if len(run.ErrorOutput()) != 0 {
		t.Errorf("Should have no stderr output: %v", run.ErrorOutput())
	}
}

func TestStreamRun(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Error getting working directory: %v", err)
	}

	var eventContent string
	run, err := g.Run(context.Background(), wd+"/test/catcher.gpt", Options{IncludeEvents: true})
	if err != nil {
		t.Fatalf("Error executing file: %v", err)
	}

	for e := range run.Events() {
		if e.Call != nil {
			for _, o := range e.Call.Output {
				eventContent += o.Content
			}
		}
	}

	out, err := run.Text()
	if err != nil {
		t.Errorf("Error reading output: %v", err)
	}

	if !strings.Contains(eventContent, "Salinger") {
		t.Errorf("Unexpected event output: %s", eventContent)
	}

	if !strings.Contains(out, "Salinger") {
		t.Errorf("Unexpected output: %s", out)
	}

	if len(run.ErrorOutput()) != 0 {
		t.Error("Should have no stderr output")
	}
}

func TestCredentialOverride(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Error getting working directory: %v", err)
	}

	gptscriptFile := "credential-override.gpt"
	if runtime.GOOS == "windows" {
		gptscriptFile = "credential-override-windows.gpt"
	}

	run, err := g.Run(context.Background(), filepath.Join(wd, "test", gptscriptFile), Options{
		DisableCache: true,
		CredentialOverrides: []string{
			"test.ts.credential_override:TEST_CRED=foo",
		},
	})
	if err != nil {
		t.Fatalf("Error executing file: %v", err)
	}

	out, err := run.Text()
	if err != nil {
		t.Errorf("Error reading output: %v", err)
	}

	if !strings.Contains(out, "foo") {
		t.Errorf("Unexpected output: %s", out)
	}

	if len(run.ErrorOutput()) != 0 {
		t.Error("Should have no stderr output")
	}
}

func TestParseSimpleFile(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Error getting working directory: %v", err)
	}

	tools, err := g.Parse(context.Background(), wd+"/test/test.gpt")
	if err != nil {
		t.Errorf("Error parsing file: %v", err)
	}

	if len(tools) != 1 {
		t.Fatalf("Unexpected number of tools: %d", len(tools))
	}

	if tools[0].ToolNode == nil {
		t.Fatalf("No tool node found")
	}

	if tools[0].ToolNode.Tool.Instructions != "Respond with a hello, in a random language. Also include the language in the response." {
		t.Errorf("Unexpected instructions: %s", tools[0].ToolNode.Tool.Instructions)
	}
}

func TestParseTool(t *testing.T) {
	tools, err := g.ParseTool(context.Background(), "echo hello")
	if err != nil {
		t.Errorf("Error parsing tool: %v", err)
	}

	if len(tools) != 1 {
		t.Fatalf("Unexpected number of tools: %d", len(tools))
	}

	if tools[0].ToolNode == nil {
		t.Fatalf("No tool node found")
	}

	if tools[0].ToolNode.Tool.Instructions != "echo hello" {
		t.Errorf("Unexpected instructions: %s", tools[0].ToolNode.Tool.Instructions)
	}
}

func TestParseToolWithTextNode(t *testing.T) {
	tools, err := g.ParseTool(context.Background(), "echo hello\n---\n!markdown\nhello")
	if err != nil {
		t.Errorf("Error parsing tool: %v", err)
	}

	if len(tools) != 2 {
		t.Fatalf("Unexpected number of tools: %d", len(tools))
	}

	if tools[0].ToolNode == nil {
		t.Fatalf("No tool node found")
	}

	if tools[0].ToolNode.Tool.Instructions != "echo hello" {
		t.Errorf("Unexpected instructions: %s", tools[0].ToolNode.Tool.Instructions)
	}

	if tools[1].TextNode == nil {
		t.Fatalf("No text node found")
	}

	if tools[1].TextNode.Text != "hello\n" {
		t.Errorf("Unexpected text: %s", tools[1].TextNode.Text)
	}
	if tools[1].TextNode.Fmt != "markdown" {
		t.Errorf("Unexpected fmt: %s", tools[1].TextNode.Fmt)
	}
}

func TestFmt(t *testing.T) {
	nodes := []Node{
		{
			ToolNode: &ToolNode{
				Tool: Tool{
					ToolDef: ToolDef{
						Tools:        []string{"echo"},
						Instructions: "echo hello there",
					},
				},
			},
		},
		{
			ToolNode: &ToolNode{
				Tool: Tool{
					ToolDef: ToolDef{
						Name:         "echo",
						Instructions: "#!/bin/bash\necho hello there",
					},
					Arguments: &openapi3.Schema{
						Type: &openapi3.Types{"object"},
						Properties: map[string]*openapi3.SchemaRef{
							"input": {
								Value: &openapi3.Schema{
									Description: "The string input to echo",
									Type:        &openapi3.Types{"string"},
								},
							},
						},
					},
				},
			},
		},
	}

	out, err := g.Fmt(context.Background(), nodes)
	if err != nil {
		t.Errorf("Error formatting nodes: %v", err)
	}

	if out != `Tools: echo

echo hello there

---
Name: echo
Parameter: input: The string input to echo

#!/bin/bash
echo hello there
` {
		t.Errorf("Unexpected output: %s", out)
	}
}

func TestFmtWithTextNode(t *testing.T) {
	nodes := []Node{
		{
			ToolNode: &ToolNode{
				Tool: Tool{
					ToolDef: ToolDef{
						Tools:        []string{"echo"},
						Instructions: "echo hello there",
					},
				},
			},
		},
		{
			TextNode: &TextNode{
				Fmt:  "markdown",
				Text: "We now echo hello there\n",
			},
		},
		{
			ToolNode: &ToolNode{
				Tool: Tool{
					ToolDef: ToolDef{
						Instructions: "#!/bin/bash\necho hello there",
						Name:         "echo",
					},
					Arguments: &openapi3.Schema{
						Type: &openapi3.Types{"object"},
						Properties: map[string]*openapi3.SchemaRef{
							"input": {
								Value: &openapi3.Schema{
									Description: "The string input to echo",
									Type:        &openapi3.Types{"string"},
								},
							},
						},
					},
				},
			},
		},
	}

	out, err := g.Fmt(context.Background(), nodes)
	if err != nil {
		t.Errorf("Error formatting nodes: %v", err)
	}

	if out != `Tools: echo

echo hello there

---
!markdown
We now echo hello there
---
Name: echo
Parameter: input: The string input to echo

#!/bin/bash
echo hello there
` {
		t.Errorf("Unexpected output: %s", out)
	}
}

func TestToolChat(t *testing.T) {
	tool := ToolDef{
		Chat:         true,
		Instructions: "You are a chat bot. Don't finish the conversation until I say 'bye'.",
		Tools:        []string{"sys.chat.finish"},
	}

	run, err := g.Evaluate(context.Background(), Options{DisableCache: true}, tool)
	if err != nil {
		t.Fatalf("Error executing tool: %v", err)
	}
	inputs := []string{
		"List the three largest states in the United States by area.",
		"What is the capital of the third one?",
		"What timezone is the first one in?",
	}

	expectedOutputs := []string{
		"California",
		"Sacramento",
		"Alaska Time Zone",
	}

	// Just wait for the chat to start up.
	_, err = run.Text()
	if err != nil {
		t.Fatalf("Error waiting for initial output: %v", err)
	}

	for i, input := range inputs {
		run, err = run.NextChat(context.Background(), input)
		if err != nil {
			t.Fatalf("Error sending next input %q: %v", input, err)
		}

		out, err := run.Text()
		if err != nil {
			t.Errorf("Error reading output: %s", run.ErrorOutput())
			t.Fatalf("Error reading output: %v", err)
		}

		if !strings.Contains(out, expectedOutputs[i]) {
			t.Fatalf("Unexpected output: %s", out)
		}
	}
}

func TestFileChat(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Error getting current working directory: %v", err)
	}

	run, err := g.Run(context.Background(), wd+"/test/chat.gpt", Options{})
	if err != nil {
		t.Fatalf("Error executing tool: %v", err)
	}
	inputs := []string{
		"List the 3 largest of the Great Lakes by volume.",
		"What is the volume of the second one in cubic miles?",
		"What is the total area of the third one in square miles?",
	}

	expectedOutputs := []string{
		"Lake Superior",
		"Lake Michigan",
		"Lake Huron",
	}

	// Just wait for the chat to start up.
	_, err = run.Text()
	if err != nil {
		t.Fatalf("Error waiting for initial output: %v", err)
	}

	for i, input := range inputs {
		run, err = run.NextChat(context.Background(), input)
		if err != nil {
			t.Fatalf("Error sending next input %q: %v", input, err)
		}

		out, err := run.Text()
		if err != nil {
			t.Errorf("Error reading output: %s", run.ErrorOutput())
			t.Fatalf("Error reading output: %v", err)
		}

		if !strings.Contains(out, expectedOutputs[i]) {
			t.Fatalf("Unexpected output: %s", out)
		}
	}
}

func TestToolWithGlobalTools(t *testing.T) {
	var runStartSeen, callStartSeen, callFinishSeen, callProgressSeen, runFinishSeen bool
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Error getting current working directory: %v", err)
	}

	var eventContent string

	run, err := g.Run(context.Background(), wd+"/test/global-tools.gpt", Options{DisableCache: true, IncludeEvents: true})
	if err != nil {
		t.Fatalf("Error executing tool: %v", err)
	}

	for e := range run.Events() {
		if e.Run != nil {
			if e.Run.Type == EventTypeRunStart {
				runStartSeen = true
			} else if e.Run.Type == EventTypeRunFinish {
				runFinishSeen = true
			}
		} else if e.Call != nil {
			if e.Call.Type == EventTypeCallStart {
				callStartSeen = true
			} else if e.Call.Type == EventTypeCallFinish {
				callFinishSeen = true

				for _, o := range e.Call.Output {
					eventContent += o.Content
				}
			} else if e.Call.Type == EventTypeCallProgress {
				callProgressSeen = true
			}
		}
	}

	out, err := run.Text()
	if err != nil {
		t.Errorf("Error reading output: %v", err)
	}

	if !strings.Contains(eventContent, "Hello") {
		t.Errorf("Unexpected event output: %s", eventContent)
	}

	if !strings.Contains(out, "Hello!") {
		t.Errorf("Unexpected output: %s", out)
	}

	if len(run.ErrorOutput()) != 0 {
		t.Errorf("Should have no stderr output: %v", run.ErrorOutput())
	}

	if !runStartSeen || !callStartSeen || !callFinishSeen || !runFinishSeen || !callProgressSeen {
		t.Errorf("Missing events: %t %t %t %t %t", runStartSeen, callStartSeen, callFinishSeen, runFinishSeen, callProgressSeen)
	}
}

func TestConfirm(t *testing.T) {
	var eventContent string
	tools := ToolDef{
		Instructions: "List all the files in the current directory. Respond with the names of the files in only the current directory.",
		Tools:        []string{"sys.exec"},
	}

	run, err := g.Evaluate(context.Background(), Options{IncludeEvents: true, Confirm: true}, tools)
	if err != nil {
		t.Errorf("Error executing tool: %v", err)
	}

	var confirmCallEvent *CallFrame
	done := make(chan struct{})
	go func() {
		defer close(done)

		for e := range run.Events() {
			if e.Call != nil {
				for _, o := range e.Call.Output {
					eventContent += o.Content
				}

				if e.Call.Type == EventTypeCallConfirm {
					confirmCallEvent = e.Call

					if !strings.Contains(confirmCallEvent.Input, "\"ls") && !strings.Contains(confirmCallEvent.Input, "\"dir") {
						t.Errorf("unexpected confirm input: %s", confirmCallEvent.Input)
					}

					// Confirm the call
					if err = g.Confirm(context.Background(), AuthResponse{
						ID:     confirmCallEvent.ID,
						Accept: true,
					}); err != nil {
						t.Errorf("Error confirming: %v", err)
					}
				}
			}
		}
	}()

	out, err := run.Text()
	if err != nil {
		t.Errorf("Error reading output: %v", err)
	}

	// Wait for events processing to finish
	<-done

	if confirmCallEvent == nil {
		t.Fatalf("No confirm call event")
	}

	if !strings.Contains(eventContent, "Makefile") || !strings.Contains(eventContent, "README.md") {
		t.Errorf("Unexpected event output: %s", eventContent)
	}

	if !strings.Contains(out, "Makefile") || !strings.Contains(out, "README.md") {
		t.Errorf("Unexpected output: %s", out)
	}

	if len(run.ErrorOutput()) != 0 {
		t.Errorf("Should have no stderr output: %v", run.ErrorOutput())
	}
}

func TestConfirmDeny(t *testing.T) {
	var eventContent string
	tools := ToolDef{
		Instructions: "List the files in the current directory as '.'. If that doesn't work print the word FAIL.",
		Tools:        []string{"sys.exec"},
	}

	run, err := g.Evaluate(context.Background(), Options{IncludeEvents: true, Confirm: true}, tools)
	if err != nil {
		t.Errorf("Error executing tool: %v", err)
	}

	// Wait for the confirm event
	var confirmCallEvent *CallFrame
	for e := range run.Events() {
		if e.Call != nil {
			for _, o := range e.Call.Output {
				eventContent += o.Content
			}

			if e.Call.Type == EventTypeCallConfirm {
				confirmCallEvent = e.Call
				break
			}
		}
	}

	if confirmCallEvent == nil {
		t.Fatalf("No confirm call event")
	}

	if !strings.Contains(confirmCallEvent.Input, "\"ls\"") {
		t.Errorf("unexpected confirm input: %s", confirmCallEvent.Input)
	}

	if err = g.Confirm(context.Background(), AuthResponse{
		ID:      confirmCallEvent.ID,
		Accept:  false,
		Message: "I will not allow it!",
	}); err != nil {
		t.Errorf("Error confirming: %v", err)
	}

	// Read the remainder of the events
	for e := range run.Events() {
		if e.Call != nil {
			for _, o := range e.Call.Output {
				eventContent += o.Content
			}
		}
	}

	out, err := run.Text()
	if err != nil {
		t.Errorf("Error reading output: %v", err)
	}

	if !strings.Contains(strings.ToLower(eventContent), "fail") {
		t.Errorf("Unexpected event output: %s", eventContent)
	}

	if !strings.Contains(strings.ToLower(out), "fail") {
		t.Errorf("Unexpected output: %s", out)
	}

	if len(run.ErrorOutput()) != 0 {
		t.Errorf("Should have no stderr output: %v", run.ErrorOutput())
	}
}

func TestPrompt(t *testing.T) {
	var eventContent string
	tools := ToolDef{
		Instructions: "Use the sys.prompt user to ask the user for 'first name' which is not sensitive. After you get their first name, say hello.",
		Tools:        []string{"sys.prompt"},
	}

	run, err := g.Evaluate(context.Background(), Options{IncludeEvents: true, Prompt: true}, tools)
	if err != nil {
		t.Errorf("Error executing tool: %v", err)
	}

	// Wait for the prompt event
	var promptFrame *PromptFrame
	for e := range run.Events() {
		if e.Call != nil {
			for _, o := range e.Call.Output {
				eventContent += o.Content
			}
		}
		if e.Prompt != nil {
			if e.Prompt.Type == EventTypePrompt {
				promptFrame = e.Prompt
				break
			}
		}
	}

	if promptFrame == nil {
		t.Fatalf("No prompt call event")
	}

	if promptFrame.Sensitive {
		t.Errorf("Unexpected sensitive prompt event: %v", promptFrame.Sensitive)
	}

	if !strings.Contains(promptFrame.Message, "first name") {
		t.Errorf("unexpected confirm input: %s", promptFrame.Message)
	}

	if len(promptFrame.Fields) != 1 {
		t.Fatalf("Unexpected number of fields: %d", len(promptFrame.Fields))
	}

	if promptFrame.Fields[0] != "first name" {
		t.Errorf("Unexpected field: %s", promptFrame.Fields[0])
	}

	if err = g.PromptResponse(context.Background(), PromptResponse{
		ID:        promptFrame.ID,
		Responses: map[string]string{promptFrame.Fields[0]: "Clicky"},
	}); err != nil {
		t.Errorf("Error responding: %v", err)
	}

	// Read the remainder of the events
	for e := range run.Events() {
		if e.Call != nil {
			for _, o := range e.Call.Output {
				eventContent += o.Content
			}
		}
	}

	out, err := run.Text()
	if err != nil {
		t.Errorf("Error reading output: %v", err)
	}

	if !strings.Contains(eventContent, "Clicky") {
		t.Errorf("Unexpected event output: %s", eventContent)
	}

	if !strings.Contains(out, "Hello") || !strings.Contains(out, "Clicky") {
		t.Errorf("Unexpected output: %s", out)
	}

	if len(run.ErrorOutput()) != 0 {
		t.Errorf("Should have no stderr output: %v", run.ErrorOutput())
	}
}

func TestPromptWithoutPromptAllowed(t *testing.T) {
	tools := ToolDef{
		Instructions: "Use the sys.prompt user to ask the user for 'first name' which is not sensitive. After you get their first name, say hello.",
		Tools:        []string{"sys.prompt"},
	}

	run, err := g.Evaluate(context.Background(), Options{IncludeEvents: true}, tools)
	if err != nil {
		t.Errorf("Error executing tool: %v", err)
	}

	// Wait for the prompt event
	var promptFrame *PromptFrame
	for e := range run.Events() {
		if e.Prompt != nil {
			if e.Prompt.Type == EventTypePrompt {
				promptFrame = e.Prompt
				break
			}
		}
	}

	if promptFrame != nil {
		t.Errorf("Prompt call event shouldn't happen")
	}

	_, err = run.Text()
	if err == nil || !strings.Contains(err.Error(), "prompt event occurred") {
		t.Errorf("Error reading output: %v", err)
	}

	if run.State() != Error {
		t.Errorf("Unexpected state: %v", run.State())
	}
}

func TestGetCommand(t *testing.T) {
	currentEnvVar := os.Getenv("GPTSCRIPT_BIN")
	t.Cleanup(func() {
		_ = os.Setenv("GPTSCRIPT_BIN", currentEnvVar)
	})

	tests := []struct {
		name   string
		envVar string
		want   string
	}{
		{
			name: "no env var set",
			want: "gptscript",
		},
		{
			name:   "env var set to absolute path",
			envVar: "/usr/local/bin/gptscript",
			want:   "/usr/local/bin/gptscript",
		},
		{
			name:   "env var set to relative path",
			envVar: "../bin/gptscript",
			want:   "../bin/gptscript",
		},
		{
			name:   "env var set to relative 'to me' path",
			envVar: "<me>/../bin/gptscript",
			want:   filepath.Join(filepath.Dir(os.Args[0]), "../bin/gptscript"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_ = os.Setenv("GPTSCRIPT_BIN", tt.envVar)
			if got := getCommand(); got != tt.want {
				t.Errorf("getCommand() = %v, want %v", got, tt.want)
			}
		})
	}
}
