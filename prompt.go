package gptscript

type PromptResponse struct {
	ID       string            `json:"id,omitempty"`
	Response map[string]string `json:"response,omitempty"`
}
