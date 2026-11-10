package actions

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/daiksud/gh-settings/internal/presentation"
	"github.com/daiksud/gh-settings/internal/settings"
)

// ActionsEventPolicyReport is a UI-shaped private report value, never API input.
type ActionsEventPolicyReport struct {
	ID             *int64                         `json:"id,omitempty"`
	Name           string                         `json:"policy_name"`
	Enforcement    string                         `json:"enforcement_status"`
	RestrictEvents ActionsEventPolicyEventsReport `json:"restrict_events"`
}
type ActionsEventPolicyEventsReport struct {
	AllowedEvents []string `json:"allowed_events"`
}
type ActionsEventPolicyPlanChanges []ActionsEventPolicyPlanChange

func (entries *ActionsEventPolicyPlanChanges) UnmarshalJSON(data []byte) error {
	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		return fmt.Errorf("null Actions policy report collection")
	}
	type plain ActionsEventPolicyPlanChanges
	var decoded plain
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	*entries = ActionsEventPolicyPlanChanges(decoded)
	return nil
}

type ActionsEventPolicyPlanChange struct {
	Operation string                    `json:"operation"`
	Before    *ActionsEventPolicyReport `json:"before,omitempty"`
	After     *ActionsEventPolicyReport `json:"after"`
}

func (entry *ActionsEventPolicyPlanChange) UnmarshalJSON(data []byte) error {
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	var invalid func(any) bool
	invalid = func(value any) bool {
		if value == nil {
			return true
		}
		switch value := value.(type) {
		case map[string]any:
			for key, v := range value {
				if key != strings.ToLower(key) || invalid(v) {
					return true
				}
			}
		case []any:
			for _, v := range value {
				if invalid(v) {
					return true
				}
			}
		}
		return false
	}
	if invalid(raw) {
		return fmt.Errorf("null or noncanonical Actions policy report value")
	}
	type plain ActionsEventPolicyPlanChange
	var decoded plain
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&decoded); err != nil {
		return err
	}
	*entry = ActionsEventPolicyPlanChange(decoded)
	return nil
}
func actionsEventPolicyReport(value settings.ActionsEventPolicy) *ActionsEventPolicyReport {
	result := &ActionsEventPolicyReport{Name: value.Name, Enforcement: value.Enforcement, RestrictEvents: ActionsEventPolicyEventsReport{AllowedEvents: append([]string(nil), value.Events...)}}
	if value.ID > 0 {
		id := value.ID
		result.ID = &id
	}
	return result
}
func (value ActionsEventPolicyReport) policy() (settings.ActionsEventPolicy, error) {
	policy := settings.ActionsEventPolicy{Name: value.Name, Enforcement: value.Enforcement, Events: append([]string(nil), value.RestrictEvents.AllowedEvents...)}
	if value.ID != nil {
		if *value.ID <= 0 {
			return policy, fmt.Errorf("Actions policy report id must be positive")
		}
		policy.ID = *value.ID
	}
	return policy, settings.ValidateActionsEventPolicy(policy)
}
func validateActionsEventPolicyReport(entries []ActionsEventPolicyPlanChange) ([]settings.ActionsEventPolicyChange, error) {
	var changes []settings.ActionsEventPolicyChange
	for _, entry := range entries {
		if entry.After == nil {
			return nil, fmt.Errorf("missing Actions policy after value")
		}
		after, err := entry.After.policy()
		if err != nil {
			return nil, err
		}
		var before *settings.ActionsEventPolicy
		if entry.Before != nil {
			value, err := entry.Before.policy()
			if err != nil {
				return nil, err
			}
			before = &value
		}
		change, err := settings.NewActionsEventPolicyChange(before, after)
		if err != nil {
			return nil, err
		}
		if entry.Operation != change.Operation() {
			return nil, fmt.Errorf("invalid Actions policy operation")
		}
		changes = append(changes, change)
	}
	if err := presentation.ValidateActionsEventPolicyChanges(changes); err != nil {
		return nil, err
	}
	return changes, nil
}
