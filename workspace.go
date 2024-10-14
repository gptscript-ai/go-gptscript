package gptscript

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"strings"
)

func (g *GPTScript) CreateWorkspace(ctx context.Context, providerType string) (string, error) {
	out, err := g.runBasicCommand(ctx, "workspaces/create", map[string]any{
		"provider":      providerType,
		"workspaceTool": g.globalOpts.WorkspaceTool,
	})
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(out), nil
}

type DeleteWorkspaceOptions struct {
	IgnoreNotFound bool
}

func (g *GPTScript) DeleteWorkspace(ctx context.Context, workspaceID string, opts ...DeleteWorkspaceOptions) error {
	var opt DeleteWorkspaceOptions
	for _, o := range opts {
		opt.IgnoreNotFound = opt.IgnoreNotFound || o.IgnoreNotFound
	}
	_, err := g.runBasicCommand(ctx, "workspaces/delete", map[string]any{
		"id":             workspaceID,
		"ignoreNotFound": opt.IgnoreNotFound,
		"workspaceTool":  g.globalOpts.WorkspaceTool,
	})

	return err
}

type CreateDirectoryInWorkspaceOptions struct {
	IgnoreExists bool
}

func (g *GPTScript) CreateDirectoryInWorkspace(ctx context.Context, workspaceID, dir string, opts ...CreateDirectoryInWorkspaceOptions) error {
	var opt CreateDirectoryInWorkspaceOptions
	for _, o := range opts {
		opt.IgnoreExists = opt.IgnoreExists || o.IgnoreExists
	}

	_, err := g.runBasicCommand(ctx, "workspaces/mkdir", map[string]any{
		"id":            workspaceID,
		"directoryName": dir,
		"ignoreExists":  opt.IgnoreExists,
		"workspaceTool": g.globalOpts.WorkspaceTool,
	})

	return err
}

type DeleteDirectoryInWorkspaceOptions struct {
	IgnoreNotFound bool
	MustBeEmpty    bool
}

func (g *GPTScript) DeleteDirectoryInWorkspace(ctx context.Context, workspaceID, dir string, opts ...DeleteDirectoryInWorkspaceOptions) error {
	var opt DeleteDirectoryInWorkspaceOptions
	for _, o := range opts {
		o.IgnoreNotFound = opt.IgnoreNotFound || o.IgnoreNotFound
		o.MustBeEmpty = opt.MustBeEmpty || o.MustBeEmpty
	}

	_, err := g.runBasicCommand(ctx, "workspaces/rmdir", map[string]any{
		"id":             workspaceID,
		"directoryName":  dir,
		"ignoreNotFound": opt.IgnoreNotFound,
		"mustBeEmpty":    opt.MustBeEmpty,
		"workspaceTool":  g.globalOpts.WorkspaceTool,
	})

	return err
}

type ListFilesInWorkspaceOptions struct {
	SubDir        string
	NonRecursive  bool
	ExcludeHidden bool
}

type WorkspaceContent struct {
	ID, Path, FileName string
	Children           []WorkspaceContent
}

func (g *GPTScript) ListFilesInWorkspace(ctx context.Context, workspaceID string, opts ...ListFilesInWorkspaceOptions) (*WorkspaceContent, error) {
	var opt ListFilesInWorkspaceOptions
	for _, o := range opts {
		if o.SubDir != "" {
			opt.SubDir = o.SubDir
		}
		opt.NonRecursive = opt.NonRecursive || o.NonRecursive
		opt.ExcludeHidden = opt.ExcludeHidden || o.ExcludeHidden
	}

	out, err := g.runBasicCommand(ctx, "workspaces/list", map[string]any{
		"id":            workspaceID,
		"subDir":        opt.SubDir,
		"excludeHidden": opt.ExcludeHidden,
		"nonRecursive":  opt.NonRecursive,
		"workspaceTool": g.globalOpts.WorkspaceTool,
		"json":          true,
	})
	if err != nil {
		return nil, err
	}

	var content []WorkspaceContent
	err = json.Unmarshal([]byte(out), &content)
	if err != nil {
		return nil, err
	}

	if len(content) == 0 {
		return &WorkspaceContent{ID: workspaceID}, nil
	}

	return &content[0], nil
}

type CreateFileInWorkspaceOptions struct {
	MustNotExist  bool
	WithoutCreate bool
	CreateDirs    bool
}

func (g *GPTScript) WriteFileInWorkspace(ctx context.Context, workspaceID, filePath string, contents []byte, opts ...CreateFileInWorkspaceOptions) error {
	var opt CreateFileInWorkspaceOptions
	for _, o := range opts {
		opt.MustNotExist = opt.MustNotExist || o.MustNotExist
		opt.WithoutCreate = opt.WithoutCreate || o.WithoutCreate
		opt.CreateDirs = opt.CreateDirs || o.CreateDirs
	}

	_, err := g.runBasicCommand(ctx, "workspaces/write-file", map[string]any{
		"id":                 workspaceID,
		"contents":           base64.StdEncoding.EncodeToString(contents),
		"filePath":           filePath,
		"mustNotExist":       opt.MustNotExist,
		"withoutCreate":      opt.WithoutCreate,
		"createDirs":         opt.CreateDirs,
		"workspaceTool":      g.globalOpts.WorkspaceTool,
		"base64EncodedInput": true,
	})

	return err
}

type DeleteFileInWorkspaceOptions struct {
	IgnoreNotFound bool
}

func (g *GPTScript) DeleteFileInWorkspace(ctx context.Context, workspaceID, filePath string, opts ...DeleteFileInWorkspaceOptions) error {
	var opt DeleteFileInWorkspaceOptions
	for _, o := range opts {
		opt.IgnoreNotFound = opt.IgnoreNotFound || o.IgnoreNotFound
	}

	_, err := g.runBasicCommand(ctx, "workspaces/delete-file", map[string]any{
		"id":             workspaceID,
		"filePath":       filePath,
		"ignoreNotFound": opt.IgnoreNotFound,
		"workspaceTool":  g.globalOpts.WorkspaceTool,
	})

	return err
}

func (g *GPTScript) ReadFileInWorkspace(ctx context.Context, workspaceID, filePath string) ([]byte, error) {
	out, err := g.runBasicCommand(ctx, "workspaces/read-file", map[string]any{
		"id":                 workspaceID,
		"filePath":           filePath,
		"workspaceTool":      g.globalOpts.WorkspaceTool,
		"base64EncodeOutput": true,
	})
	if err != nil {
		return nil, err
	}

	return base64.StdEncoding.DecodeString(out)
}
