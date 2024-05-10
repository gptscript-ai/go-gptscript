package gptscript

import (
	"fmt"
)

// Opts represents options for the gptscript tool or file.
type Opts struct {
	Input         string `json:"input"`
	DisableCache  bool   `json:"disableCache"`
	CacheDir      string `json:"cacheDir"`
	Quiet         bool   `json:"quiet"`
	Chdir         string `json:"chdir"`
	SubTool       string `json:"subTool"`
	Workspace     string `json:"workspace"`
	ChatState     string `json:"chatState"`
	IncludeEvents bool   `json:"includeEvents"`
}

func (o Opts) toArgs() []string {
	var args []string
	if o.DisableCache {
		args = append(args, "--disable-cache")
	}
	if o.CacheDir != "" {
		args = append(args, "--cache-dir="+o.CacheDir)
	}
	if o.Chdir != "" {
		args = append(args, "--chdir="+o.Chdir)
	}
	if o.SubTool != "" {
		args = append(args, "--sub-tool="+o.SubTool)
	}
	if o.Workspace != "" {
		args = append(args, "--workspace="+o.Workspace)
	}
	return append(args, "--quiet="+fmt.Sprint(o.Quiet))
}
