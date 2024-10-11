package gptscript

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
	Input           string `json:"input"`
	Workspace       string `json:"workspace"`
	DatasetToolRepo string `json:"datasetToolRepo"`
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

type listDatasetElementArgs struct {
	DatasetID string `json:"datasetID"`
}

type getDatasetElementArgs struct {
	DatasetID string `json:"datasetID"`
	Element   string `json:"element"`
}
