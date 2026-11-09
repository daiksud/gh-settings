package settings

import "testing"

func TestUnownedQualityReconciliationAndIsolation(t *testing.T) {
	before := reviewPolicy()
	before.Rules = &[]RulesetRule{{Type: "deletion"}, {Type: "non_fast_forward"}}
	before.ID = rsID(7)
	before, err := WithObservedRulesetCodeQuality(before, "FUTURE_SEVERITY")
	if err != nil {
		t.Fatal(err)
	}
	desired := cloneRuleset(before)
	desired.unownedCodeQualitySeverity = nil
	observed := rsObserved(t, RulesetState{ID: 7, Policy: before})
	if changes, err := ReconcileRulesets(observed, rsDesired(t, desired)); err != nil || len(changes) != 0 {
		t.Fatal("unowned value caused a managed difference", changes, err)
	}
	// Removing a supported rule retains the separate unowned rule.
	desired.Rules = &[]RulesetRule{{Type: "deletion"}}
	changes, err := ReconcileRulesets(observed, rsDesired(t, desired))
	if err != nil || len(changes) != 1 {
		t.Fatal(changes, err)
	}
	value := RulesetUnownedCodeQuality(changes[0].After())
	if value == nil || *value != "FUTURE_SEVERITY" {
		t.Fatal("unowned rule dropped", value)
	}
	*value = "corrupted"
	if *RulesetUnownedCodeQuality(changes[0].After()) != "FUTURE_SEVERITY" {
		t.Fatal("aliased acquired value")
	}
	for _, after := range []*RulesetDeclaration{nil, &desired} {
		if _, err := NewRulesetChange("update", 7, &before, after); err == nil {
			t.Fatal("allowed removal of unowned rule")
		}
	}
	if _, err := NewRulesetChange("delete", 7, &before, nil); err == nil {
		t.Fatal("allowed deletion of unowned rule's Ruleset")
	}
	if _, err := NewRulesetChange("create", 0, nil, &before); err == nil {
		t.Fatal("allowed creation with acquired unowned rule")
	}
	// Fresh deletion checks must detect a newly added unowned rule as drift.
	plain := cloneRuleset(before)
	plain.unownedCodeQualitySeverity = nil
	deletion, err := NewRulesetChange("delete", 7, &plain, nil)
	if err != nil || deletion.MatchesDeletionPolicy(before) {
		t.Fatal("unowned deletion drift ignored", err)
	}
}
