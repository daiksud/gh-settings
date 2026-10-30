package actions

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/daiksud/gh-settings/internal/settings"
)

// RulesetPlanChange carries selected whole supported objects and their declared
// identity. It is presentation data, never apply input.
type RulesetPlanChange struct {
	Operation                      string                                          `json:"operation"`
	Before                         *settings.RulesetDeclaration                    `json:"before,omitempty"`
	After                          *settings.RulesetDeclaration                    `json:"after,omitempty"`
	EnforcementNormalization       *settings.ObservationNormalization              `json:"enforcement_normalization,omitempty"`
	QualitySeverityNormalization   *settings.ObservationNormalization              `json:"quality_severity_normalization,omitempty"`
	RequiredApprovalsNormalization *settings.RulesetRequiredApprovalsNormalization `json:"required_approvals_normalization,omitempty"`
	BypassModeNormalizations       *[]RulesetBypassModeNormalization               `json:"bypass_mode_normalizations,omitempty"`
}

type RulesetBypassModeNormalization struct {
	ActorType     string                            `json:"actor_type"`
	ActorID       *int64                            `json:"actor_id,omitempty"`
	Normalization settings.ObservationNormalization `json:"normalization"`
}

func (c *RulesetPlanChange) UnmarshalJSON(data []byte) error {
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if invalidRulesetReportValue(raw) {
		return fmt.Errorf("null or noncanonical Ruleset report field")
	}
	type plain RulesetPlanChange
	var out plain
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&out); err != nil {
		return err
	}
	*c = RulesetPlanChange(out)
	return nil
}
func invalidRulesetReportValue(v any) bool {
	switch x := v.(type) {
	case nil:
		return true
	case map[string]any:
		for key, v := range x {
			// All protocol keys are lower-case; encoding/json otherwise accepts case aliases.
			if key != strings.ToLower(key) {
				return true
			}
			if invalidRulesetReportValue(v) {
				return true
			}
		}
	case []any:
		for _, v := range x {
			if invalidRulesetReportValue(v) {
				return true
			}
		}
	}
	return false
}
func validateRulesetReport(entries []RulesetPlanChange) ([]settings.RulesetDisplayChange, error) {
	out := make([]settings.RulesetDisplayChange, 0, len(entries))
	seen := map[string]bool{}
	seenIDs := map[int64]bool{}
	last := ""
	for _, e := range entries {
		for _, policy := range []*settings.RulesetDeclaration{e.Before, e.After} {
			if policy != nil && policy.HasWriteOnlyDismissal() {
				return nil, fmt.Errorf("Ruleset plan report does not support write-only dismissal_restriction")
			}
		}
		var change settings.RulesetDisplayChange
		var err error
		for _, record := range []*settings.ObservationNormalization{e.EnforcementNormalization, e.QualitySeverityNormalization} {
			if record != nil && *record == (settings.ObservationNormalization{}) {
				return nil, fmt.Errorf("present Ruleset normalization metadata cannot be empty")
			}
		}
		records := settings.RulesetNormalizations{}
		if e.EnforcementNormalization != nil {
			records.Enforcement = *e.EnforcementNormalization
		}
		if e.QualitySeverityNormalization != nil {
			records.QualitySeverity = *e.QualitySeverityNormalization
		}
		if e.RequiredApprovalsNormalization != nil {
			if *e.RequiredApprovalsNormalization == (settings.RulesetRequiredApprovalsNormalization{}) {
				return nil, fmt.Errorf("present Ruleset required approvals normalization metadata cannot be empty")
			}
			records.RequiredApprovals = *e.RequiredApprovalsNormalization
		}
		if e.BypassModeNormalizations != nil {
			if len(*e.BypassModeNormalizations) == 0 {
				return nil, fmt.Errorf("present Ruleset bypass mode normalization metadata cannot be empty")
			}
			for _, record := range *e.BypassModeNormalizations {
				records.BypassModes = append(records.BypassModes, settings.RulesetBypassModeNormalization{
					ActorType: record.ActorType, ActorID: record.ActorID, Normalization: record.Normalization,
				})
			}
		}
		if !records.IsZero() {
			change, err = settings.NewRulesetDisplayChangeWithNormalizations(e.Operation, e.Before, e.After, records)
		} else {
			change, err = settings.NewRulesetDisplayChange(e.Operation, e.Before, e.After)
		}
		if err != nil {
			return nil, err
		}
		if seen[change.Name()] {
			return nil, fmt.Errorf("duplicate Ruleset operation")
		}
		seen[change.Name()] = true
		if e.Before != nil && e.Before.ID != nil {
			if seenIDs[*e.Before.ID] {
				return nil, fmt.Errorf("duplicate Ruleset operation ID")
			}
			seenIDs[*e.Before.ID] = true
		}
		if last != "" && last >= change.Name() {
			return nil, fmt.Errorf("unordered Ruleset operations")
		}
		last = change.Name()
		out = append(out, change)
	}
	return out, nil
}
