package settings

import (
	"reflect"
	"testing"

	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

func TestActionsExecutionChangeOwnsRadioWithoutInventingVisibleFallback(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		for _, raw := range []string{"FUTURE_POLICY", "", " \t"} {
			for _, desired := range []uimodel.ActionsPermissionChoice{"allow_all_actions_and_reusable_workflows", "disable_actions", "allow_owner_actions_and_reusable_workflows", "allow_owner_and_select_non_owner_actions_and_reusable_workflows"} {
				record := ObservationNormalization{Path: ActionsPermissionsPath, RawValue: raw, Fallback: "allow_all_actions_and_reusable_workflows"}
				state := ActionsPolicyState{Enabled: enabled, AllowedActions: "all", SHAPinningRequired: true, AllowedActionsNormalization: record}
				change, err := NewActionsExecutionChange(state, desired)
				if !enabled && desired == "disable_actions" {
					if err == nil || change.After() != "" || change.Normalization() != (ObservationNormalization{}) || len(change.NativeFields()) != 0 {
						t.Fatal("known disabled match became write", change, err)
					}
					continue
				}
				wantBefore := uimodel.ActionsPermissionChoice("disable_actions")
				if enabled {
					wantBefore = "allow_all_actions_and_reusable_workflows"
				}
				wantFields := []string{"actions.enabled"}
				if desired != "disable_actions" {
					wantFields = append(wantFields, "actions.allowed_actions")
				}
				if err != nil || change.Before() != wantBefore || change.After() != desired || change.ProjectionVisible() != enabled || change.Normalization() != record || !reflect.DeepEqual(change.NativeFields(), wantFields) {
					t.Fatal(change, wantBefore, wantFields, err)
				}
				fields := change.NativeFields()
				fields[0] = "mutated caller"
				state.Enabled = !enabled
				state.AllowedActionsNormalization.RawValue = "mutated caller"
				copy := change.Normalization()
				copy.RawValue = "mutated copy"
				if change.Before() != wantBefore || change.Normalization() != record || !reflect.DeepEqual(change.NativeFields(), wantFields) {
					t.Fatal("mutable decision evidence", change)
				}
			}
		}
	}
}

func TestActionsExecutionChangeRejectsUnnormalizedAndForgedStates(t *testing.T) {
	valid := ObservationNormalization{Path: ActionsPermissionsPath, RawValue: "FUTURE_POLICY", Fallback: "allow_all_actions_and_reusable_workflows"}
	for _, state := range []ActionsPolicyState{
		{}, {Enabled: true, AllowedActions: "all"},
		{AllowedActions: "local_only", AllowedActionsNormalization: valid},
		{AllowedActions: "all", Selected: &SelectedActionsState{PatternsAllowed: []string{}}, AllowedActionsNormalization: valid},
		{AllowedActions: "all", AllowedActionsNormalization: ObservationNormalization{Path: ActionsPermissionsPath, RawValue: "all", Fallback: "allow_all_actions_and_reusable_workflows"}},
		{AllowedActions: "all", AllowedActionsNormalization: ObservationNormalization{Path: "wrong", RawValue: "FUTURE_POLICY", Fallback: "allow_all_actions_and_reusable_workflows"}},
	} {
		change, err := NewActionsExecutionChange(state, "allow_all_actions_and_reusable_workflows")
		if err == nil || change.After() != "" || len(change.NativeFields()) != 0 || change.Normalization() != (ObservationNormalization{}) {
			t.Fatal(state, change, err)
		}
	}
	state := ActionsPolicyState{Enabled: true, AllowedActions: "all", AllowedActionsNormalization: valid}
	for _, desired := range []uimodel.ActionsPermissionChoice{"", "all", "selected", "future", "Disable_Actions"} {
		if change, err := NewActionsExecutionChange(state, desired); err == nil || change.After() != "" || len(change.NativeFields()) != 0 {
			t.Fatal(desired, change, err)
		}
	}
}
