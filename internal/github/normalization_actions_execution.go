package github

import (
	"fmt"

	"github.com/daiksud/gh-settings/internal/settings"
)

func prepareActionsExecutionObservation(enabled *bool, allowed *string, sha *bool) (settings.ActionsPolicyState, error) {
	if enabled == nil || allowed == nil || sha == nil {
		return settings.ActionsPolicyState{}, fmt.Errorf("Actions execution policy requires complete enabled, allowed actions and SHA values")
	}
	state := settings.ActionsPolicyState{Enabled: *enabled, AllowedActions: *allowed, SHAPinningRequired: *sha}
	if !settings.ValidAllowedActions(*allowed) {
		_, fallback, _ := settings.ActionsPermissionsValueForChoice(settings.DefaultActionsPermissionsChoice)
		state.AllowedActions = *fallback
		state.AllowedActionsNormalization = settings.ObservationNormalization{Path: settings.ActionsPermissionsPath, RawValue: *allowed, Fallback: string(settings.DefaultActionsPermissionsChoice)}
	}
	if err := state.Validate(); err != nil {
		return settings.ActionsPolicyState{}, err
	}
	return state, nil
}
