package actions

import (
	"encoding/json"
	"fmt"

	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

type ActionsExecutionPlanChange struct {
	Before        uimodel.ActionsPermissionChoice   `json:"before"`
	After         uimodel.ActionsPermissionChoice   `json:"after"`
	Normalization settings.ObservationNormalization `json:"normalization"`
}

func (entry *ActionsExecutionPlanChange) UnmarshalJSON(data []byte) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	for key := range fields {
		if key != "before" && key != "after" && key != "normalization" {
			return fmt.Errorf("unknown Actions radio report member")
		}
	}
	var raw struct {
		Before        *uimodel.ActionsPermissionChoice   `json:"before"`
		After         *uimodel.ActionsPermissionChoice   `json:"after"`
		Normalization *settings.ObservationNormalization `json:"normalization"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if raw.Before == nil || raw.After == nil || raw.Normalization == nil {
		return fmt.Errorf("Actions radio report requires complete before, after and normalization")
	}
	*entry = ActionsExecutionPlanChange{Before: *raw.Before, After: *raw.After, Normalization: *raw.Normalization}
	return nil
}

func validateActionsExecutionReport(entries []ActionsExecutionPlanChange) ([]settings.ActionsExecutionChange, int, error) {
	if len(entries) > 1 {
		return nil, 0, fmt.Errorf("duplicate Actions radio report")
	}
	var changes []settings.ActionsExecutionChange
	managed := 0
	for _, entry := range entries {
		if entry.Before != uimodel.DisableActions && entry.Before != settings.DefaultActionsPermissionsChoice {
			return nil, 0, fmt.Errorf("invalid observed Actions radio choice")
		}
		change, err := settings.NewActionsExecutionChange(settings.ActionsPolicyState{Enabled: entry.Before != uimodel.DisableActions, AllowedActions: "all", AllowedActionsNormalization: entry.Normalization}, entry.After)
		if err != nil {
			return nil, 0, err
		}
		changes = append(changes, change)
		managed += len(change.NativeFields())
	}
	return changes, managed, nil
}

func validateActionsExecutionContext(context *settings.ActionsExecutionContext, changes []PlanChange, normalized []ActionsExecutionPlanChange) error {
	var enabled, allowed *PlanChange
	for i := range changes {
		switch changes[i].Field {
		case "actions.enabled":
			if enabled != nil {
				return fmt.Errorf("duplicate Actions radio component")
			}
			enabled = &changes[i]
		case "actions.allowed_actions":
			if allowed != nil {
				return fmt.Errorf("duplicate Actions radio component")
			}
			allowed = &changes[i]
		}
	}
	if context == nil {
		if enabled != nil || allowed != nil {
			return fmt.Errorf("missing Actions radio context")
		}
		return nil
	}
	if len(normalized) != 0 || context.Validate() != nil {
		return fmt.Errorf("invalid Actions radio context")
	}
	beforeEnabled, beforeAllowed, _ := settings.ActionsPermissionsValueForChoice(context.Before)
	afterEnabled, afterAllowed, _ := settings.ActionsPermissionsValueForChoice(context.After)
	if (enabled != nil) != (beforeEnabled != afterEnabled) {
		return fmt.Errorf("inconsistent Actions enabled component")
	}
	if enabled != nil && (enabled.Before != beforeEnabled || enabled.After != afterEnabled) {
		return fmt.Errorf("Actions enabled component differs from context")
	}
	if allowed != nil {
		if afterAllowed == nil || allowed.After != *afterAllowed || allowed.Before != context.ObservedAllowed {
			return fmt.Errorf("Actions allowed component differs from context")
		}
		if beforeEnabled && (beforeAllowed == nil || allowed.Before != *beforeAllowed) {
			return fmt.Errorf("Actions observed component differs from context")
		}
	} else if afterEnabled && context.ObservedAllowed != *afterAllowed {
		return fmt.Errorf("missing Actions allowed component")
	}
	if enabled == nil && allowed == nil {
		return fmt.Errorf("unreferenced Actions radio context")
	}
	return nil
}
