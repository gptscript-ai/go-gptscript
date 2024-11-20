package gptscript

import (
	"bytes"
	"context"
	"errors"
	"os"
	"testing"
)

func TestWorkspaceIDRequiredForDelete(t *testing.T) {
	if err := g.DeleteWorkspace(context.Background(), ""); err == nil {
		t.Error("Expected error but got nil")
	}
}

func TestCreateAndDeleteWorkspace(t *testing.T) {
	id, err := g.CreateWorkspace(context.Background(), "directory")
	if err != nil {
		t.Fatalf("Error creating workspace: %v", err)
	}

	err = g.DeleteWorkspace(context.Background(), id)
	if err != nil {
		t.Errorf("Error deleting workspace: %v", err)
	}
}

func TestWriteReadAndDeleteFileFromWorkspace(t *testing.T) {
	id, err := g.CreateWorkspace(context.Background(), "directory")
	if err != nil {
		t.Fatalf("Error creating workspace: %v", err)
	}

	t.Cleanup(func() {
		err := g.DeleteWorkspace(context.Background(), id)
		if err != nil {
			t.Errorf("Error deleting workspace: %v", err)
		}
	})

	err = g.WriteFileInWorkspace(context.Background(), "test.txt", []byte("test"), WriteFileInWorkspaceOptions{WorkspaceID: id})
	if err != nil {
		t.Fatalf("Error creating file: %v", err)
	}

	content, err := g.ReadFileInWorkspace(context.Background(), "test.txt", ReadFileInWorkspaceOptions{WorkspaceID: id})
	if err != nil {
		t.Errorf("Error reading file: %v", err)
	}

	if !bytes.Equal(content, []byte("test")) {
		t.Errorf("Unexpected content: %s", content)
	}

	// Stat the file to ensure it exists
	fileInfo, err := g.StatFileInWorkspace(context.Background(), "test.txt", StatFileInWorkspaceOptions{WorkspaceID: id})
	if err != nil {
		t.Errorf("Error statting file: %v", err)
	}

	if fileInfo.WorkspaceID != id {
		t.Errorf("Unexpected file workspace ID: %v", fileInfo.WorkspaceID)
	}

	if fileInfo.Name != "test.txt" {
		t.Errorf("Unexpected file name: %s", fileInfo.Name)
	}

	if fileInfo.Size != 4 {
		t.Errorf("Unexpected file size: %d", fileInfo.Size)
	}

	if fileInfo.ModTime.IsZero() {
		t.Errorf("Unexpected file mod time: %v", fileInfo.ModTime)
	}

	if fileInfo.MimeType != "text/plain" {
		t.Errorf("Unexpected file mime type: %s", fileInfo.MimeType)
	}

	// Ensure we get the error we expect when trying to read a non-existent file
	_, err = g.ReadFileInWorkspace(context.Background(), "test1.txt", ReadFileInWorkspaceOptions{WorkspaceID: id})
	if nf := (*NotFoundInWorkspaceError)(nil); !errors.As(err, &nf) {
		t.Errorf("Unexpected error reading non-existent file: %v", err)
	}

	err = g.DeleteFileInWorkspace(context.Background(), "test.txt", DeleteFileInWorkspaceOptions{WorkspaceID: id})
	if err != nil {
		t.Errorf("Error deleting file: %v", err)
	}
}

func TestLsComplexWorkspace(t *testing.T) {
	id, err := g.CreateWorkspace(context.Background(), "directory")
	if err != nil {
		t.Fatalf("Error creating workspace: %v", err)
	}

	t.Cleanup(func() {
		err := g.DeleteWorkspace(context.Background(), id)
		if err != nil {
			t.Errorf("Error deleting workspace: %v", err)
		}
	})

	err = g.WriteFileInWorkspace(context.Background(), "test/test1.txt", []byte("hello1"), WriteFileInWorkspaceOptions{WorkspaceID: id})
	if err != nil {
		t.Fatalf("Error creating file: %v", err)
	}

	err = g.WriteFileInWorkspace(context.Background(), "test1/test2.txt", []byte("hello2"), WriteFileInWorkspaceOptions{WorkspaceID: id})
	if err != nil {
		t.Fatalf("Error creating file: %v", err)
	}

	err = g.WriteFileInWorkspace(context.Background(), "test1/test3.txt", []byte("hello3"), WriteFileInWorkspaceOptions{WorkspaceID: id})
	if err != nil {
		t.Fatalf("Error creating file: %v", err)
	}

	err = g.WriteFileInWorkspace(context.Background(), ".hidden.txt", []byte("hidden"), WriteFileInWorkspaceOptions{WorkspaceID: id})
	if err != nil {
		t.Fatalf("Error creating hidden file: %v", err)
	}

	// List all files
	content, err := g.ListFilesInWorkspace(context.Background(), ListFilesInWorkspaceOptions{WorkspaceID: id})
	if err != nil {
		t.Fatalf("Error listing files: %v", err)
	}

	if len(content) != 4 {
		t.Errorf("Unexpected number of files: %d", len(content))
	}

	// List files in subdirectory
	content, err = g.ListFilesInWorkspace(context.Background(), ListFilesInWorkspaceOptions{WorkspaceID: id, Prefix: "test1"})
	if err != nil {
		t.Fatalf("Error listing files: %v", err)
	}

	if len(content) != 2 {
		t.Errorf("Unexpected number of files: %d", len(content))
	}

	// Remove all files with test1 prefix
	err = g.RemoveAll(context.Background(), RemoveAllOptions{WorkspaceID: id, WithPrefix: "test1"})
	if err != nil {
		t.Fatalf("Error removing files: %v", err)
	}

	// List files in subdirectory
	content, err = g.ListFilesInWorkspace(context.Background(), ListFilesInWorkspaceOptions{WorkspaceID: id})
	if err != nil {
		t.Fatalf("Error listing files: %v", err)
	}

	if len(content) != 2 {
		t.Errorf("Unexpected number of files: %d", len(content))
	}
}

func TestCreateAndDeleteWorkspaceS3(t *testing.T) {
	if os.Getenv("AWS_ACCESS_KEY_ID") == "" || os.Getenv("AWS_SECRET_ACCESS_KEY") == "" || os.Getenv("WORKSPACE_PROVIDER_S3_BUCKET") == "" {
		t.Skip("Skipping test because AWS credentials are not set")
	}

	id, err := g.CreateWorkspace(context.Background(), "s3")
	if err != nil {
		t.Fatalf("Error creating workspace: %v", err)
	}

	err = g.DeleteWorkspace(context.Background(), id)
	if err != nil {
		t.Errorf("Error deleting workspace: %v", err)
	}
}

func TestWriteReadAndDeleteFileFromWorkspaceS3(t *testing.T) {
	if os.Getenv("AWS_ACCESS_KEY_ID") == "" || os.Getenv("AWS_SECRET_ACCESS_KEY") == "" || os.Getenv("WORKSPACE_PROVIDER_S3_BUCKET") == "" {
		t.Skip("Skipping test because AWS credentials are not set")
	}

	id, err := g.CreateWorkspace(context.Background(), "s3")
	if err != nil {
		t.Fatalf("Error creating workspace: %v", err)
	}

	t.Cleanup(func() {
		err := g.DeleteWorkspace(context.Background(), id)
		if err != nil {
			t.Errorf("Error deleting workspace: %v", err)
		}
	})

	err = g.WriteFileInWorkspace(context.Background(), "test.txt", []byte("test"), WriteFileInWorkspaceOptions{WorkspaceID: id})
	if err != nil {
		t.Fatalf("Error creating file: %v", err)
	}

	content, err := g.ReadFileInWorkspace(context.Background(), "test.txt", ReadFileInWorkspaceOptions{WorkspaceID: id})
	if err != nil {
		t.Errorf("Error reading file: %v", err)
	}

	if !bytes.Equal(content, []byte("test")) {
		t.Errorf("Unexpected content: %s", content)
	}

	// Stat the file to ensure it exists
	fileInfo, err := g.StatFileInWorkspace(context.Background(), "test.txt", StatFileInWorkspaceOptions{WorkspaceID: id})
	if err != nil {
		t.Errorf("Error statting file: %v", err)
	}

	if fileInfo.WorkspaceID != id {
		t.Errorf("Unexpected file workspace ID: %v", fileInfo.WorkspaceID)
	}

	if fileInfo.Name != "test.txt" {
		t.Errorf("Unexpected file name: %s", fileInfo.Name)
	}

	if fileInfo.Size != 4 {
		t.Errorf("Unexpected file size: %d", fileInfo.Size)
	}

	if fileInfo.ModTime.IsZero() {
		t.Errorf("Unexpected file mod time: %v", fileInfo.ModTime)
	}

	if fileInfo.MimeType != "text/plain" {
		t.Errorf("Unexpected file mime type: %s", fileInfo.MimeType)
	}

	// Ensure we get the error we expect when trying to read a non-existent file
	_, err = g.ReadFileInWorkspace(context.Background(), "test1.txt", ReadFileInWorkspaceOptions{WorkspaceID: id})
	if nf := (*NotFoundInWorkspaceError)(nil); !errors.As(err, &nf) {
		t.Errorf("Unexpected error reading non-existent file: %v", err)
	}

	err = g.DeleteFileInWorkspace(context.Background(), "test.txt", DeleteFileInWorkspaceOptions{WorkspaceID: id})
	if err != nil {
		t.Errorf("Error deleting file: %v", err)
	}
}

func TestLsComplexWorkspaceS3(t *testing.T) {
	if os.Getenv("AWS_ACCESS_KEY_ID") == "" || os.Getenv("AWS_SECRET_ACCESS_KEY") == "" || os.Getenv("WORKSPACE_PROVIDER_S3_BUCKET") == "" {
		t.Skip("Skipping test because AWS credentials are not set")
	}

	id, err := g.CreateWorkspace(context.Background(), "s3")
	if err != nil {
		t.Fatalf("Error creating workspace: %v", err)
	}

	t.Cleanup(func() {
		err := g.DeleteWorkspace(context.Background(), id)
		if err != nil {
			t.Errorf("Error deleting workspace: %v", err)
		}
	})

	err = g.WriteFileInWorkspace(context.Background(), "test/test1.txt", []byte("hello1"), WriteFileInWorkspaceOptions{WorkspaceID: id})
	if err != nil {
		t.Fatalf("Error creating file: %v", err)
	}

	err = g.WriteFileInWorkspace(context.Background(), "test1/test2.txt", []byte("hello2"), WriteFileInWorkspaceOptions{WorkspaceID: id})
	if err != nil {
		t.Fatalf("Error creating file: %v", err)
	}

	err = g.WriteFileInWorkspace(context.Background(), "test1/test3.txt", []byte("hello3"), WriteFileInWorkspaceOptions{WorkspaceID: id})
	if err != nil {
		t.Fatalf("Error creating file: %v", err)
	}

	err = g.WriteFileInWorkspace(context.Background(), ".hidden.txt", []byte("hidden"), WriteFileInWorkspaceOptions{WorkspaceID: id})
	if err != nil {
		t.Fatalf("Error creating hidden file: %v", err)
	}

	// List all files
	content, err := g.ListFilesInWorkspace(context.Background(), ListFilesInWorkspaceOptions{WorkspaceID: id})
	if err != nil {
		t.Fatalf("Error listing files: %v", err)
	}

	if len(content) != 4 {
		t.Errorf("Unexpected number of files: %d", len(content))
	}

	// List files in subdirectory
	content, err = g.ListFilesInWorkspace(context.Background(), ListFilesInWorkspaceOptions{WorkspaceID: id, Prefix: "test1"})
	if err != nil {
		t.Fatalf("Error listing files: %v", err)
	}

	if len(content) != 2 {
		t.Errorf("Unexpected number of files: %d", len(content))
	}

	// Remove all files with test1 prefix
	err = g.RemoveAll(context.Background(), RemoveAllOptions{WorkspaceID: id, WithPrefix: "test1"})
	if err != nil {
		t.Fatalf("Error removing files: %v", err)
	}

	// List files in subdirectory
	content, err = g.ListFilesInWorkspace(context.Background(), ListFilesInWorkspaceOptions{WorkspaceID: id})
	if err != nil {
		t.Fatalf("Error listing files: %v", err)
	}

	if len(content) != 2 {
		t.Errorf("Unexpected number of files: %d", len(content))
	}
}
