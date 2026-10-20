package settings

import (
	"reflect"
	"strings"
	"testing"
)

func TestDeclaredRepositoryFields(t *testing.T) {
	empty, disabled, title := "", false, "PR_TITLE"
	desired := RepositoryDeclaration{Description: &empty, HasIssues: &disabled, AllowMergeCommit: &disabled, MergeCommitTitle: &title}
	var paths []string
	for _, field := range DeclaredRepositoryFields(desired) {
		paths = append(paths, field.Path())
	}
	want := []string{"repository.description", "repository.has_issues", "repository.allow_merge_commit", "repository.merge_commit_title"}
	if !reflect.DeepEqual(paths, want) {
		t.Fatalf("fields=%v, want %v", paths, want)
	}
	if fields := DeclaredRepositoryFields(RepositoryDeclaration{}); len(fields) != 0 {
		t.Fatalf("unmanaged fields=%v", fields)
	}
}

func TestCompareRepositoryObservationRequiresAcquiredFields(t *testing.T) {
	disabled := false
	desired := RepositoryDeclaration{HasIssues: &disabled, AllowMergeCommit: &disabled}
	for _, test := range []struct {
		name    string
		base    *RepositoryState
		merge   *MergeMethodsState
		missing string
	}{
		{"unread REST false", nil, &MergeMethodsState{}, "repository.has_issues"},
		{"unread merge false after REST comparison", &RepositoryState{HasIssues: true}, nil, "repository.allow_merge_commit"},
		{"observed false", &RepositoryState{}, &MergeMethodsState{}, ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			observation := NewRepositoryObservation(test.base, test.merge)
			got, err := CompareRepositoryObservation(observation, desired)
			if test.missing != "" {
				if err == nil || !strings.Contains(err.Error(), test.missing) || !reflect.DeepEqual(got, Comparison{}) {
					t.Fatalf("comparison=%+v error=%v", got, err)
				}
			} else if err != nil || got.Managed != 2 || len(got.Changes) != 0 {
				t.Fatalf("comparison=%+v error=%v", got, err)
			}
		})
	}
}

func TestRepositoryObservationFields(t *testing.T) {
	observed := NewRepositoryObservation(&RepositoryState{}, nil)
	var paths []string
	for _, field := range observed.Fields() {
		paths = append(paths, field.Path())
	}
	want := []string{"repository.description", "repository.homepage", "repository.default_branch", "repository.has_issues", "repository.has_projects", "repository.has_wiki", "repository.has_discussions", "repository.web_commit_signoff_required", "repository.is_template"}
	if !reflect.DeepEqual(paths, want) {
		t.Fatalf("acquired=%v, want %v", paths, want)
	}
}

func TestCompareRepositoryObservationPreservesOrderAndNativeValues(t *testing.T) {
	empty, disabled, title := "", false, "MERGE_MESSAGE"
	observed := NewRepositoryObservation(&RepositoryState{Description: "old", HasIssues: true}, &MergeMethodsState{MergeCommitTitle: "PR_TITLE"})
	desired := RepositoryDeclaration{Description: &empty, HasIssues: &disabled, AllowMergeCommit: &disabled, MergeCommitTitle: &title}
	got, err := CompareRepositoryObservation(observed, desired)
	want := Comparison{Managed: 4, Changes: []Change{
		{field: FieldID{path: "repository.description"}, values: stringPair{before: "old", after: ""}},
		{field: FieldID{path: "repository.has_issues"}, values: boolPair{before: true, after: false}},
		{field: FieldID{path: "repository.merge_commit_title"}, values: stringPair{before: "PR_TITLE", after: "MERGE_MESSAGE"}},
	}}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("comparison=%+v error=%v, want %+v", got, err, want)
	}
}

func TestRepositoryObservationDeclarationRequiresCompleteValues(t *testing.T) {
	for _, test := range []struct {
		name    string
		base    *RepositoryState
		merge   *MergeMethodsState
		missing string
	}{
		{"unread base", nil, &MergeMethodsState{}, "repository.description"},
		{"unread merge", &RepositoryState{}, nil, "repository.allow_merge_commit"},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := NewRepositoryObservation(test.base, test.merge).Declaration()
			if err == nil || !strings.Contains(err.Error(), test.missing) || !reflect.DeepEqual(got, RepositoryDeclaration{}) {
				t.Fatalf("declaration=%+v error=%v", got, err)
			}
		})
	}
}

func TestCompleteDeclarationCopiesObservedValues(t *testing.T) {
	observed := NewRepositoryObservation(&RepositoryState{Description: "before"}, &MergeMethodsState{SquashMergeCommitTitle: "PR_TITLE", SquashMergeCommitMessage: "BLANK", MergeCommitTitle: "PR_TITLE", MergeCommitMessage: "BLANK"})
	declaration, err := observed.Declaration()
	if err != nil {
		t.Fatal(err)
	}
	*declaration.Description = "changed"
	*declaration.AllowMergeCommit = true
	*declaration.MergeCommitTitle = "MERGE_MESSAGE"
	again, err := observed.Declaration()
	if err != nil || *again.Description != "before" || *again.AllowMergeCommit || *again.MergeCommitTitle != "PR_TITLE" {
		t.Fatalf("observation changed through declaration: %+v error=%v", again, err)
	}
}
