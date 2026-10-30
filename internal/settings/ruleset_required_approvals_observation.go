package settings

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// RulesetRequiredApprovalsNormalization retains the acquired integer separately
// from the UI fallback. It is evidence, never desired policy or mutation input.
type RulesetRequiredApprovalsNormalization struct {
	Path     string `json:"path"`
	RawValue int    `json:"raw_value"`
	Fallback int    `json:"fallback"`
}

// RawRequiredApprovals returns an owned acquired count, or nil for an absent PR
// rule. Missing or incomplete acquisition is an error rather than absence.
func (state RulesetState) RawRequiredApprovals() (*int, error) {
	if err := state.validateNormalizationEvidence(); err != nil {
		return nil, err
	}
	count := rulesetRequiredApprovals(state.Policy)
	if count == nil {
		return nil, nil
	}
	value := *count
	if record := state.RequiredApprovalsNormalization; record != (RulesetRequiredApprovalsNormalization{}) {
		value = record.RawValue
	}
	return &value, nil
}

func rulesetRequiredApprovals(policy RulesetDeclaration) *int {
	if policy.Rules != nil {
		for _, rule := range *policy.Rules {
			if rule.Type == "pull_request" && rule.Parameters != nil {
				return rule.Parameters.RequiredApprovingReviewCount
			}
		}
	}
	return nil
}

// UnmarshalJSON distinguishes an explicitly acquired integer and zero fallback
// from omitted or null private report metadata. Semantic checks remain separate.
func (record *RulesetRequiredApprovalsNormalization) UnmarshalJSON(data []byte) error {
	var fields struct {
		Path     *string `json:"path"`
		RawValue *int    `json:"raw_value"`
		Fallback *int    `json:"fallback"`
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&fields); err != nil {
		return fmt.Errorf("decode Ruleset required approvals normalization: %w", err)
	}
	if fields.Path == nil || fields.RawValue == nil || fields.Fallback == nil {
		return fmt.Errorf("Ruleset required approvals normalization requires string path and integer raw_value/fallback")
	}
	*record = RulesetRequiredApprovalsNormalization{Path: *fields.Path, RawValue: *fields.RawValue, Fallback: *fields.Fallback}
	return nil
}
