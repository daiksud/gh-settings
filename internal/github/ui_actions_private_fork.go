package github

import (
	"fmt"
	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

// PreparedActionsPrivateFork retains acquired hidden companions without making
// them desired UI controls when fork workflows are disabled.
type PreparedActionsPrivateFork struct {
	General uimodel.ActionsGeneral
	Raw     [4]bool
}

func WithActionsPrivateForkPolicyUI(ui uimodel.ActionsGeneral, policy settings.PrivateForkDeclaration) (uimodel.ActionsGeneral, error) {
	if policy.Empty() {
		return ui, nil
	}
	var values [4]bool
	for i, value := range policy.Values() {
		if value == nil {
			return uimodel.ActionsGeneral{}, fmt.Errorf("private fork policy requires all four workflow values")
		}
		values[i] = *value
	}
	return withActionsPrivateForkValues(ui, values), nil
}

func declareUIActionsPrivateFork(ui *uimodel.ForkPullRequestWorkflows) (settings.PrivateForkDeclaration, error) {
	if ui == nil {
		return settings.PrivateForkDeclaration{}, nil
	}
	if ui.RunWorkflowsFromForkPullRequests != nil && !*ui.RunWorkflowsFromForkPullRequests && (ui.SendWriteTokensToWorkflowsFromForkPullRequests != nil || ui.SendSecretsAndVariablesToWorkflowsFromForkPullRequests != nil || ui.RequireApprovalForForkPullRequestWorkflows != nil) {
		return settings.PrivateForkDeclaration{}, fmt.Errorf("dependent fork workflow UI controls require enabled workflows or an omitted execution checkbox")
	}
	return settings.PrivateForkDeclaration{
		Run:         copyUIActionsValue(ui.RunWorkflowsFromForkPullRequests),
		WriteTokens: copyUIActionsValue(ui.SendWriteTokensToWorkflowsFromForkPullRequests),
		Secrets:     copyUIActionsValue(ui.SendSecretsAndVariablesToWorkflowsFromForkPullRequests),
		Approval:    copyUIActionsValue(ui.RequireApprovalForForkPullRequestWorkflows),
	}, nil
}

func WithActionsPrivateForkObservation(ui uimodel.ActionsGeneral, observation settings.PrivateForkObservation) (PreparedActionsPrivateFork, error) {
	values, err := observation.Values()
	if err != nil {
		return PreparedActionsPrivateFork{}, err
	}
	return PreparedActionsPrivateFork{General: withActionsPrivateForkValues(ui, values), Raw: values}, nil
}

func withActionsPrivateForkValues(ui uimodel.ActionsGeneral, values [4]bool) uimodel.ActionsGeneral {
	controls := &uimodel.ForkPullRequestWorkflows{RunWorkflowsFromForkPullRequests: copyUIActionsValue(&values[0])}
	if values[0] {
		controls.SendWriteTokensToWorkflowsFromForkPullRequests = copyUIActionsValue(&values[1])
		controls.SendSecretsAndVariablesToWorkflowsFromForkPullRequests = copyUIActionsValue(&values[2])
		controls.RequireApprovalForForkPullRequestWorkflows = copyUIActionsValue(&values[3])
	}
	ui.ForkPullRequestWorkflows = controls
	return ui
}
