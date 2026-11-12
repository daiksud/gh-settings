package actions

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/daiksud/gh-settings/internal/settings"
)

// VariablePlanChange is same-version plaintext presentation data, never apply input.
type VariablePlanChange struct {
	Operation string  `json:"operation,omitempty"`
	Name      string  `json:"name"`
	Before    *string `json:"before,omitempty"`
	After     *string `json:"after"`
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
		change, err := variableReportChange(entry.Operation, entry.Name, entry.Before, entry.After)
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

func variableReportChange(operation, name string, before, after *string) (settings.VariableChange, error) {
	if operation == "delete" {
		if before == nil || after != nil {
			return settings.VariableChange{}, fmt.Errorf("invalid variable deletion report")
		}
		return settings.NewVariableDelete(settings.RepositoryVariable{Name: name, Value: *before})
	}
	if operation != "" || after == nil {
		return settings.VariableChange{}, fmt.Errorf("missing or invalid repository variable after value")
	}
	return settings.NewVariableChange(name, before, *after)
}
