package gptscript

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDatasets(t *testing.T) {
	workspaceID, err := g.CreateWorkspace(context.Background(), "directory")
	require.NoError(t, err)

	defer func() {
		_ = g.DeleteWorkspace(context.Background(), workspaceID)
	}()

	// Create a dataset
	dataset, err := g.CreateDataset(context.Background(), workspaceID, "test-dataset", "This is a test dataset")
	require.NoError(t, err)
	require.Equal(t, "test-dataset", dataset.Name)
	require.Equal(t, "This is a test dataset", dataset.Description)
	require.Equal(t, 0, len(dataset.Elements))

	// Add an element
	elementMeta, err := g.AddDatasetElement(context.Background(), workspaceID, dataset.ID, "test-element", "This is a test element", []byte("This is the content"))
	require.NoError(t, err)
	require.Equal(t, "test-element", elementMeta.Name)
	require.Equal(t, "This is a test element", elementMeta.Description)

	// Add two more
	err = g.AddDatasetElements(context.Background(), workspaceID, dataset.ID, []DatasetElement{
		{
			DatasetElementMeta: DatasetElementMeta{
				Name:        "test-element-2",
				Description: "This is a test element 2",
			},
			Contents: []byte("This is the content 2"),
		},
		{
			DatasetElementMeta: DatasetElementMeta{
				Name:        "test-element-3",
				Description: "This is a test element 3",
			},
			Contents: []byte("This is the content 3"),
		},
	})
	require.NoError(t, err)

	// Get the first element
	element, err := g.GetDatasetElement(context.Background(), workspaceID, dataset.ID, "test-element")
	require.NoError(t, err)
	require.Equal(t, "test-element", element.Name)
	require.Equal(t, "This is a test element", element.Description)
	require.Equal(t, []byte("This is the content"), element.Contents)

	// Get the third element
	element, err = g.GetDatasetElement(context.Background(), workspaceID, dataset.ID, "test-element-3")
	require.NoError(t, err)
	require.Equal(t, "test-element-3", element.Name)
	require.Equal(t, "This is a test element 3", element.Description)
	require.Equal(t, []byte("This is the content 3"), element.Contents)

	// List elements in the dataset
	elements, err := g.ListDatasetElements(context.Background(), workspaceID, dataset.ID)
	require.NoError(t, err)
	require.Equal(t, 3, len(elements))

	// List datasets
	datasets, err := g.ListDatasets(context.Background(), workspaceID)
	require.NoError(t, err)
	require.Equal(t, 1, len(datasets))
	require.Equal(t, "test-dataset", datasets[0].Name)
	require.Equal(t, "This is a test dataset", datasets[0].Description)
	require.Equal(t, dataset.ID, datasets[0].ID)
}
