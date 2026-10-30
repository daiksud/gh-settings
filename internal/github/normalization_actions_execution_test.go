package github

import (
	"reflect"
	"testing"

	"github.com/daiksud/gh-settings/internal/settings"
)

func TestActionsExecutionProjectionPreservesTypedBooleansAndRawPolicy(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		for _, sha := range []bool{false, true} {
			for _, raw := range []string{"all", "local_only", "selected", "FUTURE_POLICY", "", " \t"} {
				inputEnabled, inputSHA, inputRaw := enabled, sha, raw
				state, err := prepareActionsExecutionObservation(&inputEnabled, &inputRaw, &inputSHA)
				wantPolicy := raw
				wantRecord := settings.ObservationNormalization{}
				if raw != "all" && raw != "local_only" && raw != "selected" {
					wantPolicy = "all"
					wantRecord = settings.ObservationNormalization{Path: "repository.settings.actions.general.actions_permissions", RawValue: raw, Fallback: "allow_all_actions_and_reusable_workflows"}
				}
				if err != nil || state.Enabled != enabled || state.SHAPinningRequired != sha || state.AllowedActions != wantPolicy || state.Selected != nil || state.AllowedActionsNormalization != wantRecord {
					t.Fatal(state, wantPolicy, wantRecord, err)
				}
				inputEnabled = !enabled
				inputSHA = !sha
				inputRaw = "mutated caller"
				got, err := state.RawAllowedActions()
				if err != nil || got != raw || state.Enabled != enabled || state.SHAPinningRequired != sha {
					t.Fatal("aliased or lost raw evidence", state, got, err)
				}
			}
		}
	}
}

func TestActionsExecutionProjectionRequiresCompletePointers(t *testing.T) {
	for _, test := range []struct {
		enabled *bool
		policy  *string
		sha     *bool
	}{
		{nil, nil, nil}, {nil, actionsUIPtr("FUTURE_POLICY"), actionsUIPtr(false)},
		{actionsUIPtr(false), nil, actionsUIPtr(false)}, {actionsUIPtr(false), actionsUIPtr("FUTURE_POLICY"), nil},
	} {
		state, err := prepareActionsExecutionObservation(test.enabled, test.policy, test.sha)
		if err == nil || !reflect.DeepEqual(state, settings.ActionsPolicyState{}) {
			t.Fatal("incomplete policy defaulted", state, err)
		}
	}
}
