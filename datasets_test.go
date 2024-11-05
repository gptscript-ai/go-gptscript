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

	client, err := NewGPTScript(GlobalOptions{
		OpenAIAPIKey: os.Getenv("OPENAI_API_KEY"),
		Env:          []string{"GPTSCRIPT_WORKSPACE_ID=" + workspaceID},
	})
	require.NoError(t, err)

	defer func() {
		_ = g.DeleteWorkspace(context.Background(), workspaceID)
	}()

	datasetID, err := client.CreateDatasetWithElements(context.Background(), []DatasetElement{
		{
			DatasetElementMeta: DatasetElementMeta{
				Name:        "test-element-1",
				Description: "This is a test element 1",
			},
			Contents: "This is the content 1",
		},
	}, DatasetOptions{
		Name:        "test-dataset",
		Description: "this is a test dataset",
	})
	require.NoError(t, err)

	// Add three more elements
	_, err = client.AddDatasetElements(context.Background(), datasetID, []DatasetElement{
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
	element, err := client.GetDatasetElement(context.Background(), datasetID, "test-element-1")
	require.NoError(t, err)
	require.Equal(t, "test-element-1", element.Name)
	require.Equal(t, "This is a test element 1", element.Description)
	require.Equal(t, "This is the content 1", element.Contents)

	// Get the third element
	element, err = client.GetDatasetElement(context.Background(), datasetID, "test-element-3")
	require.NoError(t, err)
	require.Equal(t, "test-element-3", element.Name)
	require.Equal(t, "This is a test element 3", element.Description)
	require.Equal(t, "This is the content 3", element.Contents)

	// Get the binary element
	element, err = client.GetDatasetElement(context.Background(), datasetID, "binary-element")
	require.NoError(t, err)
	require.Equal(t, "binary-element", element.Name)
	require.Equal(t, "this element has binary contents", element.Description)
	require.Equal(t, []byte("binary contents"), element.BinaryContents)

	// List elements in the dataset
	elements, err := client.ListDatasetElements(context.Background(), datasetID)
	require.NoError(t, err)
	require.Equal(t, 4, len(elements))

	// List datasets
	datasets, err := client.ListDatasets(context.Background())
	require.NoError(t, err)
	require.Equal(t, 1, len(datasets))
	require.Equal(t, datasetID, datasets[0].ID)
	require.Equal(t, "test-dataset", datasets[0].Name)
	require.Equal(t, "this is a test dataset", datasets[0].Description)
}
