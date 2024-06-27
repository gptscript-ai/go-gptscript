package gptscript

import (
	"fmt"
	"time"
)

type ToolCategory string

type EventType string

const (
	ProviderToolCategory   ToolCategory = "provider"
	CredentialToolCategory ToolCategory = "credential"
	ContextToolCategory    ToolCategory = "context"
	InputToolCategory      ToolCategory = "input"
	OutputToolCategory     ToolCategory = "output"
	NoCategory             ToolCategory = ""

	EventTypeRunStart     EventType = "runStart"
	EventTypeCallStart    EventType = "callStart"
	EventTypeCallContinue EventType = "callContinue"
	EventTypeCallSubCalls EventType = "callSubCalls"
	EventTypeCallProgress EventType = "callProgress"
	EventTypeChat         EventType = "callChat"
	EventTypeCallConfirm  EventType = "callConfirm"
	EventTypeCallFinish   EventType = "callFinish"
	EventTypeRunFinish    EventType = "runFinish"

	EventTypePrompt EventType = "prompt"
)

type Frame struct {
	Run    *RunFrame    `json:"run,omitempty"`
	Call   *CallFrame   `json:"call,omitempty"`
	Prompt *PromptFrame `json:"prompt,omitempty"`
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
	ID           string          `json:"id"`
	Tool         Tool            `json:"tool"`
	AgentGroup   []ToolReference `json:"agentGroup,omitempty"`
	DisplayText  string          `json:"displayText"`
	InputContext []InputContext  `json:"inputContext"`
	ToolCategory ToolCategory    `json:"toolCategory,omitempty"`
	ToolName     string          `json:"toolName,omitempty"`
	ParentID     string          `json:"parentID,omitempty"`
}

type InputContext struct {
	ToolID  string `json:"toolID,omitempty"`
	Content string `json:"content,omitempty"`
}

type PromptFrame struct {
	ID        string    `json:"id,omitempty"`
	Type      EventType `json:"type,omitempty"`
	Time      time.Time `json:"time,omitempty"`
	Message   string    `json:"message,omitempty"`
	Fields    []string  `json:"fields,omitempty"`
	Sensitive bool      `json:"sensitive,omitempty"`
}

func (p *PromptFrame) String() string {
	return fmt.Sprintf(`Message: %s
Fields: %v
Sensitive: %v`, p.Message, p.Fields, p.Sensitive)
}
