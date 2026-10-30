package github

import (
	"fmt"

	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

type preparedCommitMessage struct {
	Choice        *uimodel.CommitMessageChoice
	Pair          *settings.CommitMessagePair
	Normalization settings.CommitMessageNormalization
}

func prepareCommitMessageObservation(title, message *string, menu settings.CommitMessageMenu) (preparedCommitMessage, error) {
	fallback, valid := menu.DefaultChoice()
	if !valid {
		return preparedCommitMessage{}, fmt.Errorf("unsupported commit message menu %q", menu)
	}
	if title == nil && message == nil {
		return preparedCommitMessage{}, nil
	}
	if title == nil || message == nil {
		return preparedCommitMessage{}, fmt.Errorf("incomplete %s default commit message observation", menu)
	}
	raw := settings.CommitMessagePair{Title: *title, Message: *message}
	if choice, ok := settings.CommitMessageChoiceForPair(menu, raw); ok {
		return preparedCommitMessage{Choice: &choice, Pair: &raw}, nil
	}
	pair, ok := settings.CommitMessagePairForChoice(menu, fallback)
	if !ok {
		return preparedCommitMessage{}, fmt.Errorf("missing adopted %s fallback", menu)
	}
	return preparedCommitMessage{Choice: &fallback, Pair: &pair, Normalization: settings.CommitMessageNormalization{
		Path:     menu.Path(),
		RawTitle: raw.Title, RawMessage: raw.Message, Fallback: fallback,
	}}, nil
}
