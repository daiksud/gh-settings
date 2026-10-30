package settings

import (
	"reflect"
	"strings"
	"testing"

	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

func executionComparisonState(enabled bool) ActionsPolicyState {
	return ActionsPolicyState{Enabled: enabled, AllowedActions: "all", AllowedActionsNormalization: ObservationNormalization{Path: ActionsPermissionsPath, RawValue: "UNMANAGED_POLICY_CANARY", Fallback: "allow_all_actions_and_reusable_workflows"}}
}

func TestActionsExecutionComparisonRetainsOneOwnedRadioAndSelectionEvidence(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		for _, test := range []struct {
			choice  uimodel.ActionsPermissionChoice
			enabled bool
			policy  *string
			managed int
		}{
			{uimodel.DisableActions, false, nil, 1},
			{uimodel.AllowAllActionsAndReusableWorkflows, true, actionsTestPointer("all"), 2},
			{uimodel.AllowOwnerActionsAndReusableWorkflows, true, actionsTestPointer("local_only"), 2},
			{uimodel.AllowOwnerAndSelectNonOwnerActionsAndReusableWorkflows, true, actionsTestPointer("selected"), 5},
		} {
			state := executionComparisonState(enabled)
			d := ActionsPolicyDeclaration{Enabled: &test.enabled, AllowedActions: test.policy}
			if test.choice == uimodel.AllowOwnerAndSelectNonOwnerActionsAndReusableWorkflows {
				d.SelectedActions = SelectedActionsDeclaration{GitHubOwnedAllowed: actionsTestPointer(false), VerifiedAllowed: actionsTestPointer(false), PatternsAllowed: actionsTestPointer([]string{})}
			}
			result, err := CompareActionsPolicy(state, d)
			if err != nil || result.Managed != test.managed {
				t.Fatal(result, err)
			}
			if !enabled && test.choice == uimodel.DisableActions {
				if len(result.Execution) != 0 || len(result.Changes) != 0 {
					t.Fatal("known disabled match invented policy write", result)
				}
				continue
			}
			if len(result.Execution) != 1 || result.Execution[0].After() != test.choice || result.Execution[0].ProjectionVisible() != enabled || result.Execution[0].Normalization() != state.AllowedActionsNormalization {
				t.Fatal("radio decision lost or duplicated", result)
			}
			wantNative := 0
			if test.choice == uimodel.AllowOwnerAndSelectNonOwnerActionsAndReusableWorkflows {
				wantNative = 3
			}
			if len(result.Changes) != wantNative {
				t.Fatal("native components duplicated", result)
			}
			for _, change := range result.Changes {
				before, _ := change.Values()
				if !strings.HasPrefix(change.Field(), "actions.selected_actions.") || before != nil {
					t.Fatal("selection before invented", change)
				}
			}
			if err := ValidateActionsPolicyChanges(result.Changes, result.Execution...); err != nil {
				t.Fatal("radio weakened joint validation", err)
			}
		}
	}
}

func TestActionsExecutionComparisonOmissionAndCanonicalStateStayIndependent(t *testing.T) {
	state := executionComparisonState(true)
	for _, d := range []ActionsPolicyDeclaration{{}, {SHAPinningRequired: actionsTestPointer(true)}, {Enabled: actionsTestPointer(true)}} {
		result, err := CompareActionsPolicy(state, d)
		if err != nil || len(result.Execution) != 0 {
			t.Fatal("omitted policy replayed", result, err)
		}
		for _, change := range result.Changes {
			if change.Normalization() != (ObservationNormalization{}) || change.Field() != "actions.sha_pinning_required" {
				t.Fatal(change)
			}
		}
	}
	canonical := ActionsPolicyState{Enabled: false, AllowedActions: "all"}
	result, err := CompareActionsPolicy(canonical, ActionsPolicyDeclaration{Enabled: actionsTestPointer(true), AllowedActions: actionsTestPointer("local_only")})
	if err != nil || result.Managed != 2 || len(result.Execution) != 0 || len(result.Changes) != 2 || result.Changes[0].Field() != "actions.enabled" || result.Changes[1].Field() != "actions.allowed_actions" {
		t.Fatal("canonical behavior changed", result, err)
	}
}

func TestActionsExecutionComparisonRejectsIncompleteNormalizedRadioWithoutForecast(t *testing.T) {
	state := executionComparisonState(true)
	for _, d := range []ActionsPolicyDeclaration{
		{AllowedActions: actionsTestPointer("all")},
		{Enabled: actionsTestPointer(false), AllowedActions: actionsTestPointer("all")},
		{Enabled: actionsTestPointer(true), AllowedActions: actionsTestPointer("selected")},
		{SelectedActions: SelectedActionsDeclaration{VerifiedAllowed: actionsTestPointer(false)}},
	} {
		result, err := CompareActionsPolicy(state, d)
		if err == nil || !reflect.DeepEqual(result, ActionsPolicyComparison{}) || strings.Contains(err.Error(), "UNMANAGED_POLICY_CANARY") {
			t.Fatal("incomplete radio inferred ownership or leaked", result, err)
		}
	}
	state.AllowedActionsNormalization.RawValue = "all"
	if result, err := CompareActionsPolicy(state, ActionsPolicyDeclaration{Enabled: actionsTestPointer(false)}); err == nil || !reflect.DeepEqual(result, ActionsPolicyComparison{}) {
		t.Fatal("forged provenance retained partial result", result, err)
	}
}
