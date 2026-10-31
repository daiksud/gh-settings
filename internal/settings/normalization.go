package settings

import (
	"bytes"
	"encoding/json"
	"fmt"
)

const WorkflowPermissionsPath = "repository.settings.actions.general.workflow_permissions.default_permissions"

// ObservationNormalization distinguishes an effective UI choice from the exact
// acquired API token. An empty Path means no fallback was used; an empty raw
// token is still an acquired string when Path is present.
type ObservationNormalization struct {
	Path     string `json:"path"`
	RawValue string `json:"raw_value"`
	Fallback string `json:"fallback"`
}

// UnmarshalJSON preserves the distinction between an acquired empty token and
// absent/null provenance in the same-version plan report.
func (normalization *ObservationNormalization) UnmarshalJSON(data []byte) error {
	var fields struct {
		Path     *string `json:"path"`
		RawValue *string `json:"raw_value"`
		Fallback *string `json:"fallback"`
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&fields); err != nil {
		return fmt.Errorf("decode normalization: %w", err)
	}
	if fields.Path == nil || fields.RawValue == nil || fields.Fallback == nil {
		return fmt.Errorf("normalization requires string path, raw_value and fallback")
	}
	*normalization = ObservationNormalization{Path: *fields.Path, RawValue: *fields.RawValue, Fallback: *fields.Fallback}
	return nil
}
