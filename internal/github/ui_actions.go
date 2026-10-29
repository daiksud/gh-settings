package github

import (
	"fmt"
	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

// PreparedActionsExecution retains raw latent policy outside the UI model.
type PreparedActionsExecution struct {
	General uimodel.ActionsGeneral
	Raw     settings.ActionsPolicyState
}

// ActionsExecutionPolicyUI projects exported execution values without fabricating acquisition.
func ActionsExecutionPolicyUI(policy settings.ActionsPolicyDeclaration) (uimodel.ActionsGeneral, error) {
	if policy.ExecutionEmpty() {
		return uimodel.ActionsGeneral{}, nil
	}
	if policy.Enabled == nil || policy.AllowedActions == nil || policy.SHAPinningRequired == nil {
		return uimodel.ActionsGeneral{}, fmt.Errorf("Actions execution policy requires complete enabled, allowed actions and SHA pinning values")
	}
	state := settings.ActionsPolicyState{Enabled: *policy.Enabled, AllowedActions: *policy.AllowedActions, SHAPinningRequired: *policy.SHAPinningRequired}
	if *policy.AllowedActions == "selected" {
		if !policy.SelectedActions.Complete() {
			return uimodel.ActionsGeneral{}, fmt.Errorf("selected Actions policy requires complete companion values")
		}
		state.Selected = &settings.SelectedActionsState{GitHubOwnedAllowed: *policy.SelectedActions.GitHubOwnedAllowed, VerifiedAllowed: *policy.SelectedActions.VerifiedAllowed, PatternsAllowed: *policy.SelectedActions.PatternsAllowed}
	} else if !policy.SelectedActions.Empty() {
		return uimodel.ActionsGeneral{}, fmt.Errorf("selected Actions companions require selected policy")
	}
	prepared, err := ActionsExecutionObservation(state)
	if err != nil {
		return uimodel.ActionsGeneral{}, err
	}
	return prepared.General, nil
}

func ActionsDeclaration(ui uimodel.ActionsGeneral) (settings.ActionsPolicyDeclaration, error) {
	result := settings.ActionsPolicyDeclaration{SHAPinningRequired: copyUIActionsValue(ui.RequireActionsToBePinnedToAFullLengthCommitSHA)}
	if ui.ActionsPermissions != nil {
		enabled := true
		var allowed string
		switch *ui.ActionsPermissions {
		case uimodel.DisableActions:
			enabled = false
		case uimodel.AllowAllActionsAndReusableWorkflows:
			allowed = "all"
		case uimodel.AllowOwnerActionsAndReusableWorkflows:
			allowed = "local_only"
		case uimodel.AllowOwnerAndSelectNonOwnerActionsAndReusableWorkflows:
			allowed = "selected"
		default:
			return settings.ActionsPolicyDeclaration{}, fmt.Errorf("unsupported actions_permissions UI choice")
		}
		result.Enabled = &enabled
		if enabled {
			result.AllowedActions = &allowed
		}
	}
	result.SelectedActions.GitHubOwnedAllowed = copyUIActionsValue(ui.AllowActionsCreatedByGitHub)
	result.SelectedActions.VerifiedAllowed = copyUIActionsValue(ui.AllowActionsByMarketplaceVerifiedCreators)
	if ui.AllowOrBlockSpecifiedActionsAndReusableWorkflows != nil {
		patterns := append([]string{}, (*ui.AllowOrBlockSpecifiedActionsAndReusableWorkflows)...)
		result.SelectedActions.PatternsAllowed = &patterns
	}
	if !result.SelectedActions.Empty() && ui.ActionsPermissions != nil && *ui.ActionsPermissions != uimodel.AllowOwnerAndSelectNonOwnerActionsAndReusableWorkflows {
		return settings.ActionsPolicyDeclaration{}, fmt.Errorf("selected action controls require the selected non-owner Actions permission or an omitted radio")
	}
	var err error
	result.DefaultWorkflowPermissions, err = declareUIActionsWorkflow(ui.WorkflowPermissions)
	if err != nil {
		return settings.ActionsPolicyDeclaration{}, err
	}
	result.CanApprovePullRequestReviews = copyUIActionsValue(ui.AllowGitHubActionsToCreateAndApprovePullRequests)
	result.RetentionDays = copyUIActionsValue(ui.CheckWorkflowRunStatusArtifactAndLogRetention)
	result.ForkApprovalPolicy, err = declareUIActionsForkApproval(ui.ApprovalForRunningForkPullRequestWorkflowsFromContributors)
	if err != nil {
		return settings.ActionsPolicyDeclaration{}, err
	}
	result.PrivateFork, err = declareUIActionsPrivateFork(ui.ForkPullRequestWorkflows)
	if err != nil {
		return settings.ActionsPolicyDeclaration{}, err
	}
	if err := result.Validate(); err != nil {
		return settings.ActionsPolicyDeclaration{}, err
	}
	return result, nil
}

// ActionsExecutionObservation accepts the caller's requested acquisition scope;
// a nil selected companion remains unacquired rather than an observed empty list.
func ActionsExecutionObservation(state settings.ActionsPolicyState) (PreparedActionsExecution, error) {
	if err := state.Validate(); err != nil {
		return PreparedActionsExecution{}, err
	}
	raw := state
	if state.Selected != nil {
		selected := *state.Selected
		selected.PatternsAllowed = append([]string{}, state.Selected.PatternsAllowed...)
		raw.Selected = &selected
	}
	choice := uimodel.DisableActions
	if state.Enabled {
		switch state.AllowedActions {
		case "all":
			choice = uimodel.AllowAllActionsAndReusableWorkflows
		case "local_only":
			choice = uimodel.AllowOwnerActionsAndReusableWorkflows
		case "selected":
			choice = uimodel.AllowOwnerAndSelectNonOwnerActionsAndReusableWorkflows
		}
	}
	ui := uimodel.ActionsGeneral{ActionsPermissions: &choice, RequireActionsToBePinnedToAFullLengthCommitSHA: copyUIActionsValue(&state.SHAPinningRequired)}
	if state.Enabled && state.AllowedActions == "selected" && state.Selected != nil {
		ui.AllowActionsCreatedByGitHub = copyUIActionsValue(&state.Selected.GitHubOwnedAllowed)
		ui.AllowActionsByMarketplaceVerifiedCreators = copyUIActionsValue(&state.Selected.VerifiedAllowed)
		patterns := append([]string{}, state.Selected.PatternsAllowed...)
		ui.AllowOrBlockSpecifiedActionsAndReusableWorkflows = &patterns
	}
	return PreparedActionsExecution{General: ui, Raw: raw}, nil
}

func copyUIActionsValue[T any](value *T) *T {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}
