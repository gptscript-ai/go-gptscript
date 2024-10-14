package gptscript

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDatasets(t *testing.T) {
	workspace, err := os.MkdirTemp("", "go-gptscript-test")
	require.NoError(t, err)
	defer func() {
		_ = os.RemoveAll(workspace)
	}()

	// Create a dataset
	dataset, err := g.CreateDataset(context.Background(), workspace, "test-dataset", "This is a test dataset")
	require.NoError(t, err)
	require.Equal(t, "test-dataset", dataset.Name)
	require.Equal(t, "This is a test dataset", dataset.Description)
	require.Equal(t, 0, len(dataset.Elements))

	// Add an element
	elementMeta, err := g.AddDatasetElement(context.Background(), workspace, dataset.ID, "test-element", "This is a test element", "This is the content")
	require.NoError(t, err)
	require.Equal(t, "test-element", elementMeta.Name)
	require.Equal(t, "This is a test element", elementMeta.Description)

	// Get the element
	element, err := g.GetDatasetElement(context.Background(), workspace, dataset.ID, "test-element")
	require.NoError(t, err)
	require.Equal(t, "test-element", element.Name)
	require.Equal(t, "This is a test element", element.Description)
	require.Equal(t, "This is the content", element.Contents)

	// List elements in the dataset
	elements, err := g.ListDatasetElements(context.Background(), workspace, dataset.ID)
	require.NoError(t, err)
	require.Equal(t, 1, len(elements))
	require.Equal(t, "test-element", elements[0].Name)
	require.Equal(t, "This is a test element", elements[0].Description)

	// List datasets
	datasets, err := g.ListDatasets(context.Background(), workspace)
	require.NoError(t, err)
	require.Equal(t, 1, len(datasets))
	require.Equal(t, "test-dataset", datasets[0].Name)
	require.Equal(t, "This is a test dataset", datasets[0].Description)
	require.Equal(t, dataset.ID, datasets[0].ID)
}
