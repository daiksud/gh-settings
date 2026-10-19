package settings

import "testing"

func TestOrdinaryStringFieldValidation(t *testing.T) {
	for index, name := range []string{"repository.description", "repository.homepage"} {
		field := repositoryStringFields[index]
		if field.name != name {
			t.Fatalf("field=%q want=%q", field.name, name)
		}
		for _, value := range []string{"", "example value"} {
			if !field.valid(value) {
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
			if !field.valid(value) {
				t.Errorf("%s rejected %q", test.name, value)
			}
		}
		for _, value := range []string{"", "UNKNOWN"} {
			if field.valid(value) {
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
			var valid func(string) bool
			for _, field := range mergeMethodsStringFields {
				if field.name == test.name {
					valid = field.valid
				}
			}
			if valid == nil {
				t.Fatalf("missing validator definition for %s", test.name)
			}
			for _, value := range test.allowed {
				if !valid(value) {
					t.Errorf("%s rejected %q", test.name, value)
				}
			}
			for _, value := range []string{"", "UNKNOWN"} {
				if valid(value) {
					t.Errorf("%s accepted %q", test.name, value)
				}
			}
		})
	}
}
