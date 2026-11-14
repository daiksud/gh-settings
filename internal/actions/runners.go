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

// RunnerDeletionEntry has an explicit null after value and no private evidence.
type RunnerDeletionEntry struct {
	Field  string                   `json:"field"`
	Before settings.RunnerDeletion  `json:"before"`
	After  *settings.RunnerDeletion `json:"after"`
}
type RunnerDeletionEntries []RunnerDeletionEntry

func (rows *RunnerDeletionEntries) UnmarshalJSON(data []byte) error {
	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		return fmt.Errorf("null runner deletion array")
	}
	var values []RunnerDeletionEntry
	if err := json.Unmarshal(data, &values); err != nil {
		return err
	}
	*rows = values
	return nil
}
func (entry *RunnerDeletionEntry) UnmarshalJSON(data []byte) error {
	var raw map[string]json.RawMessage
	if json.Unmarshal(data, &raw) != nil || len(raw) != 3 {
		return fmt.Errorf("invalid runner deletion report")
	}
	if _, ok := raw["field"]; !ok {
		return fmt.Errorf("missing runner deletion field")
	}
	beforeRaw, ok := raw["before"]
	if !ok {
		return fmt.Errorf("missing runner deletion before")
	}
	after, ok := raw["after"]
	if !ok || !bytes.Equal(bytes.TrimSpace(after), []byte("null")) {
		return fmt.Errorf("invalid runner deletion after")
	}
	var members map[string]json.RawMessage
	if json.Unmarshal(beforeRaw, &members) != nil || len(members) != 4 {
		return fmt.Errorf("invalid runner deletion before")
	}
	for _, name := range []string{"id", "name", "os", "custom_labels"} {
		value, ok := members[name]
		if !ok || bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return fmt.Errorf("invalid runner deletion before member")
		}
	}
	var field string
	var before settings.RunnerDeletion
	if json.Unmarshal(raw["field"], &field) != nil || json.Unmarshal(beforeRaw, &before) != nil || settings.ValidateRunnerDeletions([]settings.RunnerDeletion{before}) != nil {
		return fmt.Errorf("invalid runner deletion report")
	}
	*entry = RunnerDeletionEntry{Field: field, Before: before}
	return nil
}
func validateRunnerDeletionsReport(raw RunnerDeletionEntries, updates []settings.RunnerLabelsChange) ([]settings.RunnerDeletion, error) {
	result := make([]settings.RunnerDeletion, 0, len(raw))
	updated := map[int64]bool{}
	for _, change := range updates {
		before, _ := change.Values()
		updated[before.ID] = true
	}
	for _, entry := range raw {
		if entry.Field != settings.RunnersPath || entry.After != nil || updated[entry.Before.ID] {
			return nil, fmt.Errorf("invalid runner deletion report")
		}
		result = append(result, entry.Before)
	}
	if err := settings.ValidateRunnerDeletions(result); err != nil {
		return nil, err
	}
	return result, nil
}
