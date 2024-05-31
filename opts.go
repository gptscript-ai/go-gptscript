package gptscript

// Options represents options for the gptscript tool or file.
type Options struct {
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
