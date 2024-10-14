package gptscript

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/stretchr/testify/require"
)

var g *GPTScript

func TestMain(m *testing.M) {
	if os.Getenv("OPENAI_API_KEY") == "" && os.Getenv("GPTSCRIPT_URL") == "" {
		panic("OPENAI_API_KEY or GPTSCRIPT_URL environment variable must be set")
	}

	// Start an initial GPTScript instance.
	// This one doesn't have any options, but it's there to ensure that using another instance works as expected in all cases.
	gFirst, err := NewGPTScript(GlobalOptions{})
	if err != nil {
		panic(fmt.Sprintf("error creating gptscript: %s", err))
	}

	g, err = NewGPTScript(GlobalOptions{OpenAIAPIKey: os.Getenv("OPENAI_API_KEY")})
	if err != nil {
		gFirst.Close()
		panic(fmt.Sprintf("error creating gptscript: %s", err))
	}

	exitCode := m.Run()
	g.Close()
	gFirst.Close()
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

func TestListModels(t *testing.T) {
	models, err := g.ListModels(context.Background())
	if err != nil {
		t.Errorf("Error listing models: %v", err)
	}

	if len(models) == 0 {
		t.Error("No models found")
	}
}

func TestListModelsWithProvider(t *testing.T) {
	if os.Getenv("ANTHROPIC_API_KEY") == "" {
		t.Skip("ANTHROPIC_API_KEY not set")
	}
	models, err := g.ListModels(context.Background(), ListModelsOptions{
		Providers:           []string{"github.com/gptscript-ai/claude3-anthropic-provider"},
		CredentialOverrides: []string{"github.com/gptscript-ai/claude3-anthropic-provider/credential:ANTHROPIC_API_KEY"},
	})
	if err != nil {
		t.Errorf("Error listing models: %v", err)
	}

	if len(models) == 0 {
		t.Error("No models found")
	}

	for _, model := range models {
		if !strings.HasPrefix(model, "claude-3-") || !strings.HasSuffix(model, "from github.com/gptscript-ai/claude3-anthropic-provider") {
			t.Errorf("Unexpected model name: %s", model)
		}
	}
}

func TestListModelsWithDefaultProvider(t *testing.T) {
	if os.Getenv("ANTHROPIC_API_KEY") == "" {
		t.Skip("ANTHROPIC_API_KEY not set")
	}
	g, err := NewGPTScript(GlobalOptions{
		DefaultModelProvider: "github.com/gptscript-ai/claude3-anthropic-provider",
	})
	if err != nil {
		t.Fatalf("Error creating gptscript: %v", err)
	}
	defer g.Close()

	models, err := g.ListModels(context.Background(), ListModelsOptions{
		CredentialOverrides: []string{"github.com/gptscript-ai/claude3-anthropic-provider/credential:ANTHROPIC_API_KEY"},
	})
	if err != nil {
		t.Errorf("Error listing models: %v", err)
	}

	if len(models) == 0 {
		t.Error("No models found")
	}

	for _, model := range models {
		if !strings.HasPrefix(model, "claude-3-") || !strings.HasSuffix(model, "from github.com/gptscript-ai/claude3-anthropic-provider") {
			t.Errorf("Unexpected model name: %s", model)
		}
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

	run, err := g.Evaluate(context.Background(), Options{DisableCache: true}, tool)
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

	var promptTokens, completionTokens, totalTokens int
	for _, c := range run.calls {
		promptTokens += c.Usage.PromptTokens
		completionTokens += c.Usage.CompletionTokens
		totalTokens += c.Usage.TotalTokens
	}

	if promptTokens == 0 || completionTokens == 0 || totalTokens == 0 {
		t.Errorf("Usage not set: %d, %d, %d", promptTokens, completionTokens, totalTokens)
	}
}

func TestEvaluateWithContext(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Error getting current working directory: %v", err)
	}

	tool := ToolDef{
		Instructions: "What is the capital of the united states?",
		Tools: []string{
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

	// In this case, we expect the total number of tool results to be 1
	var toolResults int
	for _, c := range run.calls {
		toolResults += c.ToolResults
	}

	if toolResults != 1 {
		t.Errorf("Unexpected number of tool results: %d", toolResults)
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

func TestSimpleRun(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Error getting working directory: %v", err)
	}

	run, err := g.Run(context.Background(), wd+"/test/catcher.gpt", Options{})
	if err != nil {
		t.Fatalf("Error executing file: %v", err)
	}

	out, err := run.Text()
	if err != nil {
		t.Errorf("Error reading output: %v", err)
	}

	if !strings.Contains(out, "Salinger") {
		t.Errorf("Unexpected output: %s", out)
	}

	if len(run.ErrorOutput()) != 0 {
		t.Error("Should have no stderr output")
	}

	// Run it a second time, ensuring the same output and that a cached response is used
	run, err = g.Run(context.Background(), wd+"/test/catcher.gpt", Options{})
	if err != nil {
		t.Fatalf("Error executing file: %v", err)
	}

	secondOut, err := run.Text()
	if err != nil {
		t.Errorf("Error reading output: %v", err)
	}

	if secondOut != out {
		t.Errorf("Unexpected output on second run: %s != %s", out, secondOut)
	}

	// In this case, we expect a single call and that the response is cached
	for _, c := range run.calls {
		if !c.ChatResponseCached {
			t.Error("Chat response should be cached")
		}
		break
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

func TestRestartFailedRun(t *testing.T) {
	shebang := "#!/bin/bash"
	instructions := "%s\nexit ${EXIT_CODE}"
	if runtime.GOOS == "windows" {
		shebang = "#!/usr/bin/env powershell.exe"
		instructions = "%s\nexit $env:EXIT_CODE"
	}
	instructions = fmt.Sprintf(instructions, shebang)
	tools := []ToolDef{
		{
			Instructions: "say hello",
			Tools:        []string{"my-context"},
		},
		{
			Name:         "my-context",
			Type:         "context",
			Instructions: instructions,
		},
	}
	run, err := g.Evaluate(context.Background(), Options{GlobalOptions: GlobalOptions{Env: []string{"EXIT_CODE=1"}}, DisableCache: true}, tools...)
	if err != nil {
		t.Fatalf("Error executing tool: %v", err)
	}

	_, err = run.Text()
	if err == nil {
		t.Errorf("Expected error but got nil")
	}

	run.opts.GlobalOptions.Env = nil
	run, err = run.NextChat(context.Background(), "")
	if err != nil {
		t.Fatalf("Error executing next run: %v", err)
	}

	_, err = run.Text()
	if err != nil {
		t.Errorf("Error reading output: %v", err)
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

func TestParseEmptyFile(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Error getting working directory: %v", err)
	}

	tools, err := g.Parse(context.Background(), wd+"/test/empty.gpt")
	if err != nil {
		t.Errorf("Error parsing file: %v", err)
	}

	if len(tools) != 0 {
		t.Fatalf("Unexpected number of tools: %d", len(tools))
	}
}

func TestParseFileWithMetadata(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Error getting working directory: %v", err)
	}

	tools, err := g.Parse(context.Background(), wd+"/test/parse-with-metadata.gpt")
	if err != nil {
		t.Errorf("Error parsing file: %v", err)
	}

	if len(tools) != 2 {
		t.Fatalf("Unexpected number of tools: %d", len(tools))
	}

	if tools[0].ToolNode == nil {
		t.Fatalf("No tool node found")
	}

	if !strings.Contains(tools[0].ToolNode.Tool.Instructions, "requests.get(") {
		t.Errorf("Unexpected instructions: %s", tools[0].ToolNode.Tool.Instructions)
	}

	if tools[0].ToolNode.Tool.MetaData["requirements.txt"] != "requests" {
		t.Errorf("Unexpected metadata: %s", tools[0].ToolNode.Tool.MetaData["requirements.txt"])
	}

	if tools[1].TextNode == nil {
		t.Fatalf("No text node found")
	}

	if tools[1].TextNode.Fmt != "metadata:foo:requirements.txt" {
		t.Errorf("Unexpected text: %s", tools[1].TextNode.Fmt)
	}
}

func TestParseTool(t *testing.T) {
	tools, err := g.ParseContent(context.Background(), "echo hello")
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

func TestEmptyParseTool(t *testing.T) {
	tools, err := g.ParseContent(context.Background(), "")
	if err != nil {
		t.Errorf("Error parsing tool: %v", err)
	}

	if len(tools) != 0 {
		t.Fatalf("Unexpected number of tools: %d", len(tools))
	}
}

func TestParseToolWithTextNode(t *testing.T) {
	tools, err := g.ParseContent(context.Background(), "echo hello\n---\n!markdown\nhello")
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

	if strings.TrimSpace(tools[1].TextNode.Text) != "hello" {
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
		"What is the second one in the list?",
		"What is the third?",
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

	run, err := g.Run(context.Background(), wd+"/test/global-tools.gpt", Options{DisableCache: true, IncludeEvents: true, CredentialOverrides: []string{"github.com/gptscript-ai/gateway:OPENAI_API_KEY"}})
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
		return
	}

	if !strings.Contains(confirmCallEvent.Input, "ls") {
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
		return
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

func TestPromptWithMetadata(t *testing.T) {
	run, err := g.Run(context.Background(), "sys.prompt", Options{IncludeEvents: true, Prompt: true, Input: `{"fields":"first name","metadata":{"key":"value"}}`})
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

	if promptFrame == nil {
		t.Fatalf("No prompt call event")
		return
	}

	if promptFrame.Sensitive {
		t.Errorf("Unexpected sensitive prompt event: %v", promptFrame.Sensitive)
	}

	if len(promptFrame.Fields) != 1 {
		t.Fatalf("Unexpected number of fields: %d", len(promptFrame.Fields))
	}

	if promptFrame.Fields[0] != "first name" {
		t.Errorf("Unexpected field: %s", promptFrame.Fields[0])
	}

	if promptFrame.Metadata["key"] != "value" {
		t.Errorf("Unexpected metadata: %v", promptFrame.Metadata)
	}

	if err = g.PromptResponse(context.Background(), PromptResponse{
		ID:        promptFrame.ID,
		Responses: map[string]string{promptFrame.Fields[0]: "Clicky"},
	}); err != nil {
		t.Errorf("Error responding: %v", err)
	}

	// Read the remainder of the events
	//nolint:revive
	for range run.Events() {
	}

	out, err := run.Text()
	if err != nil {
		t.Errorf("Error reading output: %v", err)
	}

	if !strings.Contains(out, "Clicky") {
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

func TestGetEnv(t *testing.T) {
	// Cleaning up
	defer func(currentEnvValue string) {
		os.Setenv("testKey", currentEnvValue)
	}(os.Getenv("testKey"))

	// Tests
	testCases := []struct {
		name           string
		key            string
		def            string
		envValue       string
		expectedResult string
	}{
		{
			name:           "NoValueUseDefault",
			key:            "testKey",
			def:            "defaultValue",
			envValue:       "",
			expectedResult: "defaultValue",
		},
		{
			name:           "ValueExistsNoCompress",
			key:            "testKey",
			def:            "defaultValue",
			envValue:       "testValue",
			expectedResult: "testValue",
		},
		{
			name:     "ValueExistsCompressed",
			key:      "testKey",
			def:      "defaultValue",
			envValue: `{"_gz":"H4sIAEosrGYC/ytJLS5RKEvMKU0FACtB3ewKAAAA"}`,

			expectedResult: "test value",
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			os.Setenv(test.key, test.envValue)

			result := GetEnv(test.key, test.def)

			if result != test.expectedResult {
				t.Errorf("expected: %s, got: %s", test.expectedResult, result)
			}
		})
	}
}

func TestRunPythonWithMetadata(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Error getting working directory: %v", err)
	}

	run, err := g.Run(context.Background(), wd+"/test/parse-with-metadata.gpt", Options{IncludeEvents: true})
	if err != nil {
		t.Fatalf("Error executing file: %v", err)
	}

	out, err := run.Text()
	if err != nil {
		t.Fatalf("Error reading output: %v", err)
	}

	if out != "200" {
		t.Errorf("Unexpected output: %s", out)
	}
}

func TestParseThenEvaluateWithMetadata(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Error getting working directory: %v", err)
	}

	tools, err := g.Parse(context.Background(), wd+"/test/parse-with-metadata.gpt")
	if err != nil {
		t.Fatalf("Error parsing file: %v", err)
	}

	run, err := g.Evaluate(context.Background(), Options{}, tools[0].ToolNode.Tool.ToolDef)
	if err != nil {
		t.Fatalf("Error executing file: %v", err)
	}

	out, err := run.Text()
	if err != nil {
		t.Fatalf("Error reading output: %v", err)
	}

	if out != "200" {
		t.Errorf("Unexpected output: %s", out)
	}
}

func TestLoadFile(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Error getting working directory: %v", err)
	}

	prg, err := g.LoadFile(context.Background(), wd+"/test/global-tools.gpt")
	if err != nil {
		t.Fatalf("Error loading file: %v", err)
	}

	if prg.EntryToolID == "" {
		t.Errorf("Unexpected entry tool ID: %s", prg.EntryToolID)
	}

	if len(prg.ToolSet) == 0 {
		t.Errorf("Unexpected number of tools: %d", len(prg.ToolSet))
	}

	if prg.Name == "" {
		t.Errorf("Unexpected name: %s", prg.Name)
	}
}

func TestLoadRemoteFile(t *testing.T) {
	prg, err := g.LoadFile(context.Background(), "github.com/gptscript-ai/context/workspace")
	if err != nil {
		t.Fatalf("Error loading file: %v", err)
	}

	if prg.EntryToolID == "" {
		t.Errorf("Unexpected entry tool ID: %s", prg.EntryToolID)
	}

	if len(prg.ToolSet) == 0 {
		t.Errorf("Unexpected number of tools: %d", len(prg.ToolSet))
	}

	if prg.Name == "" {
		t.Errorf("Unexpected name: %s", prg.Name)
	}
}

func TestLoadContent(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Error getting working directory: %v", err)
	}

	content, err := os.ReadFile(wd + "/test/global-tools.gpt")
	if err != nil {
		t.Fatalf("Error reading file: %v", err)
	}

	prg, err := g.LoadContent(context.Background(), string(content))
	if err != nil {
		t.Fatalf("Error loading file: %v", err)
	}

	if prg.EntryToolID == "" {
		t.Errorf("Unexpected entry tool ID: %s", prg.EntryToolID)
	}

	if len(prg.ToolSet) == 0 {
		t.Errorf("Unexpected number of tools: %d", len(prg.ToolSet))
	}

	// Name won't be set in this case
	if prg.Name != "" {
		t.Errorf("Unexpected name: %s", prg.Name)
	}
}

func TestLoadTools(t *testing.T) {
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
			Instructions: "#!/bin/bash\n echo ${input}",
		},
	}

	prg, err := g.LoadTools(context.Background(), tools)
	if err != nil {
		t.Fatalf("Error loading file: %v", err)
	}

	if prg.EntryToolID == "" {
		t.Errorf("Unexpected entry tool ID: %s", prg.EntryToolID)
	}

	if len(prg.ToolSet) == 0 {
		t.Errorf("Unexpected number of tools: %d", len(prg.ToolSet))
	}

	// Name won't be set in this case
	if prg.Name != "" {
		t.Errorf("Unexpected name: %s", prg.Name)
	}
}

func TestCredentials(t *testing.T) {
	// We will test in the following order of create, list, reveal, delete.
	name := "test-" + strconv.Itoa(rand.Int())
	if len(name) > 20 {
		name = name[:20]
	}

	// Create
	err := g.CreateCredential(context.Background(), Credential{
		Context:      "testing",
		ToolName:     name,
		Type:         CredentialTypeTool,
		Env:          map[string]string{"ENV": "testing"},
		RefreshToken: "my-refresh-token",
	})
	require.NoError(t, err)

	// List
	creds, err := g.ListCredentials(context.Background(), ListCredentialsOptions{
		CredentialContexts: []string{"testing"},
	})
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(creds), 1)

	// Reveal
	cred, err := g.RevealCredential(context.Background(), []string{"testing"}, name)
	require.NoError(t, err)
	require.Contains(t, cred.Env, "ENV")
	require.Equal(t, cred.Env["ENV"], "testing")
	require.Equal(t, cred.RefreshToken, "my-refresh-token")

	// Delete
	err = g.DeleteCredential(context.Background(), "testing", name)
	require.NoError(t, err)

	// Delete again and make sure we get a NotFoundError
	err = g.DeleteCredential(context.Background(), "testing", name)
	require.Error(t, err)
	require.True(t, errors.As(err, &ErrNotFound{}))
}

func TestDatasets(t *testing.T) {
	workspace, err := os.MkdirTemp("", "go-gptscript-test")
	require.NoError(t, err)
	defer func() {
		_ = os.RemoveAll(workspace)
	}()

	// Create a dataset
	dataset, err := g.CreateDataset(context.Background(), workspace, "test-dataset", "This is a test dataset")
	require.NoError(t, err)
	require.Equal(t, "test-dataset", dataset.Name)
	require.Equal(t, "This is a test dataset", dataset.Description)
	require.Equal(t, 0, len(dataset.Elements))

	// Add an element
	elementMeta, err := g.AddDatasetElement(context.Background(), workspace, dataset.ID, "test-element", "This is a test element", "This is the content")
	require.NoError(t, err)
	require.Equal(t, "test-element", elementMeta.Name)
	require.Equal(t, "This is a test element", elementMeta.Description)

	// Get the element
	element, err := g.GetDatasetElement(context.Background(), workspace, dataset.ID, "test-element")
	require.NoError(t, err)
	require.Equal(t, "test-element", element.Name)
	require.Equal(t, "This is a test element", element.Description)
	require.Equal(t, "This is the content", element.Contents)

	// List elements in the dataset
	elements, err := g.ListDatasetElements(context.Background(), workspace, dataset.ID)
	require.NoError(t, err)
	require.Equal(t, 1, len(elements))
	require.Equal(t, "test-element", elements[0].Name)
	require.Equal(t, "This is a test element", elements[0].Description)

	// List datasets
	datasets, err := g.ListDatasets(context.Background(), workspace)
	require.NoError(t, err)
	require.Equal(t, 1, len(datasets))
	require.Equal(t, "test-dataset", datasets[0].Name)
	require.Equal(t, "This is a test dataset", datasets[0].Description)
	require.Equal(t, dataset.ID, datasets[0].ID)
}
