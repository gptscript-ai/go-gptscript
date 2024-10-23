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
}

type DatasetMeta struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type Dataset struct {
	DatasetMeta `json:",inline"`
	BaseDir     string                        `json:"baseDir,omitempty"`
	Elements    map[string]DatasetElementMeta `json:"elements"`
}

type datasetRequest struct {
	Input           string   `json:"input"`
	WorkspaceID     string   `json:"workspaceID"`
	DatasetToolRepo string   `json:"datasetToolRepo"`
	Env             []string `json:"env"`
}

type createDatasetArgs struct {
	Name        string `json:"datasetName"`
	Description string `json:"datasetDescription"`
}

type addDatasetElementArgs struct {
	DatasetID          string `json:"datasetID"`
	ElementName        string `json:"elementName"`
	ElementDescription string `json:"elementDescription"`
	ElementContent     string `json:"elementContent"`
}

type addDatasetElementsArgs struct {
	DatasetID string           `json:"datasetID"`
	Elements  []DatasetElement `json:"elements"`
}

type listDatasetElementArgs struct {
	DatasetID string `json:"datasetID"`
}

type getDatasetElementArgs struct {
	DatasetID string `json:"datasetID"`
	Element   string `json:"element"`
}

func (g *GPTScript) ListDatasets(ctx context.Context, workspaceID string) ([]DatasetMeta, error) {
	if workspaceID == "" {
		workspaceID = os.Getenv("GPTSCRIPT_WORKSPACE_ID")
	}

	out, err := g.runBasicCommand(ctx, "datasets", datasetRequest{
		Input:           "{}",
		WorkspaceID:     workspaceID,
		DatasetToolRepo: g.globalOpts.DatasetToolRepo,
		Env:             g.globalOpts.Env,
	})
	if err != nil {
		return nil, err
	}

	var datasets []DatasetMeta
	if err = json.Unmarshal([]byte(out), &datasets); err != nil {
		return nil, err
	}
	return datasets, nil
}

func (g *GPTScript) CreateDataset(ctx context.Context, workspaceID, name, description string) (Dataset, error) {
	if workspaceID == "" {
		workspaceID = os.Getenv("GPTSCRIPT_WORKSPACE_ID")
	}

	args := createDatasetArgs{
		Name:        name,
		Description: description,
	}
	argsJSON, err := json.Marshal(args)
	if err != nil {
		return Dataset{}, fmt.Errorf("failed to marshal dataset args: %w", err)
	}

	out, err := g.runBasicCommand(ctx, "datasets/create", datasetRequest{
		Input:           string(argsJSON),
		WorkspaceID:     workspaceID,
		DatasetToolRepo: g.globalOpts.DatasetToolRepo,
		Env:             g.globalOpts.Env,
	})
	if err != nil {
		return Dataset{}, err
	}

	var dataset Dataset
	if err = json.Unmarshal([]byte(out), &dataset); err != nil {
		return Dataset{}, err
	}
	return dataset, nil
}

func (g *GPTScript) AddDatasetElement(ctx context.Context, workspaceID, datasetID, elementName, elementDescription, elementContent string) (DatasetElementMeta, error) {
	if workspaceID == "" {
		workspaceID = os.Getenv("GPTSCRIPT_WORKSPACE_ID")
	}

	args := addDatasetElementArgs{
		DatasetID:          datasetID,
		ElementName:        elementName,
		ElementDescription: elementDescription,
		ElementContent:     elementContent,
	}
	argsJSON, err := json.Marshal(args)
	if err != nil {
		return DatasetElementMeta{}, fmt.Errorf("failed to marshal element args: %w", err)
	}

	out, err := g.runBasicCommand(ctx, "datasets/add-element", datasetRequest{
		Input:           string(argsJSON),
		WorkspaceID:     workspaceID,
		DatasetToolRepo: g.globalOpts.DatasetToolRepo,
		Env:             g.globalOpts.Env,
	})
	if err != nil {
		return DatasetElementMeta{}, err
	}

	var element DatasetElementMeta
	if err = json.Unmarshal([]byte(out), &element); err != nil {
		return DatasetElementMeta{}, err
	}
	return element, nil
}

func (g *GPTScript) AddDatasetElements(ctx context.Context, workspaceID, datasetID string, elements []DatasetElement) error {
	if workspaceID == "" {
		workspaceID = os.Getenv("GPTSCRIPT_WORKSPACE_ID")
	}

	args := addDatasetElementsArgs{
		DatasetID: datasetID,
		Elements:  elements,
	}
	argsJSON, err := json.Marshal(args)
	if err != nil {
		return fmt.Errorf("failed to marshal element args: %w", err)
	}

	_, err = g.runBasicCommand(ctx, "datasets/add-elements", datasetRequest{
		Input:           string(argsJSON),
		WorkspaceID:     workspaceID,
		DatasetToolRepo: g.globalOpts.DatasetToolRepo,
		Env:             g.globalOpts.Env,
	})
	return err
}

func (g *GPTScript) ListDatasetElements(ctx context.Context, workspaceID, datasetID string) ([]DatasetElementMeta, error) {
	if workspaceID == "" {
		workspaceID = os.Getenv("GPTSCRIPT_WORKSPACE_ID")
	}

	args := listDatasetElementArgs{
		DatasetID: datasetID,
	}
	argsJSON, err := json.Marshal(args)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal element args: %w", err)
	}

	out, err := g.runBasicCommand(ctx, "datasets/list-elements", datasetRequest{
		Input:           string(argsJSON),
		WorkspaceID:     workspaceID,
		DatasetToolRepo: g.globalOpts.DatasetToolRepo,
		Env:             g.globalOpts.Env,
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

func (g *GPTScript) GetDatasetElement(ctx context.Context, workspaceID, datasetID, elementName string) (DatasetElement, error) {
	if workspaceID == "" {
		workspaceID = os.Getenv("GPTSCRIPT_WORKSPACE_ID")
	}

	args := getDatasetElementArgs{
		DatasetID: datasetID,
		Element:   elementName,
	}
	argsJSON, err := json.Marshal(args)
	if err != nil {
		return DatasetElement{}, fmt.Errorf("failed to marshal element args: %w", err)
	}

	out, err := g.runBasicCommand(ctx, "datasets/get-element", datasetRequest{
		Input:           string(argsJSON),
		WorkspaceID:     workspaceID,
		DatasetToolRepo: g.globalOpts.DatasetToolRepo,
		Env:             g.globalOpts.Env,
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
