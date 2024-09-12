package gptscript

import "time"

type CredentialType string

const (
	CredentialTypeTool          CredentialType = "tool"
	CredentialTypeModelProvider CredentialType = "modelProvider"
)

type Credential struct {
	Context      string            `json:"context"`
	ToolName     string            `json:"toolName"`
	Type         CredentialType    `json:"type"`
	Env          map[string]string `json:"env"`
	Ephemeral    bool              `json:"ephemeral,omitempty"`
	ExpiresAt    *time.Time        `json:"expiresAt"`
	RefreshToken string            `json:"refreshToken"`
}

type CredentialRequest struct {
	Content     string `json:"content"`
	AllContexts bool   `json:"allContexts"`
	Context     string `json:"context"`
	Name        string `json:"name"`
}
