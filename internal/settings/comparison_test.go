package settings

import (
	"slices"
	"testing"
)

func TestCompareRepositoryChanges(t *testing.T) {
	want := []Change{
		{field: FieldID{path: "repository.description"}, values: stringPair{before: "old", after: ""}},
		{field: FieldID{path: "repository.homepage"}, values: stringPair{before: "old-url", after: ""}},
		{field: FieldID{path: "repository.default_branch"}, values: stringPair{before: "main", after: "release/v2"}},
		{field: FieldID{path: "repository.has_issues"}, values: boolPair{before: true, after: false}},
		{field: FieldID{path: "repository.has_projects"}, values: boolPair{before: true, after: false}},
		{field: FieldID{path: "repository.has_wiki"}, values: boolPair{before: true, after: false}},
		{field: FieldID{path: "repository.has_discussions"}, values: boolPair{before: true, after: false}},
		{field: FieldID{path: "repository.web_commit_signoff_required"}, values: boolPair{before: true, after: false}},
		{field: FieldID{path: "repository.is_template"}, values: boolPair{before: true, after: false}},
	}
	current := RepositoryState{
		Description: "old", Homepage: "old-url", DefaultBranch: "main", HasIssues: true, HasProjects: true, HasWiki: true,
		HasDiscussions: true, WebCommitSignoffRequired: true, IsTemplate: true,
	}
	empty, branch, disabled := "", "release/v2", false
	desired := RepositoryDeclaration{
		Description: &empty, Homepage: &empty, DefaultBranch: &branch, HasIssues: &disabled, HasProjects: &disabled, HasWiki: &disabled,
		HasDiscussions: &disabled, WebCommitSignoffRequired: &disabled, IsTemplate: &disabled,
	}
	got := mustCompareObservation(t, NewRepositoryObservation(&current, nil, nil), desired)
	if got.Managed != len(want) || !slices.Equal(got.Changes, want) {
		t.Fatalf("comparison=%#v, want %d managed fields and changes %#v", got, len(want), want)
	}
}

func TestCompareRepositoryDeclaredFields(t *testing.T) {
	old, empty, branch, enabled, disabled := "old", "", "main", true, false
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
		{name: "matching default branch", current: RepositoryState{DefaultBranch: branch}, desired: RepositoryDeclaration{DefaultBranch: &branch}, managed: 1},
		{
			name: "unchanged partial", current: current,
			desired: RepositoryDeclaration{Description: &old, HasProjects: &enabled}, managed: 2,
		},
		{
			name:    "unchanged declared zero values",
			desired: RepositoryDeclaration{Description: &empty, HasWiki: &disabled, HasDiscussions: &disabled, WebCommitSignoffRequired: &disabled, IsTemplate: &disabled}, managed: 5,
		},
		{
			name: "changed partial", current: current,
			desired: RepositoryDeclaration{Description: &old, HasIssues: &disabled}, managed: 2,
			changes: []Change{{field: FieldID{path: "repository.has_issues"}, values: boolPair{before: true, after: false}}},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := mustCompareObservation(t, NewRepositoryObservation(&test.current, nil, nil), test.desired)
			if got.Managed != test.managed || !slices.Equal(got.Changes, test.changes) {
				t.Fatalf("comparison=%#v, want %d managed fields and changes %#v", got, test.managed, test.changes)
			}
		})
	}
}

func mustCompareObservation(t *testing.T, observed RepositoryObservation, desired RepositoryDeclaration) Comparison {
	t.Helper()
	got, err := CompareRepositoryObservation(observed, desired)
	if err != nil {
		t.Fatal(err)
	}
	return got
}
