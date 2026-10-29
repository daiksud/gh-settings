package settings

import (
	"strings"
	"testing"
)

func TestActionsExecutionUnknownPolicyCannotAuthorizeEnableOrSelection(t *testing.T) {
	record := ObservationNormalization{Path: ActionsPermissionsPath, RawValue: "UNMANAGED_POLICY_CANARY", Fallback: "allow_all_actions_and_reusable_workflows"}
	unknownDisabled := ActionsPolicyState{AllowedActions: "all", AllowedActionsNormalization: record}
	unknownEnabled := unknownDisabled
	unknownEnabled.Enabled = true
	canonical := ActionsPolicyState{AllowedActions: "all"}
	selected := SelectedActionsDeclaration{GitHubOwnedAllowed: actionsTestPointer(false), VerifiedAllowed: actionsTestPointer(false), PatternsAllowed: actionsTestPointer([]string{})}
	for _, test := range []struct {
		name    string
		state   ActionsPolicyState
		desired ActionsPolicyDeclaration
		valid   bool
	}{
		{"unknown enable only", unknownDisabled, ActionsPolicyDeclaration{Enabled: actionsTestPointer(true)}, false},
		{"unknown enable all", unknownDisabled, ActionsPolicyDeclaration{Enabled: actionsTestPointer(true), AllowedActions: actionsTestPointer("all")}, true},
		{"unknown enable local", unknownDisabled, ActionsPolicyDeclaration{Enabled: actionsTestPointer(true), AllowedActions: actionsTestPointer("local_only")}, true},
		{"unknown enable complete selected", unknownDisabled, ActionsPolicyDeclaration{Enabled: actionsTestPointer(true), AllowedActions: actionsTestPointer("selected"), SelectedActions: selected}, true},
		{"unknown enable incomplete selected", unknownDisabled, ActionsPolicyDeclaration{Enabled: actionsTestPointer(true), AllowedActions: actionsTestPointer("selected")}, false},
		{"canonical enable only", canonical, ActionsPolicyDeclaration{Enabled: actionsTestPointer(true)}, true},
		{"already enabled only", unknownEnabled, ActionsPolicyDeclaration{Enabled: actionsTestPointer(true)}, true},
		{"unknown disable", unknownEnabled, ActionsPolicyDeclaration{Enabled: actionsTestPointer(false)}, true},
		{"unknown already disabled", unknownDisabled, ActionsPolicyDeclaration{Enabled: actionsTestPointer(false)}, true},
		{"unrelated SHA", unknownDisabled, ActionsPolicyDeclaration{SHAPinningRequired: actionsTestPointer(false)}, true},
		{"unknown companion only", unknownEnabled, ActionsPolicyDeclaration{SelectedActions: SelectedActionsDeclaration{VerifiedAllowed: actionsTestPointer(false)}}, false},
		{"unknown complete companions without policy", unknownEnabled, ActionsPolicyDeclaration{SelectedActions: selected}, false},
		{"explicit selected complete", unknownEnabled, ActionsPolicyDeclaration{AllowedActions: actionsTestPointer("selected"), SelectedActions: selected}, true},
		{"known selected companions", ActionsPolicyState{AllowedActions: "selected", Selected: &SelectedActionsState{PatternsAllowed: []string{}}}, ActionsPolicyDeclaration{SelectedActions: SelectedActionsDeclaration{VerifiedAllowed: actionsTestPointer(false)}}, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := PrepareActions(test.state, test.desired)
			if (err == nil) != test.valid {
				t.Fatal(err, test.valid)
			}
			if err != nil && strings.Contains(err.Error(), "UNMANAGED_POLICY_CANARY") {
				t.Fatal("unmanaged policy disclosed", err)
			}
		})
	}
}
