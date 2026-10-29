package settings

import (
	"fmt"

	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

// NormalizedCommitMessageChange is one actionable UI decision owning both native
// title/message members. A fallback match is not an independently observed match.
type NormalizedCommitMessageChange struct {
	menu          CommitMessageMenu
	normalization CommitMessageNormalization
	desired       uimodel.CommitMessageChoice
}

func NewNormalizedCommitMessageChange(menu CommitMessageMenu, normalization CommitMessageNormalization, desired uimodel.CommitMessageChoice) (NormalizedCommitMessageChange, error) {
	fallback, valid := menu.DefaultChoice()
	if !valid || normalization.Path != menu.Path() || normalization.Fallback != fallback {
		return NormalizedCommitMessageChange{}, fmt.Errorf("invalid commit message normalization identity or fallback")
	}
	if _, supported := CommitMessageChoiceForPair(menu, CommitMessagePair{Title: normalization.RawTitle, Message: normalization.RawMessage}); supported {
		return NormalizedCommitMessageChange{}, fmt.Errorf("supported commit message pair cannot carry normalization provenance")
	}
	if _, valid := CommitMessagePairForChoice(menu, desired); !valid {
		return NormalizedCommitMessageChange{}, fmt.Errorf("unsupported desired commit message choice")
	}
	return NormalizedCommitMessageChange{menu: menu, normalization: normalization, desired: desired}, nil
}

func (change NormalizedCommitMessageChange) Menu() CommitMessageMenu { return change.menu }
func (change NormalizedCommitMessageChange) Normalization() CommitMessageNormalization {
	return change.normalization
}
func (change NormalizedCommitMessageChange) Values() (uimodel.CommitMessageChoice, uimodel.CommitMessageChoice) {
	return change.normalization.Fallback, change.desired
}
func (change NormalizedCommitMessageChange) NativeFields() []FieldID {
	var prefix string
	switch change.menu {
	case MergeCommitMenu:
		prefix = "repository.merge_commit"
	case SquashCommitMenu:
		prefix = "repository.squash_merge_commit"
	default:
		return nil
	}
	return []FieldID{{path: prefix + "_title"}, {path: prefix + "_message"}}
}
