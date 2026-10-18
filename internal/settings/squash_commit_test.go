package settings

import (
	"reflect"
	"testing"
)

func TestSquashCommitComparison(t *testing.T) {
	title, message := "PR_TITLE", "BLANK"
	current := MergeMethodsState{SquashMergeCommitTitle: "COMMIT_OR_PR_TITLE", SquashMergeCommitMessage: "COMMIT_MESSAGES"}
	d := RepositoryDeclaration{SquashMergeCommitTitle: &title, SquashMergeCommitMessage: &message}
	want := []Change{{Field: "repository.squash_merge_commit_title", Before: "COMMIT_OR_PR_TITLE", After: "PR_TITLE"}, {Field: "repository.squash_merge_commit_message", Before: "COMMIT_MESSAGES", After: "BLANK"}}
	got := CompareMergeMethods(current, d)
	if got.Managed != 2 || !reflect.DeepEqual(got.Changes, want) {
		t.Fatalf("got=%+v", got)
	}
	equal := CompareMergeMethods(MergeMethodsState{SquashMergeCommitTitle: title, SquashMergeCommitMessage: message}, d)
	if equal.Managed != 2 || len(equal.Changes) != 0 {
		t.Fatalf("equality=%+v", equal)
	}
	d.SquashMergeCommitMessage = nil
	got = CompareMergeMethods(current, d)
	if got.Managed != 1 || len(got.Changes) != 1 || got.Changes[0].Field != "repository.squash_merge_commit_title" {
		t.Fatalf("title-only=%+v", got)
	}
	got = CompareMergeMethods(current, RepositoryDeclaration{})
	if got.Managed != 0 || len(got.Changes) != 0 {
		t.Fatalf("unmanaged=%+v", got)
	}
}
