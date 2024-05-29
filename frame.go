package gptscript

import (
	"time"
)

type Frame struct {
	Run  *RunFrame  `json:"run,omitempty"`
	Call *CallFrame `json:"call,omitempty"`
}

type RunFrame struct {
	ID        string    `json:"id"`
	Program   Program   `json:"program"`
	Input     string    `json:"input"`
	Output    string    `json:"output"`
	Error     string    `json:"error"`
	Start     time.Time `json:"start"`
	End       time.Time `json:"end"`
	State     RunState  `json:"state"`
	ChatState any       `json:"chatState"`
	Type      EventType `json:"type"`
}

type CallFrame struct {
	CallContext `json:",inline"`

	Type        EventType `json:"type"`
	Start       time.Time `json:"start"`
	End         time.Time `json:"end"`
	Input       string    `json:"input"`
	Output      []Output  `json:"output"`
	Usage       Usage     `json:"usage"`
	LLMRequest  any       `json:"llmRequest"`
	LLMResponse any       `json:"llmResponse"`
}

type Usage struct {
	PromptTokens     int `json:"promptTokens,omitempty"`
	CompletionTokens int `json:"completionTokens,omitempty"`
	TotalTokens      int `json:"totalTokens,omitempty"`
}

type Output struct {
	Content  string          `json:"content"`
	SubCalls map[string]Call `json:"subCalls"`
}

type Program struct {
	Name        string  `json:"name,omitempty"`
	EntryToolID string  `json:"entryToolId,omitempty"`
	ToolSet     ToolSet `json:"toolSet,omitempty"`
}

type ToolSet map[string]Tool

type Call struct {
	ToolID string `json:"toolID,omitempty"`
	Input  string `json:"input,omitempty"`
}

type CallContext struct {
	ID           string         `json:"id"`
	Tool         Tool           `json:"tool"`
	DisplayText  string         `json:"displayText"`
	InputContext []InputContext `json:"inputContext"`
	ToolCategory ToolCategory   `json:"toolCategory,omitempty"`
	ToolName     string         `json:"toolName,omitempty"`
	ParentID     string         `json:"parentID,omitempty"`
}

type InputContext struct {
	ToolID  string `json:"toolID,omitempty"`
	Content string `json:"content,omitempty"`
}

type ToolCategory string

const (
	CredentialToolCategory ToolCategory = "credential"
	ContextToolCategory    ToolCategory = "context"
	NoCategory             ToolCategory = ""
)

type EventType string

const (
	EventTypeRunStart     EventType = "runStart"
	EventTypeCallStart    EventType = "callStart"
	EventTypeCallContinue EventType = "callContinue"
	EventTypeCallSubCalls EventType = "callSubCalls"
	EventTypeCallProgress EventType = "callProgress"
	EventTypeChat         EventType = "callChat"
	EventTypeCallConfirm  EventType = "callConfirm"
	EventTypeCallFinish   EventType = "callFinish"
	EventTypeRunFinish    EventType = "runFinish"
)
