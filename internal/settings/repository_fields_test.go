package settings

import "testing"

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

func TestBooleanFieldValidation(t *testing.T) {
	for _, name := range []string{
		"repository.has_issues", "repository.has_projects", "repository.has_wiki", "repository.has_discussions",
		"repository.allow_merge_commit", "repository.allow_squash_merge", "repository.allow_rebase_merge",
		"repository.delete_branch_on_merge", "repository.allow_auto_merge", "repository.allow_update_branch",
	} {
		for _, value := range []bool{false, true} {
			if !ValidRepositoryFieldValue(name, value) {
				t.Errorf("%s rejected native boolean %t", name, value)
			}
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
