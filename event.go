package gptscript

import "time"

type Event struct {
	RunID              string          `json:"runID,omitempty"`
	Time               time.Time       `json:"time,omitempty"`
	CallContext        *CallContext    `json:"callContext,omitempty"`
	ToolSubCalls       map[string]Call `json:"toolSubCalls,omitempty"`
	ToolResults        int             `json:"toolResults,omitempty"`
	Type               EventType       `json:"type,omitempty"`
	ChatCompletionID   string          `json:"chatCompletionId,omitempty"`
	ChatRequest        any             `json:"chatRequest,omitempty"`
	ChatResponse       any             `json:"chatResponse,omitempty"`
	ChatResponseCached bool            `json:"chatResponseCached,omitempty"`
	Content            string          `json:"content,omitempty"`
	Program            *Program        `json:"program,omitempty"`
	Input              string          `json:"input,omitempty"`
	Output             string          `json:"output,omitempty"`
	Err                string          `json:"err,omitempty"`
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
	EventTypeCallFinish   EventType = "callFinish"
	EventTypeRunFinish    EventType = "runFinish"
)
