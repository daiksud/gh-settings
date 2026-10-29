package github

import (
	"fmt"
	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

func declareUIActionsWorkflow(choice *uimodel.WorkflowPermissionChoice) (*string, error) {
	if choice == nil {
		return nil, nil
	}
	var value string
	switch *choice {
	case uimodel.ReadAndWritePermissions:
		value = "write"
	case uimodel.ReadRepositoryContentsAndPackagesPermissions:
		value = "read"
	default:
		return nil, fmt.Errorf("unsupported UI workflow_permissions choice")
	}
	return &value, nil
}

func WithActionsWorkflowPolicyUI(ui uimodel.ActionsGeneral, policy settings.ActionsPolicyDeclaration) (uimodel.ActionsGeneral, error) {
	if policy.WorkflowPermissionsEmpty() {
		return ui, nil
	}
	if policy.DefaultWorkflowPermissions == nil || policy.CanApprovePullRequestReviews == nil {
		return uimodel.ActionsGeneral{}, fmt.Errorf("workflow policy requires complete permission and approval values")
	}
	return WithActionsWorkflowObservation(ui, settings.WorkflowPermissionsState{DefaultWorkflowPermissions: *policy.DefaultWorkflowPermissions, CanApprovePullRequestReviews: *policy.CanApprovePullRequestReviews})
}

func WithActionsWorkflowObservation(ui uimodel.ActionsGeneral, state settings.WorkflowPermissionsState) (uimodel.ActionsGeneral, error) {
	var choice uimodel.WorkflowPermissionChoice
	switch state.DefaultWorkflowPermissions {
	case "write":
		choice = uimodel.ReadAndWritePermissions
	case "read":
		choice = uimodel.ReadRepositoryContentsAndPackagesPermissions
	default:
		return uimodel.ActionsGeneral{}, fmt.Errorf("workflow_permissions is outside UI choices")
	}
	ui.WorkflowPermissions = &choice
	ui.AllowGitHubActionsToCreateAndApprovePullRequests = copyUIActionsValue(&state.CanApprovePullRequestReviews)
	return ui, nil
}
