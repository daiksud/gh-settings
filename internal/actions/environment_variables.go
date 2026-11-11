package actions

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/daiksud/gh-settings/internal/settings"
)

// EnvironmentVariablePlanChange retains the selected scope in plaintext presentation data.
type EnvironmentVariablePlanChange struct {
	Environment string  `json:"environment"`
	Name        string  `json:"name"`
	Before      *string `json:"before,omitempty"`
	After       *string `json:"after"`
}

func (c *EnvironmentVariablePlanChange) UnmarshalJSON(data []byte) error {
	type plain EnvironmentVariablePlanChange
	var value plain
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&value); err != nil {
		return err
	}
	*c = EnvironmentVariablePlanChange(value)
	return nil
}

func validateEnvironmentVariableReport(entries []EnvironmentVariablePlanChange) ([]settings.EnvironmentVariableChange, error) {
	var result []settings.EnvironmentVariableChange
	var selections []settings.EnvironmentVariableSelection
	for _, entry := range entries {
		if entry.After == nil {
			return nil, fmt.Errorf("missing Environment variable after value")
		}
		change, err := settings.NewVariableChange(entry.Name, entry.Before, *entry.After)
		if err != nil {
			return nil, err
		}
		scoped, err := settings.NewEnvironmentVariableChange(entry.Environment, change)
		if err != nil {
			return nil, err
		}
		result = append(result, scoped)
		selections = append(selections, settings.EnvironmentVariableSelection{Environment: entry.Environment, Name: entry.Name})
	}
	if _, err := settings.CanonicalEnvironmentVariableSelections(selections); err != nil {
		return nil, err
	}
	return result, nil
}
