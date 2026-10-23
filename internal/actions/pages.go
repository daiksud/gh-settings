package actions

import (
	"encoding/json"
	"fmt"
	"github.com/daiksud/gh-settings/internal/settings"
)

type PagesPlanChange struct {
	Field          string `json:"field"`
	Before         any    `json:"before"`
	After          any    `json:"after"`
	BeforeInactive bool   `json:"before_inactive,omitempty"`
}

func (r *PagesPlanChange) UnmarshalJSON(data []byte) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	for _, key := range []string{"field", "before", "after"} {
		if fields[key] == nil {
			return fmt.Errorf("missing Pages report %s", key)
		}
	}
	for key := range fields {
		if key != "field" && key != "before" && key != "after" && key != "before_inactive" {
			return fmt.Errorf("unknown Pages report key %q", key)
		}
	}
	if raw, ok := fields["before_inactive"]; ok {
		var value *bool
		if json.Unmarshal(raw, &value) != nil || value == nil {
			return fmt.Errorf("invalid Pages inactive marker")
		}
	}
	type plain PagesPlanChange
	var v plain
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	*r = PagesPlanChange(v)
	return nil
}

func validatePagesReport(raw []PagesPlanChange) ([]settings.PagesChange, error) {
	changes := []settings.PagesChange{}
	for _, r := range raw {
		c, err := settings.NewPagesChange(r.Field, r.Before, r.After, r.BeforeInactive)
		if err != nil {
			return nil, err
		}
		changes = append(changes, c)
	}
	if err := settings.ValidatePagesChanges(changes); err != nil {
		return nil, err
	}
	return changes, nil
}
