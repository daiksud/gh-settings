package settings

import (
	"reflect"
	"testing"
)

func TestSquashCommitComparison(t *testing.T) {
	title, message := "PR_TITLE", "BLANK"
	current := MergeMethodsState{SquashMergeCommitTitle: "COMMIT_OR_PR_TITLE", SquashMergeCommitMessage: "COMMIT_MESSAGES"}
	observed := NewRepositoryObservation(nil, &current, nil)
	d := RepositoryDeclaration{SquashMergeCommitTitle: &title, SquashMergeCommitMessage: &message}
	want := []Change{{field: FieldID{path: "repository.squash_merge_commit_title"}, values: stringPair{before: "COMMIT_OR_PR_TITLE", after: "PR_TITLE"}}, {field: FieldID{path: "repository.squash_merge_commit_message"}, values: stringPair{before: "COMMIT_MESSAGES", after: "BLANK"}}}
	got := mustCompareObservation(t, observed, d)
	if got.Managed != 2 || !reflect.DeepEqual(got.Changes, want) {
		t.Fatalf("got=%+v", got)
	}
	equal := mustCompareObservation(t, NewRepositoryObservation(nil, &MergeMethodsState{SquashMergeCommitTitle: title, SquashMergeCommitMessage: message}, nil), d)
	if equal.Managed != 2 || len(equal.Changes) != 0 {
		t.Fatalf("equality=%+v", equal)
	}
	d.SquashMergeCommitMessage = nil
	got = mustCompareObservation(t, observed, d)
	if got.Managed != 1 || len(got.Changes) != 1 || got.Changes[0].Field().Path() != "repository.squash_merge_commit_title" {
		t.Fatalf("title-only=%+v", got)
	}
	got = mustCompareObservation(t, observed, RepositoryDeclaration{})
	if got.Managed != 0 || len(got.Changes) != 0 {
		t.Fatalf("unmanaged=%+v", got)
	}
}
