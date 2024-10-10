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
	BaseDir     string                    `json:"baseDir,omitempty"`
	Elements    map[string]DatasetElement `json:"elements"`
}

type datasetRequest struct {
	Input           string `json:"input"`
	Workspace       string `json:"workspace"`
	DatasetToolRepo string `json:"dataset_tool_repo"`
}

type createDatasetArgs struct {
	Name        string `json:"dataset_name"`
	Description string `json:"dataset_description"`
}

type addDatasetElementArgs struct {
	DatasetID          string `json:"dataset_id"`
	ElementName        string `json:"element_name"`
	ElementDescription string `json:"element_description"`
	ElementContent     string `json:"element_content"`
}

type listDatasetElementArgs struct {
	DatasetID string `json:"dataset_id"`
}

type getDatasetElementArgs struct {
	DatasetID string `json:"dataset_id"`
	Element   string `json:"element"`
}
