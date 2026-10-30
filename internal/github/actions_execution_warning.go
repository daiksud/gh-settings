package github

import "github.com/daiksud/gh-settings/internal/settings"

func (client *ActionsPolicyClient) warnExecutionNormalization(state settings.ActionsPolicyState, desired settings.ActionsPolicyDeclaration) {
	if client.NormalizationWarning == nil || desired.Enabled == nil || *desired.Enabled && desired.AllowedActions == nil || !state.Enabled {
		return
	}
	if err := state.Validate(); err != nil {
		return
	}
	if state.AllowedActionsNormalization != (settings.ObservationNormalization{}) {
		client.NormalizationWarning(state.AllowedActionsNormalization)
	}
}
