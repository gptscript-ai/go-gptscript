package gptscript

// GlobalOptions allows specification of settings that are used for every call made.
// These options can be overridden by the corresponding Options.
type GlobalOptions struct {
	OpenAIAPIKey  string   `json:"APIKey"`
	OpenAIBaseURL string   `json:"BaseURL"`
	DefaultModel  string   `json:"DefaultModel"`
	Env           []string `json:"env"`
}

func (g GlobalOptions) toEnv() []string {
	var args []string
	if g.OpenAIAPIKey != "" {
		args = append(args, "OPENAI_API_KEY="+g.OpenAIAPIKey)
	}
	if g.OpenAIBaseURL != "" {
		args = append(args, "OPENAI_BASE_URL="+g.OpenAIBaseURL)
	}
	if g.DefaultModel != "" {
		args = append(args, "GPTSCRIPT_DEFAULT_MODEL="+g.DefaultModel)
	}

	return args
}

// Options represents options for the gptscript tool or file.
type Options struct {
	GlobalOptions `json:",inline"`

	Confirm       bool   `json:"confirm"`
	Input         string `json:"input"`
	DisableCache  bool   `json:"disableCache"`
	CacheDir      string `json:"cacheDir"`
	SubTool       string `json:"subTool"`
	Workspace     string `json:"workspace"`
	ChatState     string `json:"chatState"`
	IncludeEvents bool   `json:"includeEvents"`
	Prompt        bool   `json:"prompt"`
}
