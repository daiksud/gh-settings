package settings

import (
	"testing"

	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

func TestActionsExecutionMappingKeepsRadioAndPolicyOwnershipSeparate(t *testing.T) {
	if ActionsPermissionsPath != "repository.settings.actions.general.actions_permissions" || DefaultActionsPermissionsChoice != "allow_all_actions_and_reusable_workflows" {
		t.Fatal(ActionsPermissionsPath, DefaultActionsPermissionsChoice)
	}
	for _, test := range []struct {
		choice  uimodel.ActionsPermissionChoice
		enabled bool
		allowed string
	}{
		{"allow_all_actions_and_reusable_workflows", true, "all"}, {"disable_actions", false, ""},
		{"allow_owner_actions_and_reusable_workflows", true, "local_only"}, {"allow_owner_and_select_non_owner_actions_and_reusable_workflows", true, "selected"},
	} {
		enabled, allowed, valid := ActionsPermissionsValueForChoice(test.choice)
		if !valid || enabled != test.enabled || (allowed != nil) != test.enabled || allowed != nil && *allowed != test.allowed {
			t.Fatal(test, enabled, allowed, valid)
		}
		if allowed != nil {
			*allowed = "mutated caller"
			_, fresh, valid := ActionsPermissionsValueForChoice(test.choice)
			if !valid || fresh == nil || *fresh != test.allowed {
				t.Fatal("shared policy pointer", fresh, valid)
			}
		}
		choice, valid := ActionsPermissionsChoiceForValue(test.enabled, test.allowed)
		if !valid || choice != test.choice {
			t.Fatal(test, choice, valid)
		}
	}
	for _, latent := range []string{"all", "local_only", "selected", "FUTURE_POLICY", "", " \t"} {
		choice, valid := ActionsPermissionsChoiceForValue(false, latent)
		if !valid || choice != "disable_actions" {
			t.Fatal("disabled policy became visible", latent, choice, valid)
		}
	}
	for _, invalid := range []string{"", "FUTURE_POLICY", "ALL", " all", "disable_actions"} {
		if choice, valid := ActionsPermissionsChoiceForValue(true, invalid); valid || choice != "" {
			t.Fatal("enabled unknown accepted", invalid, choice, valid)
		}
		if ValidAllowedActions(invalid) {
			t.Fatal("validator used disabled mapping", invalid)
		}
	}
	for _, invalid := range []uimodel.ActionsPermissionChoice{"", "all", "local_only", "selected", "Disable_Actions", "future"} {
		if enabled, allowed, valid := ActionsPermissionsValueForChoice(invalid); valid || enabled || allowed != nil {
			t.Fatal("invalid choice acquired policy", invalid, enabled, allowed, valid)
		}
	}
	if err := (ActionsPolicyState{Enabled: false, AllowedActions: "FUTURE_POLICY"}).Validate(); err == nil {
		t.Fatal("mapping extraction activated unknown observation")
	}
}
