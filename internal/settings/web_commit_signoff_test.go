package settings

import (
	"reflect"
	"testing"
)

func TestWebCommitSignoffComparison(t *testing.T) {
	disabled := false
	current := RepositoryState{HasWiki: true, WebCommitSignoffRequired: true}
	if got := CompareRepository(current, RepositoryDeclaration{}); got.Managed != 0 || len(got.Changes) != 0 {
		t.Fatalf("omitted field managed: %+v", got)
	}
	desired := RepositoryDeclaration{HasWiki: &disabled, WebCommitSignoffRequired: &disabled}
	got := CompareRepository(current, desired)
	want := []Change{{field: FieldID{path: "repository.has_wiki"}, values: boolPair{before: true, after: false}}, {field: FieldID{path: "repository.web_commit_signoff_required"}, values: boolPair{before: true, after: false}}}
	if got.Managed != 2 || !reflect.DeepEqual(got.Changes, want) {
		t.Fatalf("comparison=%+v", got)
	}
	got = CompareRepository(RepositoryState{}, RepositoryDeclaration{WebCommitSignoffRequired: &disabled})
	if got.Managed != 1 || len(got.Changes) != 0 {
		t.Fatalf("declared false=%+v", got)
	}
}
