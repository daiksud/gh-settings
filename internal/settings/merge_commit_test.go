package settings

import (
	"reflect"
	"testing"
)

func TestMergeCommitComparison(t *testing.T) {
	title, message := "PR_TITLE", "BLANK"
	current := MergeMethodsState{AllowMergeCommit: true, MergeCommitTitle: "MERGE_MESSAGE", MergeCommitMessage: "PR_TITLE"}
	observed := NewRepositoryObservation(nil, &current, nil)
	d := RepositoryDeclaration{MergeCommitTitle: &title, MergeCommitMessage: &message}
	want := []Change{{field: FieldID{path: "repository.merge_commit_title"}, values: stringPair{before: "MERGE_MESSAGE", after: "PR_TITLE"}}, {field: FieldID{path: "repository.merge_commit_message"}, values: stringPair{before: "PR_TITLE", after: "BLANK"}}}
	got := mustCompareObservation(t, observed, d)
	if got.Managed != 2 || !reflect.DeepEqual(got.Changes, want) {
		t.Fatalf("got=%+v", got)
	}
	equal := mustCompareObservation(t, NewRepositoryObservation(nil, &MergeMethodsState{AllowMergeCommit: true, MergeCommitTitle: title, MergeCommitMessage: message}, nil), d)
	if equal.Managed != 2 || len(equal.Changes) != 0 {
		t.Fatalf("equality=%+v", equal)
	}
	d.MergeCommitMessage = nil
	got = mustCompareObservation(t, observed, d)
	if got.Managed != 1 || len(got.Changes) != 1 || got.Changes[0].Field().Path() != "repository.merge_commit_title" {
		t.Fatalf("title-only=%+v", got)
	}
	got = mustCompareObservation(t, observed, RepositoryDeclaration{})
	if got.Managed != 0 || len(got.Changes) != 0 {
		t.Fatalf("unmanaged=%+v", got)
	}
}
