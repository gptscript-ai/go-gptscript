package gptscript

import (
	"context"
	"encoding/base64"
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

func (g *GPTScript) DeleteWorkspace(ctx context.Context, workspaceID string) error {
	_, err := g.runBasicCommand(ctx, "workspaces/delete", map[string]any{
		"id":            workspaceID,
		"workspaceTool": g.globalOpts.WorkspaceTool,
		"env":           g.globalOpts.Env,
	})

	return err
}

type ListFilesInWorkspaceOptions struct {
	Prefix string
}

func (g *GPTScript) ListFilesInWorkspace(ctx context.Context, workspaceID string, opts ...ListFilesInWorkspaceOptions) ([]string, error) {
	var opt ListFilesInWorkspaceOptions
	for _, o := range opts {
		if o.Prefix != "" {
			opt.Prefix = o.Prefix
		}
	}

	out, err := g.runBasicCommand(ctx, "workspaces/list", map[string]any{
		"id":            workspaceID,
		"prefix":        opt.Prefix,
		"workspaceTool": g.globalOpts.WorkspaceTool,
		"env":           g.globalOpts.Env,
	})
	if err != nil {
		return nil, err
	}

	// The first line of the output is the workspace ID, ignore it.
	return strings.Split(strings.TrimSpace(out), "\n")[1:], nil
}

func (g *GPTScript) RemoveAllWithPrefix(ctx context.Context, workspaceID, prefix string) error {
	_, err := g.runBasicCommand(ctx, "workspaces/remove-all-with-prefix", map[string]any{
		"id":            workspaceID,
		"prefix":        prefix,
		"workspaceTool": g.globalOpts.WorkspaceTool,
		"env":           g.globalOpts.Env,
	})

	return err
}

func (g *GPTScript) WriteFileInWorkspace(ctx context.Context, workspaceID, filePath string, contents []byte) error {
	_, err := g.runBasicCommand(ctx, "workspaces/write-file", map[string]any{
		"id":                 workspaceID,
		"contents":           base64.StdEncoding.EncodeToString(contents),
		"filePath":           filePath,
		"workspaceTool":      g.globalOpts.WorkspaceTool,
		"base64EncodedInput": true,
		"env":                g.globalOpts.Env,
	})

	return err
}

func (g *GPTScript) DeleteFileInWorkspace(ctx context.Context, workspaceID, filePath string) error {
	_, err := g.runBasicCommand(ctx, "workspaces/delete-file", map[string]any{
		"id":            workspaceID,
		"filePath":      filePath,
		"workspaceTool": g.globalOpts.WorkspaceTool,
		"env":           g.globalOpts.Env,
	})

	return err
}

func (g *GPTScript) ReadFileInWorkspace(ctx context.Context, workspaceID, filePath string) ([]byte, error) {
	out, err := g.runBasicCommand(ctx, "workspaces/read-file", map[string]any{
		"id":                 workspaceID,
		"filePath":           filePath,
		"workspaceTool":      g.globalOpts.WorkspaceTool,
		"base64EncodeOutput": true,
		"env":                g.globalOpts.Env,
	})
	if err != nil {
		return nil, err
	}

	return base64.StdEncoding.DecodeString(out)
}
