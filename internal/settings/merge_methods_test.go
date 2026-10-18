package settings

import (
	"reflect"
	"testing"
)

func TestMergeMethodsPresenceAndComparison(t *testing.T) {
	yes, no := true, false
	for _, d := range []RepositoryDeclaration{{AllowMergeCommit: &no}, {AllowSquashMerge: &no}, {AllowRebaseMerge: &no}} {
		if d.Empty() || !d.BaseEmpty() || d.MergeMethodsEmpty() {
			t.Fatalf("lost false presence: %+v", d)
		}
	}
	if !(RepositoryDeclaration{}).MergeMethodsEmpty() || (RepositoryDeclaration{HasIssues: &no}).BaseEmpty() {
		t.Fatal("base/merge selection mixed")
	}
	current := MergeMethodsState{AllowMergeCommit: true, AllowRebaseMerge: true}
	got := CompareMergeMethods(current, RepositoryDeclaration{AllowMergeCommit: &no, AllowSquashMerge: &yes, AllowRebaseMerge: &no})
	want := []Change{{Field: "repository.allow_merge_commit", Before: true, After: false}, {Field: "repository.allow_squash_merge", Before: false, After: true}, {Field: "repository.allow_rebase_merge", Before: true, After: false}}
	if got.Managed != 3 || !reflect.DeepEqual(got.Changes, want) {
		t.Fatalf("got=%+v", got)
	}
	got = CompareMergeMethods(current, RepositoryDeclaration{AllowSquashMerge: &no})
	if got.Managed != 1 || len(got.Changes) != 0 {
		t.Fatalf("false/equality=%+v", got)
	}
	if got := CompareMergeMethods(current, RepositoryDeclaration{}); got.Managed != 0 || len(got.Changes) != 0 {
		t.Fatalf("unmanaged=%+v", got)
	}
}
