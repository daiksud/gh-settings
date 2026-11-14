package uimodel

// RunnerLabels selects an existing repository runner and owns its custom labels.
type RunnerLabels struct {
	ID           int64     `json:"id"`
	CustomLabels *[]string `json:"custom_labels"`
}
