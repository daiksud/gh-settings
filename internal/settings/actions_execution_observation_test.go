package settings

import (
	"reflect"
	"testing"
)

func TestActionsExecutionStatePreservesRawPolicyWithoutChangingEnabled(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		for _, raw := range []string{"FUTURE_POLICY", "", " \t"} {
			record := ObservationNormalization{Path: ActionsPermissionsPath, RawValue: raw, Fallback: "allow_all_actions_and_reusable_workflows"}
			state := ActionsPolicyState{Enabled: enabled, AllowedActions: "all", SHAPinningRequired: true, AllowedActionsNormalization: record}
			if err := state.Validate(); err != nil {
				t.Fatal(err)
			}
			got, err := state.RawAllowedActions()
			if err != nil || got != raw || state.Enabled != enabled || !state.SHAPinningRequired || state.Selected != nil {
				t.Fatal(state, got, err)
			}
			record.RawValue = "mutated caller"
			copy := state
			copy.AllowedActionsNormalization.RawValue = "mutated copy"
			got, err = state.RawAllowedActions()
			if err != nil || got != raw {
				t.Fatal("aliased evidence", got, err)
			}
			declaration, err := state.Declaration()
			if err != nil || declaration.Enabled == nil || *declaration.Enabled != enabled || declaration.AllowedActions == nil || *declaration.AllowedActions != "all" || declaration.SHAPinningRequired == nil || !*declaration.SHAPinningRequired || !declaration.SelectedActions.Empty() {
				t.Fatal("effective declaration lost", declaration, err)
			}
			*declaration.AllowedActions = "mutated desired"
			*declaration.Enabled = !enabled
			if state.AllowedActions != "all" || state.Enabled != enabled || state.AllowedActionsNormalization.RawValue != raw {
				t.Fatal("desired aliases observation", state)
			}
		}
	}
	for _, policy := range []string{"all", "local_only", "selected"} {
		state := ActionsPolicyState{AllowedActions: policy}
		if raw, err := state.RawAllowedActions(); err != nil || raw != policy {
			t.Fatal(raw, err)
		}
	}
	if raw, err := (ActionsPolicyState{}).RawAllowedActions(); err == nil || raw != "" {
		t.Fatal("zero state acquired", raw, err)
	}
}

func TestActionsExecutionStateRejectsForgedProvenanceAndSelectedEvidence(t *testing.T) {
	valid := ObservationNormalization{Path: ActionsPermissionsPath, RawValue: "FUTURE_POLICY", Fallback: "allow_all_actions_and_reusable_workflows"}
	for _, record := range []ObservationNormalization{
		{Path: "wrong", RawValue: "FUTURE_POLICY", Fallback: "allow_all_actions_and_reusable_workflows"},
		{Path: ActionsPermissionsPath, RawValue: "all", Fallback: "allow_all_actions_and_reusable_workflows"},
		{Path: ActionsPermissionsPath, RawValue: "selected", Fallback: "allow_all_actions_and_reusable_workflows"},
		{Path: ActionsPermissionsPath, RawValue: "FUTURE_POLICY", Fallback: "disable_actions"},
	} {
		state := ActionsPolicyState{AllowedActions: "all", AllowedActionsNormalization: record}
		if err := state.Validate(); err == nil {
			t.Fatal("forged record accepted", record)
		}
		if raw, err := state.RawAllowedActions(); err == nil || raw != "" {
			t.Fatal("invalid raw evidence exposed", raw, err)
		}
	}
	for _, state := range []ActionsPolicyState{
		{AllowedActions: "local_only", AllowedActionsNormalization: valid},
		{AllowedActions: "FUTURE_POLICY", AllowedActionsNormalization: valid},
		{AllowedActions: "all", Selected: &SelectedActionsState{PatternsAllowed: []string{}}, AllowedActionsNormalization: valid},
		{AllowedActions: "selected", Selected: &SelectedActionsState{PatternsAllowed: []string{}}, AllowedActionsNormalization: valid},
	} {
		if err := state.Validate(); err == nil {
			t.Fatal("inconsistent state accepted", state)
		}
		if d, err := state.Declaration(); err == nil || !reflect.DeepEqual(d, ActionsPolicyDeclaration{}) {
			t.Fatal("invalid state became intent", d, err)
		}
	}
}
