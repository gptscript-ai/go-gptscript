package gptscript

type AuthResponse struct {
	ID      string `json:"id"`
	Accept  bool   `json:"accept"`
	Message string `json:"message"`
}
