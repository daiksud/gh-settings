package presentation

import (
	"encoding/json"
	"fmt"

	"github.com/daiksud/gh-settings/internal/settings"
)

// ActionsEventPolicyLine renders the complete supported event policy ownership.
func ActionsEventPolicyLine(change settings.ActionsEventPolicyChange) string {
	value := func(policy settings.ActionsEventPolicy) string {
		events, _ := json.Marshal(policy.Events)
		target := ""
		if policy.TargetWorkflows != nil {
			data, _ := json.Marshal(settings.CanonicalActionsWorkflowTarget(policy.TargetWorkflows))
			target = " target_workflows=" + string(data)
		}
		return fmt.Sprintf("policy_name=%q enforcement_status=%q restrict_events.allowed_events=%s", policy.Name, policy.Enforcement, events) + target
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
		if _, err := settings.NewActionsEventPolicyChange(change.Before(), change.After()); err != nil {
			return err
		}
		if names[change.Name()] || (change.ID() > 0 && ids[change.ID()]) {
			return fmt.Errorf("duplicate Actions policy operation")
		}
		names[change.Name()], ids[change.ID()] = true, true
	}
	return nil
}
