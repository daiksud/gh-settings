package actions

import (
	"encoding/json"
	"fmt"
	"github.com/daiksud/gh-settings/internal/settings"
)

type PagesPlanChange struct {
	Field          string                             `json:"field"`
	Before         any                                `json:"before"`
	After          any                                `json:"after"`
	BeforeInactive bool                               `json:"before_inactive,omitempty"`
	Normalization  *settings.ObservationNormalization `json:"normalization,omitempty"`
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
		if key != "field" && key != "before" && key != "after" && key != "before_inactive" && key != "normalization" {
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
		var c settings.PagesChange
		var err error
		if r.Normalization != nil {
			c, err = settings.NewPagesChangeWithNormalization(r.Field, r.Before, r.After, r.BeforeInactive, *r.Normalization)
		} else {
			c, err = settings.NewPagesChange(r.Field, r.Before, r.After, r.BeforeInactive)
		}
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

func minimumPagesReportFields(entries []PagesPlanChange) int {
	hasBranch, normalizedFolder := false, false
	for _, entry := range entries {
		if entry.Field == "pages.source.branch" {
			hasBranch = true
		}
		if entry.Field == "pages.source.path" && entry.Normalization != nil {
			normalizedFolder = true
		}
	}
	minimum := len(entries)
	if normalizedFolder && !hasBranch {
		minimum++
	}
	return minimum
}
