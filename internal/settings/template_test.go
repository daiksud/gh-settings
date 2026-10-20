package settings

import (
	"reflect"
	"testing"
)

func TestTemplateComparison(t *testing.T) {
	disabled := false
	current := RepositoryState{HasWiki: true, IsTemplate: true}
	if got := CompareRepository(current, RepositoryDeclaration{}); got.Managed != 0 || len(got.Changes) != 0 {
		t.Fatalf("omitted field managed: %+v", got)
	}
	desired := RepositoryDeclaration{HasWiki: &disabled, IsTemplate: &disabled}
	got := CompareRepository(current, desired)
	want := []Change{{field: FieldID{path: "repository.has_wiki"}, values: boolPair{before: true, after: false}}, {field: FieldID{path: "repository.is_template"}, values: boolPair{before: true, after: false}}}
	if got.Managed != 2 || !reflect.DeepEqual(got.Changes, want) {
		t.Fatalf("comparison=%+v", got)
	}
	got = CompareRepository(RepositoryState{}, RepositoryDeclaration{IsTemplate: &disabled})
	if got.Managed != 1 || len(got.Changes) != 0 {
		t.Fatalf("declared false=%+v", got)
	}
}
