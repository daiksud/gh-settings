package settings

import (
	"testing"

	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

func TestCreationPolicyMappingIsClosed(t *testing.T) {
	for _, test := range []struct {
		control CreationPolicyControl
		path    string
		field   string
	}{
		{IssuePermissionsControl, "repository.settings.general.features.issue_permissions.creation_allowed_by", "repository.issue_creation_policy"},
		{PullRequestPermissionsControl, "repository.settings.general.features.pull_request_permissions.creation_allowed_by", "repository.pull_request_creation_policy"},
	} {
		choice, ok := test.control.DefaultChoice()
		if !ok || choice != "all_users" || test.control.Path() != test.path || test.control.Field().Path() != test.field {
			t.Fatal("control identity/default changed", test, choice, ok)
		}
	}
	if CreationPolicyControl("future").Field() != (FieldID{}) {
		t.Fatal("unknown control invented native identity")
	}
	for _, test := range []struct {
		choice uimodel.CreationAllowedBy
		value  string
	}{{"all_users", "ALL"}, {"collaborators_only", "COLLABORATORS_ONLY"}} {
		if value, ok := CreationPolicyValueForChoice(test.choice); !ok || value != test.value {
			t.Fatal("UI mapping changed", test, value, ok)
		}
		if choice, ok := CreationPolicyChoiceForValue(test.value); !ok || choice != test.choice {
			t.Fatal("API mapping changed", test, choice, ok)
		}
	}
	if choice, ok := CreationPolicyControl("future").DefaultChoice(); ok || choice != "" || CreationPolicyControl("future").Path() != "" {
		t.Fatal("unknown control invented policy")
	}
	for _, value := range []string{"", "FUTURE", "all_users", " ALL "} {
		if _, ok := CreationPolicyChoiceForValue(value); ok {
			t.Fatal("unknown native value accepted", value)
		}
	}
	for _, choice := range []uimodel.CreationAllowedBy{"", "FUTURE", "ALL"} {
		if _, ok := CreationPolicyValueForChoice(choice); ok {
			t.Fatal("unknown UI choice accepted", choice)
		}
	}
}
