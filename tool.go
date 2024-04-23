package gptscript

import (
	"fmt"
	"strings"
)

// Tool struct represents a tool with various configurations.
type Tool struct {
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

// NewTool is a constructor for Tool struct.
func NewTool(name, description string, tools []string, maxTokens *int, model string, cache bool, temperature *float64, args map[string]string, internalPrompt bool, instructions string, jsonResponse bool) *Tool {
	return &Tool{
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

// String method returns the string representation of Tool.
func (t *Tool) String() string {
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

type Tools []Tool

func (t Tools) String() string {
	resp := make([]string, 0, len(t))
	for _, tool := range t {
		resp = append(resp, tool.String())
	}
	return strings.Join(resp, "\n---\n")
}
