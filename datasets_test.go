package gptscript

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDatasets(t *testing.T) {
	workspaceID, err := g.CreateWorkspace(context.Background(), "directory")
	require.NoError(t, err)

	require.NoError(t, os.Setenv("GPTSCRIPT_WORKSPACE_ID", workspaceID))

	defer func() {
		_ = g.DeleteWorkspace(context.Background(), workspaceID)
	}()

	datasetID, err := g.CreateDatasetWithElements(context.Background(), []DatasetElement{
		{
			DatasetElementMeta: DatasetElementMeta{
				Name:        "test-element-1",
				Description: "This is a test element 1",
			},
			Contents: "This is the content 1",
		},
	})
	require.NoError(t, err)

	// Add two more elements
	_, err = g.AddDatasetElements(context.Background(), datasetID, []DatasetElement{
		{
			DatasetElementMeta: DatasetElementMeta{
				Name:        "test-element-2",
				Description: "This is a test element 2",
			},
			Contents: "This is the content 2",
		},
		{
			DatasetElementMeta: DatasetElementMeta{
				Name:        "test-element-3",
				Description: "This is a test element 3",
			},
			Contents: "This is the content 3",
		},
		{
			DatasetElementMeta: DatasetElementMeta{
				Name:        "binary-element",
				Description: "this element has binary contents",
			},
			BinaryContents: []byte("binary contents"),
		},
	})
	require.NoError(t, err)

	// Get the first element
	element, err := g.GetDatasetElement(context.Background(), datasetID, "test-element-1")
	require.NoError(t, err)
	require.Equal(t, "test-element-1", element.Name)
	require.Equal(t, "This is a test element 1", element.Description)
	require.Equal(t, "This is the content 1", element.Contents)

	// Get the third element
	element, err = g.GetDatasetElement(context.Background(), datasetID, "test-element-3")
	require.NoError(t, err)
	require.Equal(t, "test-element-3", element.Name)
	require.Equal(t, "This is a test element 3", element.Description)
	require.Equal(t, "This is the content 3", element.Contents)

	// Get the binary element
	element, err = g.GetDatasetElement(context.Background(), datasetID, "binary-element")
	require.NoError(t, err)
	require.Equal(t, "binary-element", element.Name)
	require.Equal(t, "this element has binary contents", element.Description)
	require.Equal(t, []byte("binary contents"), element.BinaryContents)

	// List elements in the dataset
	elements, err := g.ListDatasetElements(context.Background(), datasetID)
	require.NoError(t, err)
	require.Equal(t, 4, len(elements))

	// List datasets
	datasetIDs, err := g.ListDatasets(context.Background())
	require.NoError(t, err)
	require.Equal(t, 1, len(datasetIDs))
	require.Equal(t, datasetID, datasetIDs[0])
}
