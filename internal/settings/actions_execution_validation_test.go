package settings

import (
	"reflect"
	"testing"

	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

func executionValidationRadio(t *testing.T, desired uimodel.ActionsPermissionChoice) ActionsExecutionChange {
	t.Helper()
	c, err := NewActionsExecutionChange(ActionsPolicyState{Enabled: true, AllowedActions: "all", AllowedActionsNormalization: ObservationNormalization{Path: ActionsPermissionsPath, RawValue: "FUTURE_POLICY", Fallback: "allow_all_actions_and_reusable_workflows"}}, desired)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func executionValidationSelected(t *testing.T, inactive bool) []ActionsPolicyChange {
	t.Helper()
	var changes []ActionsPolicyChange
	for _, test := range []struct {
		field string
		after any
	}{{"actions.selected_actions.github_owned_allowed", false}, {"actions.selected_actions.verified_allowed", false}, {"actions.selected_actions.patterns_allowed", []string{}}} {
		var before any
		if !inactive {
			if test.field == "actions.selected_actions.patterns_allowed" {
				before = []string{"owner/action@v1"}
			} else {
				before = true
			}
		}
		c, err := NewActionsPolicyChange(test.field, before, test.after)
		if err != nil {
			t.Fatal(err)
		}
		changes = append(changes, c)
	}
	return changes
}

func TestActionsExecutionValidationSharesCompleteSelectedTransitionRule(t *testing.T) {
	radio := executionValidationRadio(t, uimodel.AllowOwnerAndSelectNonOwnerActionsAndReusableWorkflows)
	selected := executionValidationSelected(t, true)
	if err := ValidateActionsPolicyChanges(selected, radio); err != nil {
		t.Fatal("complete radio transition rejected", err)
	}
	for _, changes := range [][]ActionsPolicyChange{nil, selected[:1], selected[:2], executionValidationSelected(t, false)} {
		if err := ValidateActionsPolicyChanges(changes, radio); err == nil {
			t.Fatal("incomplete or invented active selection accepted", changes)
		}
	}
	policy, err := NewActionsPolicyChange("actions.allowed_actions", "all", "selected")
	if err != nil {
		t.Fatal(err)
	}
	ordinary := append([]ActionsPolicyChange{policy}, selected...)
	if err := ValidateActionsPolicyChanges(ordinary); err != nil {
		t.Fatal("ordinary transition changed", err)
	}
	if err := ValidateActionsPolicyChanges(selected); err == nil {
		t.Fatal("ordinary unavailable companions lost prerequisite")
	}
	sha, err := NewActionsPolicyChange("actions.sha_pinning_required", false, true)
	if err != nil {
		t.Fatal(err)
	}
	backing := make([]ActionsPolicyChange, 4)
	copy(backing, selected)
	backing[3] = sha
	if err := ValidateActionsPolicyChanges(backing[:3], radio); err != nil || !reflect.DeepEqual(backing[3], sha) {
		t.Fatal("derived evidence aliases caller", backing, err)
	}
}

func TestActionsExecutionValidationRejectsOverlapAndNonSelectedCompanions(t *testing.T) {
	enabled, err := NewActionsPolicyChange("actions.enabled", true, false)
	if err != nil {
		t.Fatal(err)
	}
	policy, err := NewActionsPolicyChange("actions.allowed_actions", "all", "local_only")
	if err != nil {
		t.Fatal(err)
	}
	sha, err := NewActionsPolicyChange("actions.sha_pinning_required", false, true)
	if err != nil {
		t.Fatal(err)
	}
	for _, desired := range []uimodel.ActionsPermissionChoice{uimodel.AllowAllActionsAndReusableWorkflows, uimodel.DisableActions, uimodel.AllowOwnerActionsAndReusableWorkflows, uimodel.AllowOwnerAndSelectNonOwnerActionsAndReusableWorkflows} {
		radio := executionValidationRadio(t, desired)
		for _, overlap := range []ActionsPolicyChange{enabled, policy} {
			if err := ValidateActionsPolicyChanges([]ActionsPolicyChange{overlap}, radio); err == nil {
				t.Fatal("parallel radio/native evidence accepted", desired, overlap)
			}
		}
		if desired == uimodel.AllowOwnerAndSelectNonOwnerActionsAndReusableWorkflows {
			continue
		}
		if err := ValidateActionsPolicyChanges([]ActionsPolicyChange{sha}, radio); err != nil {
			t.Fatal("unrelated native change rejected", err)
		}
		for _, active := range []bool{false, true} {
			if err := ValidateActionsPolicyChanges(executionValidationSelected(t, active), radio); err == nil {
				t.Fatal("nonselected target accepted companions", desired, active)
			}
		}
	}
}

func TestActionsExecutionValidationRejectsDuplicateZeroAndForgedRadio(t *testing.T) {
	valid := executionValidationRadio(t, uimodel.AllowAllActionsAndReusableWorkflows)
	if err := ValidateActionsPolicyChanges(nil, valid, valid); err == nil {
		t.Fatal("duplicate radio accepted")
	}
	for _, change := range []ActionsExecutionChange{
		{}, {beforeEnabled: true, after: uimodel.AllowAllActionsAndReusableWorkflows},
		{beforeEnabled: false, after: uimodel.DisableActions, normalization: valid.normalization},
		{beforeEnabled: true, after: "future", normalization: valid.normalization},
		{beforeEnabled: true, after: uimodel.AllowAllActionsAndReusableWorkflows, normalization: ObservationNormalization{Path: ActionsPermissionsPath, RawValue: "all", Fallback: "allow_all_actions_and_reusable_workflows"}},
	} {
		if err := ValidateActionsPolicyChanges(nil, change); err == nil {
			t.Fatal("invalid radio accepted", change)
		}
	}
}
