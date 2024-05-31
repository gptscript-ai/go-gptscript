package gptscript

type PromptResponse struct {
	ID        string            `json:"id,omitempty"`
	Responses map[string]string `json:"response,omitempty"`
}
