package settings

import (
	"reflect"
	"testing"
)

func TestMergeCommitComparison(t *testing.T) {
	title, message := "PR_TITLE", "BLANK"
	current := MergeMethodsState{MergeCommitTitle: "MERGE_MESSAGE", MergeCommitMessage: "PR_TITLE"}
	d := RepositoryDeclaration{MergeCommitTitle: &title, MergeCommitMessage: &message}
	want := []Change{{field: FieldID{path: "repository.merge_commit_title"}, Before: "MERGE_MESSAGE", After: "PR_TITLE"}, {field: FieldID{path: "repository.merge_commit_message"}, Before: "PR_TITLE", After: "BLANK"}}
	got := CompareMergeMethods(current, d)
	if got.Managed != 2 || !reflect.DeepEqual(got.Changes, want) {
		t.Fatalf("got=%+v", got)
	}
	equal := CompareMergeMethods(MergeMethodsState{MergeCommitTitle: title, MergeCommitMessage: message}, d)
	if equal.Managed != 2 || len(equal.Changes) != 0 {
		t.Fatalf("equality=%+v", equal)
	}
	d.MergeCommitMessage = nil
	got = CompareMergeMethods(current, d)
	if got.Managed != 1 || len(got.Changes) != 1 || got.Changes[0].Field().Path() != "repository.merge_commit_title" {
		t.Fatalf("title-only=%+v", got)
	}
	got = CompareMergeMethods(current, RepositoryDeclaration{})
	if got.Managed != 0 || len(got.Changes) != 0 {
		t.Fatalf("unmanaged=%+v", got)
	}
}
