package actions

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/daiksud/gh-settings/internal/presentation"
	"github.com/daiksud/gh-settings/internal/settings"
)

// ActionsEventPolicyReport is a complete UI-shaped private report, never API input.
// Its canonical restrictions use enabled:false for absence and an allowed list
// for presence; enabled:true is intentionally not part of this report contract.
type ActionsEventPolicyReport struct {
	ID              *int64                          `json:"id,omitempty"`
	Name            string                          `json:"policy_name"`
	Enforcement     string                          `json:"enforcement_status"`
	RestrictEvents  ActionsEventPolicyEventsReport  `json:"restrict_events"`
	RestrictActors  ActionsEventPolicyActorsReport  `json:"restrict_actors"`
	TargetWorkflows *ActionsEventPolicyTargetReport `json:"target_workflows,omitempty"`
}
type ActionsEventPolicyTargetReport struct {
	Include []string `json:"include"`
	Exclude []string `json:"exclude"`
}
type ActionsEventPolicyEventsReport struct {
	Enabled       *bool    `json:"enabled,omitempty"`
	AllowedEvents []string `json:"allowed_events,omitempty"`
}
type ActionsEventPolicyActorsReport struct {
	Enabled       *bool                            `json:"enabled,omitempty"`
	AllowedActors *[]ActionsEventPolicyActorReport `json:"allowed_actors,omitempty"`
}
type ActionsEventPolicyActorReport struct {
	User *int64  `json:"user,omitempty"`
	Role *string `json:"role,omitempty"`
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
	disabled := false
	result := &ActionsEventPolicyReport{Name: value.Name, Enforcement: value.Enforcement}
	if value.Events == nil {
		result.RestrictEvents.Enabled = &disabled
	} else {
		result.RestrictEvents.AllowedEvents = append([]string{}, value.Events...)
	}
	if value.Actors == nil {
		result.RestrictActors.Enabled = &disabled
	} else {
		actors := make([]ActionsEventPolicyActorReport, 0, len(*value.Actors))
		for _, actor := range *value.Actors {
			entry := ActionsEventPolicyActorReport{}
			if actor.Type == "User" {
				id := actor.ID
				entry.User = &id
			} else {
				role := ""
				switch actor.ID {
				case 4:
					role = "write"
				case 2:
					role = "maintain"
				case 5:
					role = "repository_admin"
				}
				entry.Role = &role
			}
			actors = append(actors, entry)
		}
		result.RestrictActors.AllowedActors = &actors
	}
	if value.TargetWorkflows != nil {
		target := settings.CanonicalActionsWorkflowTarget(value.TargetWorkflows)
		result.TargetWorkflows = &ActionsEventPolicyTargetReport{Include: target.Include, Exclude: target.Exclude}
	}
	if value.ID > 0 {
		id := value.ID
		result.ID = &id
	}
	return result
}
func (value ActionsEventPolicyReport) policy() (settings.ActionsEventPolicy, error) {
	policy := settings.ActionsEventPolicy{Name: value.Name, Enforcement: value.Enforcement}
	if enabled := value.RestrictEvents.Enabled; enabled != nil {
		if *enabled || value.RestrictEvents.AllowedEvents != nil {
			return policy, fmt.Errorf("invalid disabled event restriction report")
		}
	} else {
		if value.RestrictEvents.AllowedEvents == nil {
			return policy, fmt.Errorf("missing event restriction report")
		}
		policy.Events = append([]string{}, value.RestrictEvents.AllowedEvents...)
	}
	if enabled := value.RestrictActors.Enabled; enabled != nil {
		if *enabled || value.RestrictActors.AllowedActors != nil {
			return policy, fmt.Errorf("invalid disabled actor restriction report")
		}
	} else {
		if value.RestrictActors.AllowedActors == nil || *value.RestrictActors.AllowedActors == nil {
			return policy, fmt.Errorf("missing actor restriction report")
		}
		actors := make([]settings.ActionsPolicyActor, 0, len(*value.RestrictActors.AllowedActors))
		for _, entry := range *value.RestrictActors.AllowedActors {
			if (entry.User == nil) == (entry.Role == nil) {
				return policy, fmt.Errorf("actor report requires exactly one identity")
			}
			actor := settings.ActionsPolicyActor{}
			if entry.User != nil {
				actor.Type = "User"
				actor.ID = *entry.User
			} else {
				actor.Type = "RepositoryRole"
				switch *entry.Role {
				case "write":
					actor.ID = 4
				case "maintain":
					actor.ID = 2
				case "repository_admin":
					actor.ID = 5
				default:
					return policy, fmt.Errorf("unsupported actor role report")
				}
			}
			actors = append(actors, actor)
		}
		policy.Actors = &actors
	}
	if value.TargetWorkflows != nil {
		if value.TargetWorkflows.Include == nil || value.TargetWorkflows.Exclude == nil {
			return policy, fmt.Errorf("target_workflows report requires include and exclude")
		}
		policy.TargetWorkflows = &settings.ActionsWorkflowTarget{Include: value.TargetWorkflows.Include, Exclude: value.TargetWorkflows.Exclude}
	}
	if value.ID != nil {
		if *value.ID <= 0 {
			return policy, fmt.Errorf("Actions policy report id must be positive")
		}
		policy.ID = *value.ID
	}
	return policy, settings.ValidateActionsEventPolicyState(policy)
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
		change, err := settings.NewActionsEventPolicyStateChange(before, after)
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
