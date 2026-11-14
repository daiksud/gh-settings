package actions

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/daiksud/gh-settings/internal/settings"
)

// RunnerPlanChange keeps numeric selectors exact through private JSON decoding.
type RunnerPlanChange struct {
	Field  string                `json:"field"`
	Before settings.RunnerLabels `json:"before"`
	After  settings.RunnerLabels `json:"after"`
}
type RunnerPlanChanges []RunnerPlanChange

func (rows *RunnerPlanChanges) UnmarshalJSON(data []byte) error {
	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		return fmt.Errorf("null runner report array")
	}
	var values []RunnerPlanChange
	if err := json.Unmarshal(data, &values); err != nil {
		return err
	}
	*rows = values
	return nil
}
func (entry *RunnerPlanChange) UnmarshalJSON(data []byte) error {
	var raw map[string]json.RawMessage
	if json.Unmarshal(data, &raw) != nil || len(raw) != 3 {
		return fmt.Errorf("incomplete runner report change")
	}
	for k := range raw {
		if k != "field" && k != "before" && k != "after" {
			return fmt.Errorf("unknown runner report change member")
		}
	}
	var field *string
	if json.Unmarshal(raw["field"], &field) != nil || field == nil {
		return fmt.Errorf("invalid runner report field")
	}
	before, err := runnerReportValue(raw["before"])
	if err != nil {
		return err
	}
	after, err := runnerReportValue(raw["after"])
	if err != nil {
		return err
	}
	*entry = RunnerPlanChange{*field, before, after}
	return nil
}

func runnerReportValue(v any) (settings.RunnerLabels, error) {
	b, e := json.Marshal(v)
	if e != nil {
		return settings.RunnerLabels{}, e
	}
	var raw map[string]json.RawMessage
	if json.Unmarshal(b, &raw) != nil || len(raw) != 2 {
		return settings.RunnerLabels{}, fmt.Errorf("invalid runner labels report")
	}
	for k, v := range raw {
		if (k != "id" && k != "custom_labels") || bytes.Equal(bytes.TrimSpace(v), []byte("null")) {
			return settings.RunnerLabels{}, fmt.Errorf("invalid runner labels report member")
		}
	}
	var d settings.RunnerLabels
	if json.Unmarshal(b, &d) != nil {
		return d, fmt.Errorf("invalid runner labels report value")
	}
	return d, settings.ValidateRunnerLabels([]settings.RunnerLabels{d})
}
func validateRunnersReport(raw []RunnerPlanChange) ([]settings.RunnerLabelsChange, error) {
	result := []settings.RunnerLabelsChange{}
	seen := map[int64]bool{}
	for _, entry := range raw {
		if entry.Field != settings.RunnersPath {
			return nil, fmt.Errorf("invalid runner labels report")
		}
		before, e := runnerReportValue(entry.Before)
		if e != nil {
			return nil, e
		}
		after, e := runnerReportValue(entry.After)
		if e != nil {
			return nil, e
		}
		if seen[after.ID] {
			return nil, fmt.Errorf("duplicate runner labels change")
		}
		seen[after.ID] = true
		c, e := settings.NewRunnerLabelsChange(before, after)
		if e != nil {
			return nil, e
		}
		result = append(result, c)
	}
	return result, nil
}
