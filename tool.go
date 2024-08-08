package gptscript

import (
	"fmt"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

// ToolDef struct represents a tool with various configurations.
type ToolDef struct {
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
	Agents          []string         `json:"agents,omitempty"`
	Credentials     []string         `json:"credentials,omitempty"`
	Instructions    string           `json:"instructions,omitempty"`
	Type            string           `json:"type,omitempty"`
}

func ObjectSchema(kv ...string) *openapi3.Schema {
	s := &openapi3.Schema{
		Type:       &openapi3.Types{"object"},
		Properties: openapi3.Schemas{},
	}
	for i, v := range kv {
		if i%2 == 1 {
			s.Properties[kv[i-1]] = &openapi3.SchemaRef{
				Value: &openapi3.Schema{
					Description: v,
					Type:        &openapi3.Types{"string"},
				},
			}
		}
	}
	return s
}

type Document struct {
	Nodes []Node `json:"nodes,omitempty"`
}

type Node struct {
	TextNode *TextNode `json:"textNode,omitempty"`
	ToolNode *ToolNode `json:"toolNode,omitempty"`
}

type TextNode struct {
	Fmt  string `json:"fmt,omitempty"`
	Text string `json:"text,omitempty"`
}

func (n *TextNode) combine() {
	if n != nil && n.Fmt != "" {
		n.Text = fmt.Sprintf("!%s\n%s", n.Fmt, n.Text)
		n.Fmt = ""
	}
}

func (n *TextNode) process() {
	if n != nil && strings.HasPrefix(n.Text, "!") {
		n.Fmt, n.Text, _ = strings.Cut(strings.TrimPrefix(n.Text, "!"), "\n")
	}
}

type ToolNode struct {
	Fmt  string `json:"fmt,omitempty"`
	Tool Tool   `json:"tool,omitempty"`
}

type Tool struct {
	ToolDef     `json:",inline"`
	ID          string                     `json:"id,omitempty"`
	Arguments   *openapi3.Schema           `json:"arguments,omitempty"`
	ToolMapping map[string][]ToolReference `json:"toolMapping,omitempty"`
	MetaData    map[string]string          `json:"metadata,omitempty"`
	LocalTools  map[string]string          `json:"localTools,omitempty"`
	Source      ToolSource                 `json:"source,omitempty"`
	WorkingDir  string                     `json:"workingDir,omitempty"`
}

type ToolReference struct {
	Named     string `json:"named,omitempty"`
	Reference string `json:"reference,omitempty"`
	Arg       string `json:"arg,omitempty"`
	ToolID    string `json:"toolID,omitempty"`
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
