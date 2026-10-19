package settings

import (
	"reflect"
	"testing"
)

func TestMergeMethodsPresenceAndComparison(t *testing.T) {
	yes, no := true, false
	for _, d := range []RepositoryDeclaration{{AllowMergeCommit: &no}, {AllowSquashMerge: &no}, {AllowRebaseMerge: &no}, {DeleteBranchOnMerge: &no}, {AllowAutoMerge: &no}, {AllowUpdateBranch: &no}} {
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

func TestDeleteBranchOnMergePresenceAndComparison(t *testing.T) {
	no := false
	current := MergeMethodsState{DeleteBranchOnMerge: true}
	got := CompareMergeMethods(current, RepositoryDeclaration{DeleteBranchOnMerge: &no})
	want := []Change{{Field: "repository.delete_branch_on_merge", Before: true, After: false}}
	if got.Managed != 1 || !reflect.DeepEqual(got.Changes, want) {
		t.Fatalf("comparison=%+v", got)
	}
	if got := CompareMergeMethods(current, RepositoryDeclaration{}); got.Managed != 0 || len(got.Changes) != 0 {
		t.Fatalf("omitted=%+v", got)
	}
	if got := CompareMergeMethods(MergeMethodsState{}, RepositoryDeclaration{DeleteBranchOnMerge: &no}); got.Managed != 1 || len(got.Changes) != 0 {
		t.Fatalf("false=%+v", got)
	}
}

func TestPROptionsComparison(t *testing.T) {
	no, yes := false, true
	current := MergeMethodsState{AllowAutoMerge: true, AllowUpdateBranch: true}
	got := CompareMergeMethods(current, RepositoryDeclaration{AllowAutoMerge: &no, AllowUpdateBranch: &no})
	want := []Change{{Field: "repository.allow_auto_merge", Before: true, After: false}, {Field: "repository.allow_update_branch", Before: true, After: false}}
	if got.Managed != 2 || !reflect.DeepEqual(got.Changes, want) {
		t.Fatalf("comparison=%+v", got)
	}
	for _, test := range []struct {
		desired RepositoryDeclaration
		managed int
	}{
		{desired: RepositoryDeclaration{AllowAutoMerge: &yes, AllowUpdateBranch: &yes}, managed: 2},
		{desired: RepositoryDeclaration{AllowAutoMerge: &yes}, managed: 1},
		{desired: RepositoryDeclaration{AllowUpdateBranch: &yes}, managed: 1},
	} {
		got := CompareMergeMethods(current, test.desired)
		if got.Managed != test.managed || len(got.Changes) != 0 {
			t.Fatalf("equal=%+v", got)
		}
	}
	if got := CompareMergeMethods(current, RepositoryDeclaration{}); got.Managed != 0 || len(got.Changes) != 0 {
		t.Fatalf("omitted=%+v", got)
	}
}
