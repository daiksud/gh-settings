package github

import (
	"fmt"
	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

type commitMenuOption struct {
	choice         uimodel.CommitMessageChoice
	title, message string
}

var mergeCommitMenu = [...]commitMenuOption{
	{uimodel.DefaultMessage, "MERGE_MESSAGE", "PR_TITLE"},
	{uimodel.PullRequestTitle, "PR_TITLE", "BLANK"},
	{uimodel.PullRequestTitleAndDescription, "PR_TITLE", "PR_BODY"},
}

var squashCommitMenu = [...]commitMenuOption{
	{uimodel.DefaultMessage, "COMMIT_OR_PR_TITLE", "COMMIT_MESSAGES"},
	{uimodel.PullRequestTitle, "PR_TITLE", "BLANK"},
	{uimodel.PullRequestTitleAndCommitDetails, "PR_TITLE", "COMMIT_MESSAGES"},
	{uimodel.PullRequestTitleAndDescription, "PR_TITLE", "PR_BODY"},
}

func declareCommitMessage(choice *uimodel.CommitMessageChoice, allowed *bool, menu []commitMenuOption, path string) (*string, *string, error) {
	if choice == nil {
		return nil, nil, nil
	}
	if allowed != nil && !*allowed {
		return nil, nil, fmt.Errorf("general.pull_requests.%s.default_commit_message requires the merge method to be enabled", path)
	}
	for _, option := range menu {
		if option.choice == *choice {
			return &option.title, &option.message, nil
		}
	}
	return nil, nil, fmt.Errorf("unsupported general.pull_requests.%s.default_commit_message %q", path, *choice)
}

func observeCommitMessage(title, message *string, menu []commitMenuOption, path string) (*uimodel.CommitMessageChoice, error) {
	if title == nil && message == nil {
		return nil, nil
	}
	if title == nil || message == nil {
		return nil, fmt.Errorf("incomplete %s default commit message observation", path)
	}
	for _, option := range menu {
		if option.title == *title && option.message == *message {
			return &option.choice, nil
		}
	}
	return nil, fmt.Errorf("%s default commit message pair is outside UI choices", path)
}
