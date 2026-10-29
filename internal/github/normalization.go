package github

import (
	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

// NormalizeWorkflowPermissions projects a complete, correctly typed observation.
// Acquisition must establish presence and type before calling it. The raw token
// remains evidence for comparisons and write verification, never desired intent.
func NormalizeWorkflowPermissions(state settings.WorkflowPermissionsState) settings.WorkflowPermissionsState {
	if settings.ValidDefaultWorkflowPermissions(state.DefaultWorkflowPermissions) {
		return state
	}
	state.Normalization = settings.ObservationNormalization{
		Path:     settings.WorkflowPermissionsPath,
		RawValue: state.DefaultWorkflowPermissions,
		Fallback: string(uimodel.DefaultActionsWorkflowPermissions),
	}
	state.DefaultWorkflowPermissions = "read"
	return state
}
