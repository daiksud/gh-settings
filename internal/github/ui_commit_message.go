package github

import (
	"fmt"
	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

func declareCommitMessage(choice *uimodel.CommitMessageChoice, allowed *bool, menu settings.CommitMessageMenu) (*string, *string, error) {
	if choice == nil {
		return nil, nil, nil
	}
	if allowed != nil && !*allowed {
		return nil, nil, fmt.Errorf("general.pull_requests.%s.default_commit_message requires the merge method to be enabled", menu)
	}
	if pair, ok := settings.CommitMessagePairForChoice(menu, *choice); ok {
		return &pair.Title, &pair.Message, nil
	}
	return nil, nil, fmt.Errorf("unsupported general.pull_requests.%s.default_commit_message %q", menu, *choice)
}

func observeCommitMessage(title, message *string, menu settings.CommitMessageMenu) (*uimodel.CommitMessageChoice, error) {
	if title == nil && message == nil {
		return nil, nil
	}
	if title == nil || message == nil {
		return nil, fmt.Errorf("incomplete %s default commit message observation", menu)
	}
	if choice, ok := settings.CommitMessageChoiceForPair(menu, settings.CommitMessagePair{Title: *title, Message: *message}); ok {
		return &choice, nil
	}
	return nil, fmt.Errorf("%s default commit message pair is outside UI choices", menu)
}
