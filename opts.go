package gptscript

// GlobalOptions allows specification of settings that are used for every call made.
// These options can be overridden by the corresponding Options.
type GlobalOptions struct {
	OpenAIAPIKey  string `json:"APIKey"`
	OpenAIBaseURL string `json:"BaseURL"`
	DefaultModel  string `json:"DefaultModel"`
}

func (g GlobalOptions) toArgs() []string {
	var args []string
	if g.OpenAIAPIKey != "" {
		args = append(args, "--openai-api-key", g.OpenAIAPIKey)
	}
	if g.OpenAIBaseURL != "" {
		args = append(args, "--openai-base-url", g.OpenAIBaseURL)
	}
	if g.DefaultModel != "" {
		args = append(args, "--default-model", g.DefaultModel)
	}

	return args
}

// Options represents options for the gptscript tool or file.
type Options struct {
	GlobalOptions `json:",inline"`

	Confirm       bool     `json:"confirm"`
	Input         string   `json:"input"`
	DisableCache  bool     `json:"disableCache"`
	CacheDir      string   `json:"cacheDir"`
	SubTool       string   `json:"subTool"`
	Workspace     string   `json:"workspace"`
	ChatState     string   `json:"chatState"`
	IncludeEvents bool     `json:"includeEvents"`
	Prompt        bool     `json:"prompt"`
	Env           []string `json:"env"`
}
