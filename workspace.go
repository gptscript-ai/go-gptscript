package gptscript

import (
	"context"
	"os"
	"strings"
)

func (g *GPTScript) CreateWorkspace(ctx context.Context, providerType string, fromWorkspaces ...string) (string, error) {
	out, err := g.runBasicCommand(ctx, "workspaces/create", map[string]any{
		"providerType":     providerType,
		"fromWorkspaceIDs": fromWorkspaces,
		"workspaceTool":    g.globalOpts.WorkspaceTool,
		"env":              g.globalOpts.Env,
	})
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(out), nil
}

type DeleteWorkspaceOptions struct {
	WorkspaceID string
}

func (g *GPTScript) DeleteWorkspace(ctx context.Context, opts ...DeleteWorkspaceOptions) error {
	var opt DeleteWorkspaceOptions
	for _, o := range opts {
		if o.WorkspaceID != "" {
			opt.WorkspaceID = o.WorkspaceID
		}
	}

	if opt.WorkspaceID == "" {
		opt.WorkspaceID = os.Getenv("GPTSCRIPT_WORKSPACE_ID")
	}

	_, err := g.runBasicCommand(ctx, "workspaces/delete", map[string]any{
		"id":            opt.WorkspaceID,
		"workspaceTool": g.globalOpts.WorkspaceTool,
		"env":           g.globalOpts.Env,
	})

	return err
}

type ListFilesInWorkspaceOptions struct {
	WorkspaceID string
	Prefix      string
}

func (g *GPTScript) ListFilesInWorkspace(ctx context.Context, opts ...ListFilesInWorkspaceOptions) ([]string, error) {
	var opt ListFilesInWorkspaceOptions
	for _, o := range opts {
		if o.Prefix != "" {
			opt.Prefix = o.Prefix
		}
		if o.WorkspaceID != "" {
			opt.WorkspaceID = o.WorkspaceID
		}
	}

	if opt.WorkspaceID == "" {
		opt.WorkspaceID = os.Getenv("GPTSCRIPT_WORKSPACE_ID")
	}

	out, err := g.runBasicCommand(ctx, "workspaces/list", map[string]any{
		"id":            opt.WorkspaceID,
		"prefix":        opt.Prefix,
		"workspaceTool": g.globalOpts.WorkspaceTool,
		"env":           g.globalOpts.Env,
	})
	if err != nil {
		return nil, err
	}

	return strings.Split(strings.TrimSpace(out), "\n"), nil
}

type RemoveAllOptions struct {
	WorkspaceID string
	WithPrefix  string
}

func (g *GPTScript) RemoveAll(ctx context.Context, opts ...RemoveAllOptions) error {
	var opt RemoveAllOptions
	for _, o := range opts {
		if o.WithPrefix != "" {
			opt.WithPrefix = o.WithPrefix
		}
		if o.WorkspaceID != "" {
			opt.WorkspaceID = o.WorkspaceID
		}
	}

	if opt.WorkspaceID == "" {
		opt.WorkspaceID = os.Getenv("GPTSCRIPT_WORKSPACE_ID")
	}

	_, err := g.runBasicCommand(ctx, "workspaces/remove-all-with-prefix", map[string]any{
		"id":            opt.WorkspaceID,
		"prefix":        opt.WithPrefix,
		"workspaceTool": g.globalOpts.WorkspaceTool,
		"env":           g.globalOpts.Env,
	})

	return err
}

type WriteFileInWorkspaceOptions struct {
	WorkspaceID string
}

func (g *GPTScript) WriteFileInWorkspace(ctx context.Context, filePath string, contents []byte, opts ...WriteFileInWorkspaceOptions) error {
	var opt WriteFileInWorkspaceOptions
	for _, o := range opts {
		if o.WorkspaceID != "" {
			opt.WorkspaceID = o.WorkspaceID
		}
	}

	if opt.WorkspaceID == "" {
		opt.WorkspaceID = os.Getenv("GPTSCRIPT_WORKSPACE_ID")
	}

	_, err := g.runBasicCommand(ctx, "workspaces/write-file", map[string]any{
		"id":            opt.WorkspaceID,
		"contents":      contents,
		"filePath":      filePath,
		"workspaceTool": g.globalOpts.WorkspaceTool,
		"env":           g.globalOpts.Env,
	})

	return err
}

type DeleteFileInWorkspaceOptions struct {
	WorkspaceID string
}

func (g *GPTScript) DeleteFileInWorkspace(ctx context.Context, filePath string, opts ...DeleteFileInWorkspaceOptions) error {
	var opt DeleteFileInWorkspaceOptions
	for _, o := range opts {
		if o.WorkspaceID != "" {
			opt.WorkspaceID = o.WorkspaceID
		}
	}

	if opt.WorkspaceID == "" {
		opt.WorkspaceID = os.Getenv("GPTSCRIPT_WORKSPACE_ID")
	}

	_, err := g.runBasicCommand(ctx, "workspaces/delete-file", map[string]any{
		"id":            opt.WorkspaceID,
		"filePath":      filePath,
		"workspaceTool": g.globalOpts.WorkspaceTool,
		"env":           g.globalOpts.Env,
	})

	return err
}

type ReadFileInWorkspaceOptions struct {
	WorkspaceID string
}

func (g *GPTScript) ReadFileInWorkspace(ctx context.Context, filePath string, opts ...ReadFileInWorkspaceOptions) ([]byte, error) {
	var opt ReadFileInWorkspaceOptions
	for _, o := range opts {
		if o.WorkspaceID != "" {
			opt.WorkspaceID = o.WorkspaceID
		}
	}

	if opt.WorkspaceID == "" {
		opt.WorkspaceID = os.Getenv("GPTSCRIPT_WORKSPACE_ID")
	}

	out, err := g.runBasicCommand(ctx, "workspaces/read-file", map[string]any{
		"id":            opt.WorkspaceID,
		"filePath":      filePath,
		"workspaceTool": g.globalOpts.WorkspaceTool,
		"env":           g.globalOpts.Env,
	})
	if err != nil {
		return nil, err
	}

	return []byte(out), nil
}
