package github

import (
	"fmt"
	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

func declareUIActionsForkApproval(choice *uimodel.ForkApprovalChoice) (*string, error) {
	if choice == nil {
		return nil, nil
	}
	var value string
	switch *choice {
	case uimodel.RequireApprovalForFirstTimeContributorsWhoAreNewToGitHub:
		value = "first_time_contributors_new_to_github"
	case uimodel.RequireApprovalForFirstTimeContributors:
		value = "first_time_contributors"
	case uimodel.RequireApprovalForAllExternalContributors:
		value = "all_external_contributors"
	default:
		return nil, fmt.Errorf("unsupported UI fork workflow approval choice")
	}
	return &value, nil
}

func WithActionsForkApprovalPolicyUI(ui uimodel.ActionsGeneral, policy *string) (uimodel.ActionsGeneral, error) {
	if policy == nil {
		return ui, nil
	}
	var choice uimodel.ForkApprovalChoice
	switch *policy {
	case "first_time_contributors_new_to_github":
		choice = uimodel.RequireApprovalForFirstTimeContributorsWhoAreNewToGitHub
	case "first_time_contributors":
		choice = uimodel.RequireApprovalForFirstTimeContributors
	case "all_external_contributors":
		choice = uimodel.RequireApprovalForAllExternalContributors
	default:
		return uimodel.ActionsGeneral{}, fmt.Errorf("fork workflow approval is outside UI choices")
	}
	ui.ApprovalForRunningForkPullRequestWorkflowsFromContributors = &choice
	return ui, nil

}

func WithActionsForkApprovalObservation(ui uimodel.ActionsGeneral, observation settings.ForkApprovalObservation) (uimodel.ActionsGeneral, error) {
	value, err := observation.Value()
	if err != nil {
		return uimodel.ActionsGeneral{}, err
	}
	return WithActionsForkApprovalPolicyUI(ui, &value)
}
