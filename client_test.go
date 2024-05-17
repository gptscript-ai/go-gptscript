package gptscript

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
)

var client *Client

func TestMain(m *testing.M) {
	if os.Getenv("OPENAI_API_KEY") == "" && os.Getenv("GPTSCRIPT_URL") == "" {
		panic("OPENAI_API_KEY or GPTSCRIPT_URL environment variable must be set")
	}

	client = NewClient(ClientOpts{GPTScriptURL: os.Getenv("GPTSCRIPT_URL"), GPTScriptBin: os.Getenv("GPTSCRIPT_BIN")})
	os.Exit(m.Run())
}

func TestVersion(t *testing.T) {
	out, err := client.Version(context.Background())
	if err != nil {
		t.Errorf("Error getting version: %v", err)
	}

	if !strings.HasPrefix(out, "gptscript version") {
		t.Errorf("Unexpected output: %s", out)
	}
}

func TestListTools(t *testing.T) {
	tools, err := client.ListTools(context.Background())
	if err != nil {
		t.Errorf("Error listing tools: %v", err)
	}

	if len(tools) == 0 {
		t.Error("No tools found")
	}
}

func TestListModels(t *testing.T) {
	models, err := client.ListModels(context.Background())
	if err != nil {
		t.Errorf("Error listing models: %v", err)
	}

	if len(models) == 0 {
		t.Error("No models found")
	}
}

func TestSimpleEvaluate(t *testing.T) {
	tool := &ToolDef{Instructions: "What is the capital of the united states?"}

	run, err := client.Evaluate(context.Background(), Opts{}, tool)
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
}

func TestEvaluateWithContext(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Error getting current working directory: %v", err)
	}

	tool := &ToolDef{
		Instructions: "What is the capital of the united states?",
		Context: []string{
			wd + "/test/acorn-labs-context.gpt",
		},
	}

	run, err := client.Evaluate(context.Background(), Opts{DisableCache: true, IncludeEvents: true}, tool)
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

func TestRunFileChdir(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Error getting current working directory: %v", err)
	}
	// By changing the directory here, we should be able to find the test.gpt file without `./test` (see TestStreamRunFile)
	run, err := client.Run(context.Background(), "test.gpt", Opts{Chdir: wd + "/test"})
	if err != nil {
		t.Errorf("Error executing tool: %v", err)
	}

	out, err := run.Text()
	if err != nil {
		t.Errorf("Error reading output: %v", err)
	}

	if out == "" {
		t.Error("No output from tool")
	}
}

func TestEvaluateComplexTool(t *testing.T) {
	tool := &ToolDef{
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

	run, err := client.Evaluate(context.Background(), Opts{DisableCache: true}, tool)
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
	tools := []fmt.Stringer{
		&ToolDef{
			Tools:        []string{"echo"},
			Instructions: "echo hello there",
		},
		&ToolDef{
			Name:        "echo",
			Tools:       []string{"sys.exec"},
			Description: "Echoes the input",
			Args: map[string]string{
				"input": "The string input to echo",
			},
			Instructions: shebang + "\n echo ${input}",
		},
	}

	run, err := client.Evaluate(context.Background(), Opts{}, tools...)
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
	tools := []fmt.Stringer{
		&ToolDef{
			Tools:        []string{"echo"},
			Instructions: "echo hello there",
		},
		&ToolDef{
			Name:         "other",
			Tools:        []string{"echo"},
			Instructions: "echo hello somewhere else",
		},
		&ToolDef{
			Name:        "echo",
			Tools:       []string{"sys.exec"},
			Description: "Echoes the input",
			Args: map[string]string{
				"input": "The string input to echo",
			},
			Instructions: shebang + "\n echo ${input}",
		},
	}

	run, err := client.Evaluate(context.Background(), Opts{SubTool: "other"}, tools...)
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
	tool := &ToolDef{Instructions: "What is the capital of the united states?"}

	run, err := client.Evaluate(context.Background(), Opts{IncludeEvents: true}, tool)
	if err != nil {
		t.Errorf("Error executing tool: %v", err)
	}

	for e := range run.Events() {
		eventContent += e.Content
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

	if len(run.ErrorOutput()) == 0 {
		t.Error("No stderr output")
	}
}

func TestStreamRun(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Error getting working directory: %v", err)
	}

	var eventContent string
	run, err := client.Run(context.Background(), wd+"/test/catcher.gpt", Opts{IncludeEvents: true})
	if err != nil {
		t.Errorf("Error executing file: %v", err)
	}

	for e := range run.Events() {
		eventContent += e.Content
	}

	stdErr, err := io.ReadAll(run.stderr)
	if err != nil {
		t.Errorf("Error reading stderr: %v", err)
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

	if len(stdErr) == 0 {
		t.Error("No stderr output")
	}
}

func TestParseSimpleFile(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Error getting working directory: %v", err)
	}

	tools, err := client.Parse(context.Background(), wd+"/test/test.gpt")
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
	tools, err := client.ParseTool(context.Background(), "echo hello")
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
	tools, err := client.ParseTool(context.Background(), "echo hello\n---\n!markdown\nhello")
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

	if tools[1].TextNode.Text != "!markdown\nhello\n" {
		t.Errorf("Unexpected text: %s", tools[1].TextNode.Text)
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
						Type: "object",
						Properties: map[string]*openapi3.SchemaRef{
							"input": {
								Value: &openapi3.Schema{
									Description: "The string input to echo",
									Type:        "string",
								},
							},
						},
					},
				},
			},
		},
	}

	out, err := client.Fmt(context.Background(), nodes)
	if err != nil {
		t.Errorf("Error formatting nodes: %v", err)
	}

	if out != `Tools: echo

echo hello there

---
Name: echo
Args: input: The string input to echo

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
				Text: "!markdown\nWe now echo hello there\n",
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
						Type: "object",
						Properties: map[string]*openapi3.SchemaRef{
							"input": {
								Value: &openapi3.Schema{
									Description: "The string input to echo",
									Type:        "string",
								},
							},
						},
					},
				},
			},
		},
	}

	out, err := client.Fmt(context.Background(), nodes)
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
Args: input: The string input to echo

#!/bin/bash
echo hello there
` {
		t.Errorf("Unexpected output: %s", out)
	}
}

func TestToolChat(t *testing.T) {
	tool := &ToolDef{
		Chat:         true,
		Instructions: "You are a chat bot. Don't finish the conversation until I say 'bye'.",
		Tools:        []string{"sys.chat.finish"},
	}

	run, err := client.Evaluate(context.Background(), Opts{DisableCache: true}, tool)
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

	run, err := client.Run(context.Background(), wd+"/test/chat.gpt", Opts{})
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

	run, err := client.Run(context.Background(), wd+"/test/global-tools.gpt", Opts{DisableCache: true, IncludeEvents: true})
	if err != nil {
		t.Errorf("Error executing tool: %v", err)
	}

	for e := range run.Events() {
		if e.Type == EventTypeRunStart {
			runStartSeen = true
		} else if e.Type == EventTypeCallStart {
			callStartSeen = true
		} else if e.Type == EventTypeCallFinish {
			callFinishSeen = true
		} else if e.Type == EventTypeRunFinish {
			runFinishSeen = true
		} else if e.Type == EventTypeCallProgress {
			callProgressSeen = true
		}
		eventContent += e.Content
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

	if len(run.ErrorOutput()) == 0 {
		t.Error("No stderr output")
	}

	if !runStartSeen || !callStartSeen || !callFinishSeen || !runFinishSeen || !callProgressSeen {
		t.Errorf("Missing events: %t %t %t %t %t", runStartSeen, callStartSeen, callFinishSeen, runFinishSeen, callProgressSeen)
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
