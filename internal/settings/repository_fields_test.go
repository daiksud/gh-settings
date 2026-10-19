package settings

import (
	"reflect"
	"testing"
)

func TestOrdinaryStringFieldValidation(t *testing.T) {
	for index, name := range []string{"repository.description", "repository.homepage"} {
		field := repositoryStringFields[index]
		if field.name != name {
			t.Fatalf("field=%q want=%q", field.name, name)
		}
		for _, value := range []string{"", "example value"} {
			if !ValidRepositoryFieldValue(name, value) {
				t.Errorf("%s rejected ordinary string %q", name, value)
			}
		}
	}
}

func TestSquashStringFieldValidation(t *testing.T) {
	for index, test := range []struct {
		name    string
		allowed []string
	}{
		{name: "repository.squash_merge_commit_title", allowed: []string{"PR_TITLE", "COMMIT_OR_PR_TITLE"}},
		{name: "repository.squash_merge_commit_message", allowed: []string{"PR_BODY", "COMMIT_MESSAGES", "BLANK"}},
	} {
		field := mergeMethodsStringFields[index]
		if field.name != test.name {
			t.Fatalf("field=%q want=%q", field.name, test.name)
		}
		for _, value := range test.allowed {
			if !ValidRepositoryFieldValue(test.name, value) {
				t.Errorf("%s rejected %q", test.name, value)
			}
		}
		for _, value := range []string{"", "UNKNOWN"} {
			if ValidRepositoryFieldValue(test.name, value) {
				t.Errorf("%s accepted %q", test.name, value)
			}
		}
	}
}

func TestMergeStringFieldValidation(t *testing.T) {
	for _, test := range []struct {
		name    string
		allowed []string
	}{
		{name: "repository.merge_commit_title", allowed: []string{"PR_TITLE", "MERGE_MESSAGE"}},
		{name: "repository.merge_commit_message", allowed: []string{"PR_TITLE", "PR_BODY", "BLANK"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			for _, value := range test.allowed {
				if !ValidRepositoryFieldValue(test.name, value) {
					t.Errorf("%s rejected %q", test.name, value)
				}
			}
			for _, value := range []string{"", "UNKNOWN"} {
				if ValidRepositoryFieldValue(test.name, value) {
					t.Errorf("%s accepted %q", test.name, value)
				}
			}
		})
	}
}

func TestBooleanFieldDefinitions(t *testing.T) {
	for _, test := range []struct {
		name string
		want func(bool) RepositoryDeclaration
	}{
		{"repository.has_issues", func(value bool) RepositoryDeclaration { return RepositoryDeclaration{HasIssues: &value} }},
		{"repository.has_projects", func(value bool) RepositoryDeclaration { return RepositoryDeclaration{HasProjects: &value} }},
		{"repository.has_wiki", func(value bool) RepositoryDeclaration { return RepositoryDeclaration{HasWiki: &value} }},
		{"repository.has_discussions", func(value bool) RepositoryDeclaration { return RepositoryDeclaration{HasDiscussions: &value} }},
		{"repository.allow_merge_commit", func(value bool) RepositoryDeclaration { return RepositoryDeclaration{AllowMergeCommit: &value} }},
		{"repository.allow_squash_merge", func(value bool) RepositoryDeclaration { return RepositoryDeclaration{AllowSquashMerge: &value} }},
		{"repository.allow_rebase_merge", func(value bool) RepositoryDeclaration { return RepositoryDeclaration{AllowRebaseMerge: &value} }},
		{"repository.delete_branch_on_merge", func(value bool) RepositoryDeclaration { return RepositoryDeclaration{DeleteBranchOnMerge: &value} }},
		{"repository.allow_auto_merge", func(value bool) RepositoryDeclaration { return RepositoryDeclaration{AllowAutoMerge: &value} }},
		{"repository.allow_update_branch", func(value bool) RepositoryDeclaration { return RepositoryDeclaration{AllowUpdateBranch: &value} }},
	} {
		for _, value := range []bool{false, true} {
			if !ValidRepositoryFieldValue(test.name, value) {
				t.Errorf("%s rejected native boolean %t", test.name, value)
			}
			set, ok := RepositoryBoolSetter(test.name)
			if !ok || set == nil {
				t.Fatalf("missing setter for %s", test.name)
			}
			var declaration RepositoryDeclaration
			set(&declaration, value)
			if !reflect.DeepEqual(declaration, test.want(value)) {
				t.Errorf("%s assigned %+v, want %+v", test.name, declaration, test.want(value))
			}
		}
	}
}

func TestRepositoryBoolSetterRejectsOtherNames(t *testing.T) {
	for _, name := range []string{"", "has_issues", "repository.unknown", "repository.description", "repository.merge_commit_title", "topics.names"} {
		if set, ok := RepositoryBoolSetter(name); ok || set != nil {
			t.Errorf("unexpected boolean setter for %q", name)
		}
	}
}

func TestRepositoryFieldValidationRejectsUnknownNamesAndTypes(t *testing.T) {
	for _, test := range []struct {
		name  string
		value any
	}{
		{"repository.unknown", false},
		{"repository.unknown", "value"},
		{"repository.has_issues", "false"},
		{"repository.has_issues", nil},
		{"repository.description", false},
		{"repository.description", nil},
		{"repository.description", float64(1)},
		{"repository.description", []string{}},
		{"topics.names", "value"},
	} {
		if ValidRepositoryFieldValue(test.name, test.value) {
			t.Errorf("%s accepted %T value %#v", test.name, test.value, test.value)
		}
	}
}
