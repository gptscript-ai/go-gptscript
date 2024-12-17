package gptscript

type Permission struct {
	CreatedAt          int64       `json:"created"`
	ID                 string      `json:"id"`
	Object             string      `json:"object"`
	AllowCreateEngine  bool        `json:"allow_create_engine"`
	AllowSampling      bool        `json:"allow_sampling"`
	AllowLogprobs      bool        `json:"allow_logprobs"`
	AllowSearchIndices bool        `json:"allow_search_indices"`
	AllowView          bool        `json:"allow_view"`
	AllowFineTuning    bool        `json:"allow_fine_tuning"`
	Organization       string      `json:"organization"`
	Group              interface{} `json:"group"`
	IsBlocking         bool        `json:"is_blocking"`
}

type Model struct {
	CreatedAt  int64             `json:"created"`
	ID         string            `json:"id"`
	Object     string            `json:"object"`
	OwnedBy    string            `json:"owned_by"`
	Permission []Permission      `json:"permission"`
	Root       string            `json:"root"`
	Parent     string            `json:"parent"`
	Metadata   map[string]string `json:"metadata"`
}
