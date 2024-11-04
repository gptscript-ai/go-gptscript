package gptscript

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
)

type DatasetElementMeta struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type DatasetElement struct {
	DatasetElementMeta `json:",inline"`
	Contents           string `json:"contents"`
	BinaryContents     []byte `json:"binaryContents"`
}

type datasetRequest struct {
	Input       string   `json:"input"`
	DatasetTool string   `json:"datasetTool"`
	Env         []string `json:"env"`
}

type addDatasetElementsArgs struct {
	WorkspaceID string           `json:"workspaceID"`
	DatasetID   string           `json:"datasetID"`
	Elements    []DatasetElement `json:"elements"`
}

type listDatasetElementArgs struct {
	WorkspaceID string `json:"workspaceID"`
	DatasetID   string `json:"datasetID"`
}

type getDatasetElementArgs struct {
	WorkspaceID string `json:"workspaceID"`
	DatasetID   string `json:"datasetID"`
	Element     string `json:"name"`
}

func (g *GPTScript) ListDatasets(ctx context.Context) ([]string, error) {
	out, err := g.runBasicCommand(ctx, "datasets", datasetRequest{
		Input:       fmt.Sprintf(`{"workspaceID": %q}`, os.Getenv("GPTSCRIPT_WORKSPACE_ID")),
		DatasetTool: g.globalOpts.DatasetTool,
		Env:         g.globalOpts.Env,
	})
	if err != nil {
		return nil, err
	}

	var datasets []string
	if err = json.Unmarshal([]byte(out), &datasets); err != nil {
		return nil, err
	}
	return datasets, nil
}

func (g *GPTScript) CreateDatasetWithElements(ctx context.Context, elements []DatasetElement) (string, error) {
	return g.AddDatasetElements(ctx, "", elements)
}

func (g *GPTScript) AddDatasetElements(ctx context.Context, datasetID string, elements []DatasetElement) (string, error) {
	args := addDatasetElementsArgs{
		WorkspaceID: os.Getenv("GPTSCRIPT_WORKSPACE_ID"),
		DatasetID:   datasetID,
		Elements:    elements,
	}
	argsJSON, err := json.Marshal(args)
	if err != nil {
		return "", fmt.Errorf("failed to marshal element args: %w", err)
	}

	return g.runBasicCommand(ctx, "datasets/add-elements", datasetRequest{
		Input:       string(argsJSON),
		DatasetTool: g.globalOpts.DatasetTool,
		Env:         g.globalOpts.Env,
	})
}

func (g *GPTScript) ListDatasetElements(ctx context.Context, datasetID string) ([]DatasetElementMeta, error) {
	args := listDatasetElementArgs{
		WorkspaceID: os.Getenv("GPTSCRIPT_WORKSPACE_ID"),
		DatasetID:   datasetID,
	}
	argsJSON, err := json.Marshal(args)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal element args: %w", err)
	}

	out, err := g.runBasicCommand(ctx, "datasets/list-elements", datasetRequest{
		Input:       string(argsJSON),
		DatasetTool: g.globalOpts.DatasetTool,
		Env:         g.globalOpts.Env,
	})
	if err != nil {
		return nil, err
	}

	var elements []DatasetElementMeta
	if err = json.Unmarshal([]byte(out), &elements); err != nil {
		return nil, err
	}
	return elements, nil
}

func (g *GPTScript) GetDatasetElement(ctx context.Context, datasetID, elementName string) (DatasetElement, error) {
	args := getDatasetElementArgs{
		WorkspaceID: os.Getenv("GPTSCRIPT_WORKSPACE_ID"),
		DatasetID:   datasetID,
		Element:     elementName,
	}
	argsJSON, err := json.Marshal(args)
	if err != nil {
		return DatasetElement{}, fmt.Errorf("failed to marshal element args: %w", err)
	}

	out, err := g.runBasicCommand(ctx, "datasets/get-element", datasetRequest{
		Input:       string(argsJSON),
		DatasetTool: g.globalOpts.DatasetTool,
		Env:         g.globalOpts.Env,
	})
	if err != nil {
		return DatasetElement{}, err
	}

	var element DatasetElement
	if err = json.Unmarshal([]byte(out), &element); err != nil {
		return DatasetElement{}, err
	}

	return element, nil
}
