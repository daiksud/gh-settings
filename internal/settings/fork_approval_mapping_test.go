package settings

import (
	"testing"

	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

func TestForkApprovalMappingIsClosed(t *testing.T) {
	for _, test := range []struct {
		choice uimodel.ForkApprovalChoice
		value  string
	}{
		{"require_approval_for_first_time_contributors_who_are_new_to_github", "first_time_contributors_new_to_github"},
		{"require_approval_for_first_time_contributors", "first_time_contributors"},
		{"require_approval_for_all_external_contributors", "all_external_contributors"},
	} {
		if value, valid := ForkApprovalValueForChoice(test.choice); !valid || value != test.value {
			t.Fatal("UI mapping changed", test, value, valid)
		}
		if choice, valid := ForkApprovalChoiceForValue(test.value); !valid || choice != test.choice {
			t.Fatal("native mapping changed", test, choice, valid)
		}
	}
	if DefaultForkApprovalChoice != "require_approval_for_first_time_contributors" || ForkApprovalPath != "repository.settings.actions.general.approval_for_running_fork_pull_request_workflows_from_contributors" {
		t.Fatal("adopted default/path changed")
	}
	for _, value := range []string{"", "FUTURE", "FIRST_TIME_CONTRIBUTORS", " first_time_contributors "} {
		if _, valid := ForkApprovalChoiceForValue(value); valid {
			t.Fatal("unadopted native value accepted", value)
		}
	}
	for _, choice := range []uimodel.ForkApprovalChoice{"", "FUTURE", "first_time_contributors"} {
		if _, valid := ForkApprovalValueForChoice(choice); valid {
			t.Fatal("unadopted UI choice accepted", choice)
		}
	}
}
