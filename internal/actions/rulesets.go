package actions

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/daiksud/gh-settings/internal/settings"
)

// RulesetPlanChange carries full writable policies, never transport IDs or apply input.
type RulesetPlanChange struct {
	Operation string                       `json:"operation"`
	Before    *settings.RulesetDeclaration `json:"before,omitempty"`
	After     *settings.RulesetDeclaration `json:"after,omitempty"`
}

func (c *RulesetPlanChange) UnmarshalJSON(data []byte) error {
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if rulesetReportHasNull(raw) {
		return fmt.Errorf("null Ruleset report value")
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
func rulesetReportHasNull(v any) bool {
	switch x := v.(type) {
	case nil:
		return true
	case map[string]any:
		for _, v := range x {
			if rulesetReportHasNull(v) {
				return true
			}
		}
	case []any:
		for _, v := range x {
			if rulesetReportHasNull(v) {
				return true
			}
		}
	}
	return false
}
func validateRulesetReport(entries []RulesetPlanChange) ([]settings.RulesetChange, error) {
	out := make([]settings.RulesetChange, 0, len(entries))
	seen := map[string]bool{}
	last := ""
	deleting := false
	for _, e := range entries {
		var id int64 = 1
		if e.Operation == "create" {
			id = 0
		}
		change, err := settings.NewRulesetChange(e.Operation, id, e.Before, e.After)
		if err != nil {
			return nil, err
		}
		if seen[change.Name()] {
			return nil, fmt.Errorf("duplicate Ruleset operation")
		}
		seen[change.Name()] = true
		if e.Operation == "delete" && !deleting {
			deleting = true
			last = ""
		} else if deleting && e.Operation != "delete" {
			return nil, fmt.Errorf("Ruleset operations must delete last")
		}
		if last != "" && last >= change.Name() {
			return nil, fmt.Errorf("unordered Ruleset operations")
		}
		last = change.Name()
		out = append(out, change)
	}
	return out, nil
}
