package settings

import (
	"slices"
	"testing"
)

func TestCompareChanges(t *testing.T) {
	want := []Change{
		{field: FieldID{path: "repository.description"}, values: stringPair{before: "old", after: ""}},
		{field: FieldID{path: "repository.homepage"}, values: stringPair{before: "old-url", after: ""}},
		{field: FieldID{path: "repository.has_issues"}, values: boolPair{before: true, after: false}},
		{field: FieldID{path: "repository.has_projects"}, values: boolPair{before: true, after: false}},
		{field: FieldID{path: "repository.has_wiki"}, values: boolPair{before: true, after: false}},
	}
	current := RepositoryState{
		Description: "old", Homepage: "old-url", HasIssues: true, HasProjects: true, HasWiki: true,
	}
	empty, disabled := "", false
	desired := RepositoryDeclaration{
		Description: &empty, Homepage: &empty, HasIssues: &disabled, HasProjects: &disabled, HasWiki: &disabled,
	}
	got := CompareRepository(current, desired)
	if got.Managed != 5 || !slices.Equal(got.Changes, want) {
		t.Fatalf("comparison=%#v, want 5 managed fields and changes %#v", got, want)
	}
}

func TestCompareRepositoryDeclaredFields(t *testing.T) {
	old, empty, enabled, disabled := "old", "", true, false
	current := RepositoryState{
		Description: "old", Homepage: "old-url", HasIssues: true, HasProjects: true, HasWiki: true,
	}
	tests := []struct {
		name    string
		current RepositoryState
		desired RepositoryDeclaration
		managed int
		changes []Change
	}{
		{name: "unmanaged", current: current},
		{
			name: "unchanged partial", current: current,
			desired: RepositoryDeclaration{Description: &old, HasProjects: &enabled}, managed: 2,
		},
		{
			name:    "unchanged declared zero values",
			desired: RepositoryDeclaration{Description: &empty, HasWiki: &disabled}, managed: 2,
		},
		{
			name: "changed partial", current: current,
			desired: RepositoryDeclaration{Description: &old, HasIssues: &disabled}, managed: 2,
			changes: []Change{{field: FieldID{path: "repository.has_issues"}, values: boolPair{before: true, after: false}}},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := CompareRepository(test.current, test.desired)
			if got.Managed != test.managed || !slices.Equal(got.Changes, test.changes) {
				t.Fatalf("comparison=%#v, want %d managed fields and changes %#v", got, test.managed, test.changes)
			}
		})
	}
}
