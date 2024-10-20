package gptscript

// GlobalOptions allows specification of settings that are used for every call made.
// These options can be overridden by the corresponding Options.
type GlobalOptions struct {
	URL                  string   `json:"url"`
	Token                string   `json:"token"`
	OpenAIAPIKey         string   `json:"APIKey"`
	OpenAIBaseURL        string   `json:"BaseURL"`
	DefaultModel         string   `json:"DefaultModel"`
	DefaultModelProvider string   `json:"DefaultModelProvider"`
	CacheDir             string   `json:"CacheDir"`
	Env                  []string `json:"env"`
	DatasetToolRepo      string   `json:"DatasetToolRepo"`
	WorkspaceTool        string   `json:"WorkspaceTool"`
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
		args = append(args, "GPTSCRIPT_SDKSERVER_DEFAULT_MODEL="+g.DefaultModel)
	}
	if g.DefaultModelProvider != "" {
		args = append(args, "GPTSCRIPT_SDKSERVER_DEFAULT_MODEL_PROVIDER="+g.DefaultModelProvider)
	}
	if g.WorkspaceTool != "" {
		args = append(args, "GPTSCRIPT_SDKSERVER_WORKSPACE_TOOL="+g.WorkspaceTool)
	}

	return args
}

func completeGlobalOptions(opts ...GlobalOptions) GlobalOptions {
	var result GlobalOptions
	for _, opt := range opts {
		result.CacheDir = firstSet(opt.CacheDir, result.CacheDir)
		result.URL = firstSet(opt.URL, result.URL)
		result.Token = firstSet(opt.Token, result.Token)
		result.OpenAIAPIKey = firstSet(opt.OpenAIAPIKey, result.OpenAIAPIKey)
		result.OpenAIBaseURL = firstSet(opt.OpenAIBaseURL, result.OpenAIBaseURL)
		result.DefaultModel = firstSet(opt.DefaultModel, result.DefaultModel)
		result.DefaultModelProvider = firstSet(opt.DefaultModelProvider, result.DefaultModelProvider)
		result.DatasetToolRepo = firstSet(opt.DatasetToolRepo, result.DatasetToolRepo)
		result.WorkspaceTool = firstSet(opt.WorkspaceTool, result.WorkspaceTool)
		result.Env = append(result.Env, opt.Env...)
	}
	return result
}

func firstSet[T comparable](in ...T) T {
	var result T
	for _, i := range in {
		if i != result {
			return i
		}
	}

	return result
}

// Options represents options for the gptscript tool or file.
type Options struct {
	GlobalOptions `json:",inline"`

	DisableCache        bool     `json:"disableCache"`
	Confirm             bool     `json:"confirm"`
	Input               string   `json:"input"`
	SubTool             string   `json:"subTool"`
	Workspace           string   `json:"workspace"`
	ChatState           string   `json:"chatState"`
	IncludeEvents       bool     `json:"includeEvents"`
	Prompt              bool     `json:"prompt"`
	CredentialOverrides []string `json:"credentialOverrides"`
	CredentialContexts  []string `json:"credentialContexts"`
	Location            string   `json:"location"`
	ForceSequential     bool     `json:"forceSequential"`
}
