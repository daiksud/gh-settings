package settings

import (
	"reflect"
	"testing"

	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

func TestNormalizedCommitComparisonOwnsMenuWithoutDuplicateComponents(t *testing.T) {
	state := normalizedCommitComparisonState()
	observed := NewRepositoryObservation(nil, &state, nil)
	for _, menu := range []CommitMessageMenu{SquashCommitMenu, MergeCommitMenu} {
		for _, choice := range []string{"default_message", "pull_request_title"} {
			desired := commitComparisonIntent(menu, choice)
			comparison, err := CompareRepositoryObservation(observed, desired)
			if err != nil || comparison.Managed != 2 || len(comparison.CommitMessages) != 1 || len(comparison.Changes) != 0 || comparison.CommitMessages[0].Menu() != menu {
				t.Fatal("normalized menu became equality or duplicate components", comparison, err)
			}
			before, after := comparison.CommitMessages[0].Values()
			if before != "default_message" || string(after) != choice {
				t.Fatal("normalized comparison lost UI intent", before, after)
			}
			if menu == SquashCommitMenu && (desired.SquashMergeCommitTitle == nil || desired.SquashMergeCommitMessage == nil) || menu == MergeCommitMenu && (desired.MergeCommitTitle == nil || desired.MergeCommitMessage == nil) {
				t.Fatal("comparison modified incoming desired field ownership")
			}
		}
	}
	desired := commitComparisonIntent(SquashCommitMenu, "default_message")
	other := commitComparisonIntent(MergeCommitMenu, "default_message")
	desired.MergeCommitTitle, desired.MergeCommitMessage = other.MergeCommitTitle, other.MergeCommitMessage
	comparison, err := CompareRepositoryObservation(observed, desired)
	if err != nil || comparison.Managed != 4 || len(comparison.CommitMessages) != 2 || len(comparison.Changes) != 0 || comparison.CommitMessages[0].Menu() != SquashCommitMenu || comparison.CommitMessages[1].Menu() != MergeCommitMenu {
		t.Fatal("two normalized menus lost ownership or stable ordering", comparison, err)
	}
}

func TestNormalizedCommitComparisonPreservesOmissionAndOrdinaryValues(t *testing.T) {
	state := normalizedCommitComparisonState()
	no := false
	comparison, err := CompareRepositoryObservation(NewRepositoryObservation(nil, &state, nil), RepositoryDeclaration{AllowSquashMerge: &no})
	if err != nil || comparison.Managed != 1 || len(comparison.Changes) != 1 || len(comparison.CommitMessages) != 0 {
		t.Fatal("omitted menu provenance affected ordinary managed boolean", comparison, err)
	}
	state.SquashCommitNormalization, state.MergeCommitNormalization = CommitMessageNormalization{}, CommitMessageNormalization{}
	for _, choice := range []string{"default_message", "pull_request_title"} {
		comparison, err := CompareRepositoryObservation(NewRepositoryObservation(nil, &state, nil), commitComparisonIntent(SquashCommitMenu, choice))
		want := 0
		if choice != "default_message" {
			want = 2
		}
		if err != nil || comparison.Managed != 2 || len(comparison.Changes) != want || len(comparison.CommitMessages) != 0 {
			t.Fatal("ordinary native comparison changed", comparison, err)
		}
	}
}

func TestNormalizedCommitComparisonRejectsInconsistentProvenance(t *testing.T) {
	for _, failure := range []string{"path", "fallback", "supported raw", "effective pair", "unrepresentable desired", "partial desired"} {
		state := normalizedCommitComparisonState()
		desired := commitComparisonIntent(SquashCommitMenu, "default_message")
		before, after := "old description", "new description"
		desired.Description = &after
		switch failure {
		case "path":
			state.SquashCommitNormalization.Path = "unmanaged"
		case "fallback":
			state.SquashCommitNormalization.Fallback = uimodel.PullRequestTitle
		case "supported raw":
			state.SquashCommitNormalization.RawTitle, state.SquashCommitNormalization.RawMessage = "PR_TITLE", "BLANK"
		case "effective pair":
			state.SquashMergeCommitMessage = "PR_BODY"
		case "unrepresentable desired":
			blank := "BLANK"
			desired.SquashMergeCommitMessage = &blank
		case "partial desired":
			desired.SquashMergeCommitMessage = nil
		}
		comparison, err := CompareRepositoryObservation(NewRepositoryObservation(&RepositoryState{Description: before, DefaultBranch: "main"}, &state, nil), desired)
		if err == nil || !reflect.DeepEqual(comparison, Comparison{}) {
			t.Fatal("inconsistent menu exposed a partial comparison", failure, comparison, err)
		}
	}
}

func normalizedCommitComparisonState() MergeMethodsState {
	return MergeMethodsState{AllowMergeCommit: true, AllowSquashMerge: true, SquashMergeCommitTitle: "COMMIT_OR_PR_TITLE", SquashMergeCommitMessage: "COMMIT_MESSAGES", MergeCommitTitle: "MERGE_MESSAGE", MergeCommitMessage: "PR_TITLE", SquashCommitNormalization: CommitMessageNormalization{Path: "repository.settings.general.pull_requests.squash_merge.default_commit_message", RawTitle: "COMMIT_OR_PR_TITLE", RawMessage: "BLANK", Fallback: uimodel.DefaultMessage}, MergeCommitNormalization: CommitMessageNormalization{Path: "repository.settings.general.pull_requests.merge_commit.default_commit_message", RawTitle: "MERGE_MESSAGE", RawMessage: "BLANK", Fallback: uimodel.DefaultMessage}}
}

func commitComparisonIntent(menu CommitMessageMenu, choice string) RepositoryDeclaration {
	var title, message string
	if menu == SquashCommitMenu {
		title, message = "COMMIT_OR_PR_TITLE", "COMMIT_MESSAGES"
	} else {
		title, message = "MERGE_MESSAGE", "PR_TITLE"
	}
	if choice == "pull_request_title" {
		title, message = "PR_TITLE", "BLANK"
	}
	if menu == SquashCommitMenu {
		return RepositoryDeclaration{SquashMergeCommitTitle: &title, SquashMergeCommitMessage: &message}
	}
	return RepositoryDeclaration{MergeCommitTitle: &title, MergeCommitMessage: &message}
}
