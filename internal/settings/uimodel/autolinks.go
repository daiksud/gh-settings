package uimodel

type Autolink struct {
	ID              *int64 `json:"id,omitempty"`
	ReferencePrefix string `json:"reference_prefix"`
	TargetURL       string `json:"target_url"`
	ReferenceFormat string `json:"reference_format"`
}
