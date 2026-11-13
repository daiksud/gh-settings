package settings

import "testing"

func TestRulesetAdditionalApprovalOwnership(t *testing.T) {
	for _, value := range []bool{false, true} {
		before := reviewPolicy()
		before.ID = rsID(7)
		ruleOf(t, before, "pull_request").RequireExtraApprovalForUnattributedChanges = paramBool(value)
		current := rsObserved(t, RulesetState{ID: 7, Policy: before})
		desired := cloneRuleset(before)
		if changes, err := ReconcileRulesets(current, rsDesired(t, desired)); err != nil || len(changes) != 0 {
			t.Fatal("matching value caused difference", changes, err)
		}
		ruleOf(t, desired, "pull_request").RequireExtraApprovalForUnattributedChanges = paramBool(!value)
		changes, err := ReconcileRulesets(current, rsDesired(t, desired))
		if err != nil || len(changes) != 1 {
			t.Fatal(changes, err)
		}
		actual, present := RulesetAdditionalApproval(changes[0].After())
		if !present || actual == nil || *actual != !value {
			t.Fatal("lost declared value", actual)
		}
		*actual = value
		again, _ := RulesetAdditionalApproval(changes[0].After())
		if *again != !value {
			t.Fatal("value alias")
		}
		ruleOf(t, desired, "pull_request").RequireExtraApprovalForUnattributedChanges = nil
		changes, err = ReconcileRulesets(current, rsDesired(t, desired))
		if err != nil || len(changes) != map[bool]int{false: 1, true: 0}[value] {
			t.Fatal("omission did not own true default", changes, err)
		}
		desired.Rules = &[]RulesetRule{{Type: "deletion"}}
		changes, err = ReconcileRulesets(current, rsDesired(t, desired))
		if err != nil || len(changes) != 1 {
			t.Fatal("owned PR removal rejected", changes, err)
		}
		empty, err := NewRulesetDeclarations(nil)
		if err != nil {
			t.Fatal(err)
		}
		changes, err = ReconcileRulesetsWithPrune(current, &empty, true)
		if err != nil || len(changes) != 1 || changes[0].Operation() != "delete" || !changes[0].MatchesDeletionPolicy(before) {
			t.Fatal("owned Ruleset prune rejected", changes, err)
		}
	}
}

func TestRulesetAdditionalApprovalObservationRequiresValue(t *testing.T) {
	policy := reviewPolicy()
	ruleOf(t, policy, "pull_request").RequireExtraApprovalForUnattributedChanges = nil
	if _, err := NewRulesetObservation([]RulesetState{{ID: 7, Policy: policy}}, 0); err == nil {
		t.Fatal("defaulted missing observation")
	}
}
