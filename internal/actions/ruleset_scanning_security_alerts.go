package actions

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// RulesetScanningSecurityAlertsNormalization is private presentation evidence for one tool.
type RulesetScanningSecurityAlertsNormalization struct {
	Tool     string `json:"tool"`
	Path     string `json:"path"`
	RawValue string `json:"raw_value"`
	Fallback string `json:"fallback"`
}

func (record *RulesetScanningSecurityAlertsNormalization) UnmarshalJSON(data []byte) error {
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if invalidRulesetReportValue(raw) {
		return fmt.Errorf("null or noncanonical scanning security alerts normalization field")
	}
	var fields struct {
		Tool     *string `json:"tool"`
		Path     *string `json:"path"`
		RawValue *string `json:"raw_value"`
		Fallback *string `json:"fallback"`
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&fields); err != nil {
		return err
	}
	if fields.Tool == nil || fields.Path == nil || fields.RawValue == nil || fields.Fallback == nil {
		return fmt.Errorf("scanning security alerts normalization requires string tool, path, raw_value and fallback")
	}
	*record = RulesetScanningSecurityAlertsNormalization{Tool: *fields.Tool, Path: *fields.Path, RawValue: *fields.RawValue, Fallback: *fields.Fallback}
	return nil
}
