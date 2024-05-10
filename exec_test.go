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
	tool := &SimpleTool{
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
		&SimpleTool{
			Tools:        []string{"echo"},
			Instructions: "echo hello there",
		},
		&SimpleTool{
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

func TestExecWithToolListAndSubTool(t *testing.T) {
	shebang := "#!/bin/bash"
	if runtime.GOOS == "windows" {
		shebang = "#!/usr/bin/env powershell.exe"
	}
	tools := []fmt.Stringer{
		&SimpleTool{
			Tools:        []string{"echo"},
			Instructions: "echo hello there",
		},
		&SimpleTool{
			Name:         "other",
			Tools:        []string{"echo"},
			Instructions: "echo hello somewhere else",
		},
		&SimpleTool{
			Name:        "echo",
			Tools:       []string{"sys.exec"},
			Description: "Echoes the input",
			Args: map[string]string{
				"input": "The string input to echo",
			},
			Instructions: shebang + "\n echo ${input}",
		},
	}

	out, err := ExecTool(context.Background(), Opts{SubTool: "other"}, tools...)
	if err != nil {
		t.Errorf("Error executing tool: %v", err)
	}

	if !strings.Contains(out, "hello somewhere else") {
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

func TestParseSimpleFile(t *testing.T) {
	tools, err := Parse(context.Background(), "./test/test.gpt", Opts{})
	if err != nil {
		t.Errorf("Error parsing file: %v", err)
	}

	if len(tools) != 1 {
		t.Errorf("Unexpected number of tools: %d", len(tools))
	}

	if tools[0].ToolNode == nil {
		t.Error("No tool node found")
	}

	if tools[0].ToolNode.Tool.Instructions != "Respond with a hello, in a random language. Also include the language in the response." {
		t.Errorf("Unexpected instructions: %s", tools[0].ToolNode.Tool.Instructions)
	}
}

func TestParseSimpleFileWithChdir(t *testing.T) {
	tools, err := Parse(context.Background(), "./test.gpt", Opts{Chdir: "./test"})
	if err != nil {
		t.Errorf("Error parsing file: %v", err)
	}

	if len(tools) != 1 {
		t.Errorf("Unexpected number of tools: %d", len(tools))
	}

	if tools[0].ToolNode == nil {
		t.Error("No tool node found")
	}

	if tools[0].ToolNode.Tool.Instructions != "Respond with a hello, in a random language. Also include the language in the response." {
		t.Errorf("Unexpected instructions: %s", tools[0].ToolNode.Tool.Instructions)
	}
}

func TestParseTool(t *testing.T) {
	tools, err := ParseTool(context.Background(), "echo hello")
	if err != nil {
		t.Errorf("Error parsing tool: %v", err)
	}

	if len(tools) != 1 {
		t.Errorf("Unexpected number of tools: %d", len(tools))
	}

	if tools[0].ToolNode == nil {
		t.Error("No tool node found")
	}

	if tools[0].ToolNode.Tool.Instructions != "echo hello" {
		t.Errorf("Unexpected instructions: %s", tools[0].ToolNode.Tool.Instructions)
	}
}

func TestParseToolWithTextNode(t *testing.T) {
	tools, err := ParseTool(context.Background(), "echo hello\n---\n!markdown\nhello")
	if err != nil {
		t.Errorf("Error parsing tool: %v", err)
	}

	if len(tools) != 2 {
		t.Errorf("Unexpected number of tools: %d", len(tools))
	}

	if tools[0].ToolNode == nil {
		t.Error("No tool node found")
	}

	if tools[0].ToolNode.Tool.Instructions != "echo hello" {
		t.Errorf("Unexpected instructions: %s", tools[0].ToolNode.Tool.Instructions)
	}

	if tools[1].TextNode == nil {
		t.Error("No text node found")
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
					Parameters: Parameters{
						Tools: []string{"echo"},
					},
					Instructions: "echo hello there",
				},
			},
		},
		{
			ToolNode: &ToolNode{
				Tool: Tool{
					Parameters: Parameters{
						Name: "echo",
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
					Instructions: "#!/bin/bash\necho hello there",
				},
			},
		},
	}

	out, err := Fmt(context.Background(), nodes)
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
					Parameters: Parameters{
						Tools: []string{"echo"},
					},
					Instructions: "echo hello there",
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
					Parameters: Parameters{
						Name: "echo",
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
					Instructions: "#!/bin/bash\necho hello there",
				},
			},
		},
	}

	out, err := Fmt(context.Background(), nodes)
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

func TestExecWithWorkspace(t *testing.T) {
	tool := &SimpleTool{
		Tools:        []string{"sys.workspace.ls", "sys.workspace.write"},
		Instructions: "Write a file named 'hello.txt' to the workspace with the content 'Hello!' and list the contents of the workspace.",
	}

	out, err := ExecTool(context.Background(), Opts{Workspace: "./workspace"}, tool)
	if err != nil {
		t.Errorf("Error executing tool: %v", err)
	}

	if !strings.Contains(out, "hello.txt") {
		t.Errorf("Unexpected output: %s", out)
	}
}

func TestDetermineProperCommand(t *testing.T) {
	tests := []struct {
		name     string
		dir, bin string
		want     string
	}{
		{
			name: "no dir",
			bin:  "gptscript",
			want: "gptscript",
		},
		{
			name: "bin set to absolute path",
			bin:  string(os.PathSeparator) + filepath.Join("usr", "local", "bin", "gptscript"),
			dir:  string(os.PathSeparator) + filepath.Join("usr", "local"),
			want: string(os.PathSeparator) + filepath.Join("usr", "local", "bin", "gptscript"),
		},
		{
			name: "bin set to relative path",
			bin:  filepath.Join("..", "bin", "gptscript"),
			dir:  string(os.PathSeparator) + filepath.Join("usr", "local"),
			want: filepath.Join("..", "bin", "gptscript"),
		},
		{
			name: "bin set to relative 'to me' path with os.Args[0]",
			bin:  "<me>" + string(os.PathSeparator) + filepath.Join("..", "bin", "gptscript"),
			dir:  filepath.Dir(os.Args[0]),
			want: filepath.Join(filepath.Dir(os.Args[0]), filepath.Join("..", "bin", "gptscript")),
		},
		{
			name: "env var set to relative 'to me' path with extra and dir is current",
			bin:  "<me>" + string(os.PathSeparator) + filepath.Join("..", "bin", "gptscript"),
			dir:  ".",
			want: "." + string(os.PathSeparator) + filepath.Join("..", "bin", "gptscript"),
		},
		{
			name: "env var set to relative 'to me' path and dir is current",
			bin:  "<me>" + string(os.PathSeparator) + "gptscript",
			dir:  ".",
			want: "." + string(os.PathSeparator) + "gptscript",
		},
		{
			name: "env var set to relative 'to me' path with extra and dir is current",
			bin:  "<me>" + string(os.PathSeparator) + filepath.Join("..", "bin", "gptscript"),
			dir:  ".",
			want: "." + string(os.PathSeparator) + filepath.Join("..", "bin", "gptscript"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := determineProperCommand(tt.dir, tt.bin); got != tt.want {
				t.Errorf("determineProperCommand() = %v, want %v", got, tt.want)
			}
		})
	}
}
