package settings

import "testing"

func TestComparisonRetainsActionableActionsRadioProjection(t *testing.T) {
	radio := executionValidationRadio(t, DefaultActionsPermissionsChoice)
	comparison := Comparison{Managed: 2, ActionsExecution: []ActionsExecutionChange{radio}}
	if !comparison.HasChanges() {
		t.Fatal("fallback match became independently verified no drift")
	}
	if (Comparison{Managed: 2}).HasChanges() {
		t.Fatal("managed count invented a difference")
	}
}
