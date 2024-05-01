package gptscript

import (
	"fmt"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

// SimpleTool struct represents a tool with various configurations.
type SimpleTool struct {
	Name           string
	Description    string
	Tools          []string
	MaxTokens      *int // Using a pointer to represent optional int
	Model          string
	Cache          bool
	Temperature    *float64 // Using a pointer to represent optional float64
	Args           map[string]string
	InternalPrompt bool
	Instructions   string
	JSONResponse   bool
}

// NewSimpleTool is a constructor for SimpleTool struct.
func NewSimpleTool(name, description string, tools []string, maxTokens *int, model string, cache bool, temperature *float64, args map[string]string, internalPrompt bool, instructions string, jsonResponse bool) *SimpleTool {
	return &SimpleTool{
		Name:           name,
		Description:    description,
		Tools:          tools,
		MaxTokens:      maxTokens,
		Model:          model,
		Cache:          cache,
		Temperature:    temperature,
		Args:           args,
		InternalPrompt: internalPrompt,
		Instructions:   instructions,
		JSONResponse:   jsonResponse,
	}
}

// String method returns the string representation of SimpleTool.
func (t *SimpleTool) String() string {
	var sb strings.Builder

	if t.Name != "" {
		sb.WriteString(fmt.Sprintf("Name: %s\n", t.Name))
	}
	if t.Description != "" {
		sb.WriteString(fmt.Sprintf("Description: %s\n", t.Description))
	}
	if len(t.Tools) > 0 {
		sb.WriteString(fmt.Sprintf("Tools: %s\n", strings.Join(t.Tools, ", ")))
	}
	if t.MaxTokens != nil {
		sb.WriteString(fmt.Sprintf("Max tokens: %d\n", *t.MaxTokens))
	}
	if t.Model != "" {
		sb.WriteString(fmt.Sprintf("Model: %s\n", t.Model))
	}
	if !t.Cache {
		sb.WriteString("Cache: false\n")
	}
	if t.Temperature != nil {
		sb.WriteString(fmt.Sprintf("Temperature: %f\n", *t.Temperature))
	}
	if t.JSONResponse {
		sb.WriteString("JSON Response: true\n")
	}
	if len(t.Args) > 0 {
		for arg, desc := range t.Args {
			sb.WriteString(fmt.Sprintf("Args: %s: %s\n", arg, desc))
		}
	}
	if t.InternalPrompt {
		sb.WriteString("Internal prompt: true\n")
	}
	if t.Instructions != "" {
		sb.WriteString(t.Instructions)
	}

	return sb.String()
}

// FreeForm struct represents free-form content.
type FreeForm struct {
	Content string
}

// NewFreeForm is a constructor for FreeForm struct.
func NewFreeForm(content string) *FreeForm {
	return &FreeForm{Content: content}
}

// String method returns the string representation of FreeForm.
func (f *FreeForm) String() string {
	return f.Content
}

type Tools []SimpleTool

func (t Tools) String() string {
	resp := make([]string, 0, len(t))
	for _, tool := range t {
		resp = append(resp, tool.String())
	}
	return strings.Join(resp, "\n---\n")
}

type Document struct {
	Nodes []Node `json:"nodes,omitempty"`
}

type Node struct {
	TextNode *TextNode `json:"textNode,omitempty"`
	ToolNode *ToolNode `json:"toolNode,omitempty"`
}

type TextNode struct {
	Text string `json:"text,omitempty"`
}

type ToolNode struct {
	Tool Tool `json:"tool,omitempty"`
}

type Tool struct {
	Parameters   `json:",inline"`
	Instructions string `json:"instructions,omitempty"`

	ID          string            `json:"id,omitempty"`
	ToolMapping map[string]string `json:"toolMapping,omitempty"`
	LocalTools  map[string]string `json:"localTools,omitempty"`
	Source      ToolSource        `json:"source,omitempty"`
	WorkingDir  string            `json:"workingDir,omitempty"`
}

type ToolSource struct {
	Location string `json:"location,omitempty"`
	LineNo   int    `json:"lineNo,omitempty"`
	Repo     *Repo  `json:"repo,omitempty"`
}

type Repo struct {
	VCS      string
	Root     string
	Path     string
	Name     string
	Revision string
}

type Parameters struct {
	Name            string           `json:"name,omitempty"`
	Description     string           `json:"description,omitempty"`
	MaxTokens       int              `json:"maxTokens,omitempty"`
	ModelName       string           `json:"modelName,omitempty"`
	ModelProvider   bool             `json:"modelProvider,omitempty"`
	JSONResponse    bool             `json:"jsonResponse,omitempty"`
	Chat            bool             `json:"chat,omitempty"`
	Temperature     *float32         `json:"temperature,omitempty"`
	Cache           *bool            `json:"cache,omitempty"`
	InternalPrompt  *bool            `json:"internalPrompt"`
	Arguments       *openapi3.Schema `json:"arguments,omitempty"`
	Tools           []string         `json:"tools,omitempty"`
	GlobalTools     []string         `json:"globalTools,omitempty"`
	GlobalModelName string           `json:"globalModelName,omitempty"`
	Context         []string         `json:"context,omitempty"`
	ExportContext   []string         `json:"exportContext,omitempty"`
	Export          []string         `json:"export,omitempty"`
	Credentials     []string         `json:"credentials,omitempty"`
	Blocking        bool             `json:"-"`
}
