package presentation

import (
	"encoding/json"
	"fmt"

	"github.com/daiksud/gh-settings/internal/settings"
)

// ActionsEventPolicyLine renders complete event and actor policy ownership.
func ActionsEventPolicyLine(change settings.ActionsEventPolicyChange) string {
	value := func(policy settings.ActionsEventPolicy) string {
		events := "restrict_events.enabled=false"
		if policy.Events != nil {
			data, _ := json.Marshal(policy.Events)
			events = "restrict_events.allowed_events=" + string(data)
		}
		actors := "restrict_actors.enabled=false"
		if policy.Actors != nil {
			type actorValue struct {
				User *int64 `json:"user,omitempty"`
				Role string `json:"role,omitempty"`
			}
			values := make([]actorValue, 0, len(*policy.Actors))
			for _, actor := range *policy.Actors {
				value := actorValue{}
				if actor.Type == "User" {
					id := actor.ID
					value.User = &id
				} else {
					switch actor.ID {
					case 4:
						value.Role = "write"
					case 2:
						value.Role = "maintain"
					case 5:
						value.Role = "repository_admin"
					}
				}
				values = append(values, value)
			}
			data, _ := json.Marshal(values)
			actors = "restrict_actors.allowed_actors=" + string(data)
		}
		target := ""
		if policy.TargetWorkflows != nil {
			data, _ := json.Marshal(settings.CanonicalActionsWorkflowTarget(policy.TargetWorkflows))
			target = " target_workflows=" + string(data)
		}
		return fmt.Sprintf("policy_name=%q enforcement_status=%q %s %s", policy.Name, policy.Enforcement, events, actors) + target
	}
	after := change.After()
	identity := ""
	if after.ID > 0 {
		identity = fmt.Sprintf(" id=%d", after.ID)
	}
	prefix := fmt.Sprintf("repository.settings.actions.policies: %s %q%s: ", change.Operation(), change.Name(), identity)
	if before := change.Before(); before != nil {
		return prefix + value(*before) + " → " + value(after)
	}
	return prefix + value(after)
}

func ValidateActionsEventPolicyChanges(changes []settings.ActionsEventPolicyChange) error {
	names, ids := map[string]bool{}, map[int64]bool{}
	for _, change := range changes {
		if _, err := settings.NewActionsEventPolicyStateChange(change.Before(), change.After()); err != nil {
			return err
		}
		if names[change.Name()] || (change.ID() > 0 && ids[change.ID()]) {
			return fmt.Errorf("duplicate Actions policy operation")
		}
		names[change.Name()], ids[change.ID()] = true, true
	}
	return nil
}
