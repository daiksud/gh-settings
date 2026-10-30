package settings

import "github.com/daiksud/gh-settings/internal/settings/uimodel"

const ForkApprovalPath = "repository.settings.actions.general.approval_for_running_fork_pull_request_workflows_from_contributors"
const DefaultForkApprovalChoice = uimodel.DefaultActionsForkApproval

var forkApprovalChoices = [...]struct {
	choice uimodel.ForkApprovalChoice
	value  string
}{
	{uimodel.RequireApprovalForFirstTimeContributorsWhoAreNewToGitHub, "first_time_contributors_new_to_github"},
	{uimodel.RequireApprovalForFirstTimeContributors, "first_time_contributors"},
	{uimodel.RequireApprovalForAllExternalContributors, "all_external_contributors"},
}

func ForkApprovalValueForChoice(choice uimodel.ForkApprovalChoice) (string, bool) {
	for _, option := range forkApprovalChoices {
		if option.choice == choice {
			return option.value, true
		}
	}
	return "", false
}

func ForkApprovalChoiceForValue(value string) (uimodel.ForkApprovalChoice, bool) {
	for _, option := range forkApprovalChoices {
		if option.value == value {
			return option.choice, true
		}
	}
	return "", false
}
