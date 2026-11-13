package actions

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/daiksud/gh-settings/internal/settings"
)

type DeployKeyPlanEntry struct {
	Operation string `json:"operation"`
	Before    any    `json:"before,omitempty"`
	After     any    `json:"after,omitempty"`
}

func (entry *DeployKeyPlanEntry) UnmarshalJSON(data []byte) error {
	type plain DeployKeyPlanEntry
	var result plain
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	decoder.UseNumber()
	if err := decoder.Decode(&result); err != nil {
		return err
	}
	*entry = DeployKeyPlanEntry(result)
	return nil
}

func deployKeyReportValue(value any) (*settings.DeployKeyDisplay, error) {
	if value == nil {
		return nil, nil
	}
	data, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("invalid deploy key report")
	}
	if bytes.Equal(data, []byte("null")) {
		return nil, nil
	}
	var raw map[string]json.RawMessage
	if json.Unmarshal(data, &raw) != nil || raw == nil {
		return nil, fmt.Errorf("invalid deploy key report")
	}
	var result settings.DeployKeyDisplay
	for _, field := range []string{"title", "fingerprint", "allow_write_access"} {
		if len(raw[field]) == 0 || string(raw[field]) == "null" {
			return nil, fmt.Errorf("incomplete deploy key report")
		}
	}
	for field := range raw {
		if field != "id" && field != "title" && field != "fingerprint" && field != "allow_write_access" {
			return nil, fmt.Errorf("invalid deploy key report field")
		}
	}
	if id, ok := raw["id"]; ok && string(id) == "null" {
		return nil, fmt.Errorf("invalid deploy key report id")
	}
	if json.Unmarshal(data, &result) != nil || result.Validate() != nil {
		return nil, fmt.Errorf("invalid deploy key report")
	}
	return &result, nil
}

func validateDeployKeyReport(rows []DeployKeyPlanEntry) ([]settings.DeployKeyChange, error) {
	result := make([]settings.DeployKeyChange, 0, len(rows))
	ids, fingerprints := map[int64]bool{}, map[string]bool{}
	for _, row := range rows {
		before, err := deployKeyReportValue(row.Before)
		if err != nil {
			return nil, err
		}
		after, err := deployKeyReportValue(row.After)
		if err != nil {
			return nil, err
		}
		change, err := settings.NewDeployKeyChange(row.Operation, before, after)
		if err != nil {
			return nil, err
		}
		value := after
		if value == nil {
			value = before
		}
		if fingerprints[value.Fingerprint] || value.ID != nil && ids[*value.ID] {
			return nil, fmt.Errorf("duplicate deploy key report identity")
		}
		fingerprints[value.Fingerprint] = true
		if value.ID != nil {
			ids[*value.ID] = true
		}
		result = append(result, change)
	}
	return result, nil
}
