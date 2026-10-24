package actions

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/daiksud/gh-settings/internal/settings"
)

// RulesetPlanChange carries declaration-owned presentation values, never
// transport IDs, preserved unmanaged values, or apply input.
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
	last := ""
	deleting := false
	for _, e := range entries {
		change, err := settings.NewRulesetDisplayChange(e.Operation, e.Before, e.After)
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
