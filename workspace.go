package gptscript

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

type NotFoundInWorkspaceError struct {
	id   string
	name string
}

func (e *NotFoundInWorkspaceError) Error() string {
	return fmt.Sprintf("not found: %s/%s", e.id, e.name)
}

func newNotFoundInWorkspaceError(id, name string) *NotFoundInWorkspaceError {
	return &NotFoundInWorkspaceError{id: id, name: name}
}

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
	if workspaceID == "" {
		return fmt.Errorf("workspace ID cannot be empty")
	}

	_, err := g.runBasicCommand(ctx, "workspaces/delete", map[string]any{
		"id":            workspaceID,
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

	out = strings.TrimSpace(out)
	if len(out) == 0 {
		return nil, nil
	}

	var files []string
	return files, json.Unmarshal([]byte(out), &files)
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
		"contents":      base64.StdEncoding.EncodeToString(contents),
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

	if err != nil && strings.HasSuffix(err.Error(), fmt.Sprintf("not found: %s/%s", opt.WorkspaceID, filePath)) {
		return newNotFoundInWorkspaceError(opt.WorkspaceID, filePath)
	}

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
		if strings.HasSuffix(err.Error(), fmt.Sprintf("not found: %s/%s", opt.WorkspaceID, filePath)) {
			return nil, newNotFoundInWorkspaceError(opt.WorkspaceID, filePath)
		}
		return nil, err
	}

	return base64.StdEncoding.DecodeString(out)
}

type StatFileInWorkspaceOptions struct {
	WorkspaceID string
}

func (g *GPTScript) StatFileInWorkspace(ctx context.Context, filePath string, opts ...StatFileInWorkspaceOptions) (FileInfo, error) {
	var opt StatFileInWorkspaceOptions
	for _, o := range opts {
		if o.WorkspaceID != "" {
			opt.WorkspaceID = o.WorkspaceID
		}
	}

	if opt.WorkspaceID == "" {
		opt.WorkspaceID = os.Getenv("GPTSCRIPT_WORKSPACE_ID")
	}

	out, err := g.runBasicCommand(ctx, "workspaces/stat-file", map[string]any{
		"id":            opt.WorkspaceID,
		"filePath":      filePath,
		"workspaceTool": g.globalOpts.WorkspaceTool,
		"env":           g.globalOpts.Env,
	})
	if err != nil {
		if strings.HasSuffix(err.Error(), fmt.Sprintf("not found: %s/%s", opt.WorkspaceID, filePath)) {
			return FileInfo{}, newNotFoundInWorkspaceError(opt.WorkspaceID, filePath)
		}
		return FileInfo{}, err
	}

	var info FileInfo
	err = json.Unmarshal([]byte(out), &info)
	if err != nil {
		return FileInfo{}, err
	}

	return info, nil
}

type FileInfo struct {
	WorkspaceID string
	Name        string
	Size        int64
	ModTime     time.Time
	MimeType    string
}
