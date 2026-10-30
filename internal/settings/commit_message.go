package settings

import "github.com/daiksud/gh-settings/internal/settings/uimodel"

type CommitMessageMenu string

const (
	MergeCommitMenu  CommitMessageMenu = "merge_commit"
	SquashCommitMenu CommitMessageMenu = "squash_merge"
)

func (menu CommitMessageMenu) Path() string {
	switch menu {
	case MergeCommitMenu, SquashCommitMenu:
		return "repository.settings.general.pull_requests." + string(menu) + ".default_commit_message"
	default:
		return ""
	}
}

func (menu CommitMessageMenu) DefaultChoice() (uimodel.CommitMessageChoice, bool) {
	switch menu {
	case MergeCommitMenu:
		return uimodel.DefaultMergeCommitMessage, true
	case SquashCommitMenu:
		return uimodel.DefaultSquashMergeCommitMessage, true
	default:
		return "", false
	}
}

type CommitMessagePair struct{ Title, Message string }

// CommitMessageNormalization retains both acquired members of one UI menu.
// The effective pair is separate; neither raw member becomes desired intent.
type CommitMessageNormalization struct {
	Path       string                      `json:"path"`
	RawTitle   string                      `json:"raw_title"`
	RawMessage string                      `json:"raw_message"`
	Fallback   uimodel.CommitMessageChoice `json:"fallback"`
}

type commitMessageOption struct {
	choice uimodel.CommitMessageChoice
	pair   CommitMessagePair
}

var mergeCommitMessages = [...]commitMessageOption{
	{uimodel.DefaultMessage, CommitMessagePair{"MERGE_MESSAGE", "PR_TITLE"}},
	{uimodel.PullRequestTitle, CommitMessagePair{"PR_TITLE", "BLANK"}},
	{uimodel.PullRequestTitleAndDescription, CommitMessagePair{"PR_TITLE", "PR_BODY"}},
}

var squashCommitMessages = [...]commitMessageOption{
	{uimodel.DefaultMessage, CommitMessagePair{"COMMIT_OR_PR_TITLE", "COMMIT_MESSAGES"}},
	{uimodel.PullRequestTitle, CommitMessagePair{"PR_TITLE", "BLANK"}},
	{uimodel.PullRequestTitleAndCommitDetails, CommitMessagePair{"PR_TITLE", "COMMIT_MESSAGES"}},
	{uimodel.PullRequestTitleAndDescription, CommitMessagePair{"PR_TITLE", "PR_BODY"}},
}

func commitMessageOptions(menu CommitMessageMenu) []commitMessageOption {
	switch menu {
	case MergeCommitMenu:
		return mergeCommitMessages[:]
	case SquashCommitMenu:
		return squashCommitMessages[:]
	default:
		return nil
	}
}

func CommitMessagePairForChoice(menu CommitMessageMenu, choice uimodel.CommitMessageChoice) (CommitMessagePair, bool) {
	for _, option := range commitMessageOptions(menu) {
		if option.choice == choice {
			return option.pair, true
		}
	}
	return CommitMessagePair{}, false
}

func CommitMessageChoiceForPair(menu CommitMessageMenu, pair CommitMessagePair) (uimodel.CommitMessageChoice, bool) {
	for _, option := range commitMessageOptions(menu) {
		if option.pair == pair {
			return option.choice, true
		}
	}
	return "", false
}
