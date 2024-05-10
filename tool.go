package gptscript

import (
	"fmt"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

// ToolDef struct represents a tool with various configurations.
type ToolDef struct {
	Name            string            `json:"name,omitempty"`
	Description     string            `json:"description,omitempty"`
	MaxTokens       int               `json:"maxTokens,omitempty"`
	ModelName       string            `json:"modelName,omitempty"`
	ModelProvider   bool              `json:"modelProvider,omitempty"`
	JSONResponse    bool              `json:"jsonResponse,omitempty"`
	Chat            bool              `json:"chat,omitempty"`
	Temperature     *float32          `json:"temperature,omitempty"`
	Cache           *bool             `json:"cache,omitempty"`
	InternalPrompt  *bool             `json:"internalPrompt"`
	Args            map[string]string `json:"args,omitempty"`
	Tools           []string          `json:"tools,omitempty"`
	GlobalTools     []string          `json:"globalTools,omitempty"`
	GlobalModelName string            `json:"globalModelName,omitempty"`
	Context         []string          `json:"context,omitempty"`
	ExportContext   []string          `json:"exportContext,omitempty"`
	Export          []string          `json:"export,omitempty"`
	Credentials     []string          `json:"credentials,omitempty"`
	Instructions    string            `json:"instructions,omitempty"`
}

// String method returns the string representation of ToolDef.
func (t *ToolDef) String() string {
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
	if t.MaxTokens != 0 {
		sb.WriteString(fmt.Sprintf("Max tokens: %d\n", t.MaxTokens))
	}
	if t.ModelName != "" {
		sb.WriteString(fmt.Sprintf("Model: %s\n", t.ModelName))
	}
	if t.Cache != nil && !*t.Cache {
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
	if t.Chat {
		sb.WriteString("Chat: true\n")
	}
	if t.InternalPrompt != nil && *t.InternalPrompt {
		sb.WriteString("Internal prompt: true\n")
	}
	if t.Instructions != "" {
		sb.WriteString(t.Instructions)
	}

	return sb.String()
}

type ToolDefs []ToolDef

func (t ToolDefs) String() string {
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
	ToolDef     `json:",inline"`
	ID          string            `json:"id,omitempty"`
	Arguments   *openapi3.Schema  `json:"arguments,omitempty"`
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
