package actions

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// RulesetTeamApprovalsNormalization is private presentation metadata for one Team.
type RulesetTeamApprovalsNormalization struct {
	ReviewerID int64  `json:"reviewer_id"`
	Path       string `json:"path"`
	RawValue   int    `json:"raw_value"`
	Fallback   int    `json:"fallback"`
}

func (record *RulesetTeamApprovalsNormalization) UnmarshalJSON(data []byte) error {
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if invalidRulesetReportValue(raw) {
		return fmt.Errorf("null or noncanonical team approvals normalization field")
	}
	var fields struct {
		ReviewerID *int64  `json:"reviewer_id"`
		Path       *string `json:"path"`
		RawValue   *int    `json:"raw_value"`
		Fallback   *int    `json:"fallback"`
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&fields); err != nil {
		return err
	}
	if fields.ReviewerID == nil || fields.Path == nil || fields.RawValue == nil || fields.Fallback == nil {
		return fmt.Errorf("team approvals normalization requires integer reviewer_id, string path and integer raw_value/fallback")
	}
	*record = RulesetTeamApprovalsNormalization{ReviewerID: *fields.ReviewerID, Path: *fields.Path, RawValue: *fields.RawValue, Fallback: *fields.Fallback}
	return nil
}
