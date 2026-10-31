package settings

import "fmt"

// CommitMessageContext retains the complete native pair needed to present one
// UI menu. Changes remain the original declared-field comparisons.
type CommitMessageContext struct {
	Menu   CommitMessageMenu `json:"menu"`
	Before CommitMessagePair `json:"before"`
	After  CommitMessagePair `json:"after"`
}

// CommitMessageMenuForField identifies a native component without treating it
// as a separately selectable UI control.
func CommitMessageMenuForField(path string) (CommitMessageMenu, bool) {
	switch path {
	case "repository.merge_commit_title", "repository.merge_commit_message":
		return MergeCommitMenu, true
	case "repository.squash_merge_commit_title", "repository.squash_merge_commit_message":
		return SquashCommitMenu, true
	default:
		return "", false
	}
}

func (c CommitMessageContext) Choices() (string, string, error) {
	before, ok := CommitMessageChoiceForPair(c.Menu, c.Before)
	if !ok {
		return "", "", fmt.Errorf("unsupported observed commit message pair")
	}
	after, ok := CommitMessageChoiceForPair(c.Menu, c.After)
	if !ok || before == after {
		return "", "", fmt.Errorf("unsupported or unchanged desired commit message pair")
	}
	return string(before), string(after), nil
}

func commitMessageContexts(observed, desired RepositoryDeclaration) ([]CommitMessageContext, error) {
	var contexts []CommitMessageContext
	for _, c := range []struct {
		menu           CommitMessageMenu
		bt, bm, at, am *string
	}{
		{SquashCommitMenu, observed.SquashMergeCommitTitle, observed.SquashMergeCommitMessage, desired.SquashMergeCommitTitle, desired.SquashMergeCommitMessage},
		{MergeCommitMenu, observed.MergeCommitTitle, observed.MergeCommitMessage, desired.MergeCommitTitle, desired.MergeCommitMessage},
	} {
		if c.at == nil && c.am == nil {
			continue
		}
		// Internal component-only declarations are not a selectable UI menu.
		// Leave their existing comparison contract intact; display must not
		// manufacture the missing choice.
		if c.at == nil || c.am == nil {
			continue
		}
		if c.bt == nil || c.bm == nil {
			return nil, fmt.Errorf("%s observation requires complete commit message pair", c.menu)
		}
		context := CommitMessageContext{Menu: c.menu, Before: CommitMessagePair{Title: *c.bt, Message: *c.bm}, After: CommitMessagePair{Title: *c.at, Message: *c.am}}
		if context.Before == context.After {
			continue
		}
		if _, _, err := context.Choices(); err != nil {
			return nil, err
		}
		contexts = append(contexts, context)
	}
	return contexts, nil
}
