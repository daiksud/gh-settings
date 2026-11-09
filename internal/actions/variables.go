package actions

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/daiksud/gh-settings/internal/settings"
)

// VariablePlanChange is same-version plaintext presentation data, never apply input.
type VariablePlanChange struct {
	Name   string  `json:"name"`
	Before *string `json:"before,omitempty"`
	After  *string `json:"after"`
}

func (c *VariablePlanChange) UnmarshalJSON(data []byte) error {
	type plain VariablePlanChange
	var value plain
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&value); err != nil {
		return err
	}
	*c = VariablePlanChange(value)
	return nil
}
func validateVariableReport(entries []VariablePlanChange) ([]settings.VariableChange, error) {
	var result []settings.VariableChange
	seen := map[string]bool{}
	for _, entry := range entries {
		if entry.After == nil {
			return nil, fmt.Errorf("missing repository variable after value")
		}
		change, err := settings.NewVariableChange(entry.Name, entry.Before, *entry.After)
		if err != nil {
			return nil, err
		}
		if seen[change.Name()] {
			return nil, fmt.Errorf("duplicate repository variable report identity")
		}
		seen[change.Name()] = true
		result = append(result, change)
	}
	return result, nil
}
