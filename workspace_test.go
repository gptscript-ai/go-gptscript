package gptscript

import (
	"bytes"
	"context"
	"testing"
)

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

func TestCreateDirectory(t *testing.T) {
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

	err = g.CreateDirectoryInWorkspace(context.Background(), id, "test")
	if err != nil {
		t.Fatalf("Error creating directory: %v", err)
	}

	err = g.DeleteDirectoryInWorkspace(context.Background(), id, "test")
	if err != nil {
		t.Errorf("Error listing files: %v", err)
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

	err = g.WriteFileInWorkspace(context.Background(), id, "test.txt", []byte("test"))
	if err != nil {
		t.Fatalf("Error creating file: %v", err)
	}

	content, err := g.ReadFileInWorkspace(context.Background(), id, "test.txt")
	if err != nil {
		t.Errorf("Error reading file: %v", err)
	}

	if !bytes.Equal(content, []byte("test")) {
		t.Errorf("Unexpected content: %s", content)
	}

	err = g.DeleteFileInWorkspace(context.Background(), id, "test.txt")
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

	err = g.CreateDirectoryInWorkspace(context.Background(), id, "test")
	if err != nil {
		t.Fatalf("Error creating directory: %v", err)
	}

	err = g.WriteFileInWorkspace(context.Background(), id, "test/test1.txt", []byte("hello1"))
	if err != nil {
		t.Fatalf("Error creating file: %v", err)
	}

	err = g.WriteFileInWorkspace(context.Background(), id, "test1/test2.txt", []byte("hello2"), CreateFileInWorkspaceOptions{CreateDirs: true})
	if err != nil {
		t.Fatalf("Error creating file: %v", err)
	}

	err = g.WriteFileInWorkspace(context.Background(), id, "test1/test2.txt", []byte("hello-2"), CreateFileInWorkspaceOptions{MustNotExist: true})
	if err == nil {
		t.Fatalf("Expected error creating file that must not exist")
	}

	err = g.WriteFileInWorkspace(context.Background(), id, "test1/test3.txt", []byte("hello3"), CreateFileInWorkspaceOptions{WithoutCreate: true})
	if err == nil {
		t.Fatalf("Expected error creating file that doesn't exist")
	}

	err = g.WriteFileInWorkspace(context.Background(), id, ".hidden.txt", []byte("hidden"))
	if err != nil {
		t.Fatalf("Error creating hidden file: %v", err)
	}

	// List all files
	content, err := g.ListFilesInWorkspace(context.Background(), id)
	if err != nil {
		t.Fatalf("Error listing files: %v", err)
	}

	if content.ID != id {
		t.Errorf("Unexpected ID: %s", content.ID)
	}

	if content.Path != "" {
		t.Errorf("Unexpected path: %s", content.Path)
	}

	if content.FileName != "" {
		t.Errorf("Unexpected filename: %s", content.FileName)
	}

	if len(content.Children) != 3 {
		t.Errorf("Unexpected number of files: %d", len(content.Children))
	}

	// List files in subdirectory
	content, err = g.ListFilesInWorkspace(context.Background(), id, ListFilesInWorkspaceOptions{SubDir: "test1"})
	if err != nil {
		t.Fatalf("Error listing files: %v", err)
	}

	if len(content.Children) != 1 {
		t.Errorf("Unexpected number of files: %d", len(content.Children))
	}

	// Exclude hidden files
	content, err = g.ListFilesInWorkspace(context.Background(), id, ListFilesInWorkspaceOptions{ExcludeHidden: true})
	if err != nil {
		t.Fatalf("Error listing files: %v", err)
	}

	if len(content.Children) != 2 {
		t.Errorf("Unexpected number of files when listing without hidden: %d", len(content.Children))
	}

	// List non-recursive
	content, err = g.ListFilesInWorkspace(context.Background(), id, ListFilesInWorkspaceOptions{NonRecursive: true})
	if err != nil {
		t.Fatalf("Error listing files: %v", err)
	}

	if len(content.Children) != 1 {
		t.Errorf("Unexpected number of files when listing non-recursive: %d", len(content.Children))
	}
}
