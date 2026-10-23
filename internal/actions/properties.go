package actions

import (
	"encoding/json"
	"fmt"

	"github.com/daiksud/gh-settings/internal/settings"
)

type PropertyPlanChange struct {
	Name   string `json:"name"`
	Type   string `json:"type"`
	Before any    `json:"before"`
	After  any    `json:"after"`
}

func (r *PropertyPlanChange) UnmarshalJSON(data []byte) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	if len(fields) != 4 {
		return fmt.Errorf("invalid custom property report fields")
	}
	for _, key := range []string{"name", "type", "before", "after"} {
		if fields[key] == nil {
			return fmt.Errorf("missing custom property report %s", key)
		}
	}
	type plain PropertyPlanChange
	var v plain
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	*r = PropertyPlanChange(v)
	return nil
}
func validatePropertyReport(raw []PropertyPlanChange) ([]settings.PropertyChange, error) {
	changes := []settings.PropertyChange{}
	seen := map[string]bool{}
	for _, r := range raw {
		if seen[r.Name] {
			return nil, fmt.Errorf("duplicate custom property comparison")
		}
		seen[r.Name] = true
		c, err := settings.NewPropertyChange(r.Name, r.Type, r.Before, r.After)
		if err != nil {
			return nil, err
		}
		changes = append(changes, c)
	}
	return changes, nil
}
