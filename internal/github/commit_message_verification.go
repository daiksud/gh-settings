package github

import "github.com/daiksud/gh-settings/internal/settings"

// Selective reads contain only acquired menus, never fabricated repository state.
type repositoryCommitObservations map[settings.CommitMessageMenu]preparedCommitMessage

func (observed repositoryCommitObservations) rawPair(menu settings.CommitMessageMenu) (settings.CommitMessagePair, bool) {
	value, present := observed[menu]
	if !present || value.Pair == nil || value.Choice == nil {
		return settings.CommitMessagePair{}, false
	}
	if value.Normalization == (settings.CommitMessageNormalization{}) {
		choice, valid := settings.CommitMessageChoiceForPair(menu, *value.Pair)
		return *value.Pair, valid && choice == *value.Choice
	}
	if _, err := settings.NewNormalizedCommitMessageChange(menu, value.Normalization, value.Normalization.Fallback); err != nil {
		return settings.CommitMessagePair{}, false
	}
	fallback, valid := settings.CommitMessagePairForChoice(menu, value.Normalization.Fallback)
	if !valid || *value.Pair != fallback || *value.Choice != value.Normalization.Fallback {
		return settings.CommitMessagePair{}, false
	}
	return settings.CommitMessagePair{Title: value.Normalization.RawTitle, Message: value.Normalization.RawMessage}, true
}

func repositoryCommitMessagesMatch(before, actual repositoryCommitObservations, desired settings.RepositoryDeclaration) bool {
	for _, entry := range []struct {
		menu           settings.CommitMessageMenu
		title, message *string
	}{
		{settings.SquashCommitMenu, desired.SquashMergeCommitTitle, desired.SquashMergeCommitMessage},
		{settings.MergeCommitMenu, desired.MergeCommitTitle, desired.MergeCommitMessage},
	} {
		if entry.title == nil && entry.message == nil {
			continue
		}
		pair, valid := actual.rawPair(entry.menu)
		if !valid {
			return false
		}
		var expected settings.CommitMessagePair
		if entry.title == nil || entry.message == nil {
			expected, valid = before.rawPair(entry.menu)
			if !valid {
				return false
			}
		}
		if entry.title != nil {
			expected.Title = *entry.title
		}
		if entry.message != nil {
			expected.Message = *entry.message
		}
		if pair != expected {
			return false
		}
	}
	return true
}
