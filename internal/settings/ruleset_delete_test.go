package settings

import (
	"testing"
)

func TestRulesetDeletionSelectorsAndReconciliation(t *testing.T) {
	yes, no := true, false
	id := int64(7)
	tomb := RulesetDeclaration{ID: &id, Name: "basic", Delete: &yes}
	for _, d := range []RulesetDeclaration{
		{Name: "basic", Delete: &yes}, {ID: copyRuleValue(new(int64)), Name: "basic", Delete: &yes}, {ID: &id, Delete: &yes}, {ID: &id, Name: "basic", Delete: &no},
		{ID: &id, Name: "basic", Delete: &yes, Target: copyRuleValue(new(string))},
		{ID: &id, Name: "basic", Delete: &yes, Enforcement: copyRuleValue(new(string))},
		{ID: &id, Name: "basic", Delete: &yes, Conditions: &RulesetConditions{}},
		{ID: &id, Name: "basic", Delete: &yes, BypassActors: &[]BypassActor{}},
		{ID: &id, Name: "basic", Delete: &yes, Rules: &[]RulesetRule{}},
	} {
		if _, err := NewRulesetDeclarations([]RulesetDeclaration{d}); err == nil {
			t.Errorf("invalid tombstone accepted: %+v", d)
		}
	}
	desired, err := NewRulesetDeclarations([]RulesetDeclaration{tomb})
	if err != nil {
		t.Fatal(err)
	}
	policy := basicPolicy("basic")
	policy.ID = &id
	observed, err := NewRulesetObservationWithUnmanagedIDs([]RulesetState{{ID: id, Policy: policy}}, 1, []int64{9, 10})
	if err != nil {
		t.Fatal(err)
	}
	changes, err := ReconcileRulesets(observed, &desired)
	if err != nil || len(changes) != 1 || changes[0].Operation() != "delete" || changes[0].ID() != id || changes[0].After() != nil || changes[0].Name() != "basic" {
		t.Fatalf("changes=%v err=%v", changes, err)
	}
	if !changes[0].MatchesDeletionPolicy(policy) {
		t.Fatal("complete identical policy did not match")
	}
	policy.Name = "changed"
	if changes[0].MatchesDeletionPolicy(policy) {
		t.Fatal("drifted policy matched")
	}
	for _, selected := range []int64{9, 10} {
		d, _ := NewRulesetDeclarations([]RulesetDeclaration{{ID: &selected, Name: "basic", Delete: &yes}})
		if _, err := ReconcileRulesets(observed, &d); err == nil {
			t.Fatal("excluded ID treated as absent")
		}
	}
	absent, _ := NewRulesetObservation(nil, 0)
	if changes, err := ReconcileRulesets(absent, &desired); err != nil || len(changes) != 0 {
		t.Fatal("repeat deletion is not no-op", changes, err)
	}

	otherID := int64(8)
	replacement := basicPolicy("basic")
	replacement.ID = &otherID
	replacementObserved, err := NewRulesetObservation([]RulesetState{{ID: otherID, Policy: replacement}}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if changes, err := ReconcileRulesets(replacementObserved, &desired); err != nil || len(changes) != 0 {
		t.Fatal("same-name replacement selected as fallback", changes, err)
	}
	normalized := basicPolicy("basic")
	normalized.ID = &id
	disabled := "disabled"
	normalized.Enforcement = &disabled
	normalizedObservation, err := NewRulesetObservation([]RulesetState{{ID: id, Policy: normalized, EnforcementNormalization: ObservationNormalization{Path: RulesetEnforcementPath, RawValue: "FUTURE_ENFORCEMENT", Fallback: "disabled"}}}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ReconcileRulesets(normalizedObservation, &desired); err == nil {
		t.Fatal("normalized unknown protection authorized deletion")
	}
	wrong, _ := NewRulesetDeclarations([]RulesetDeclaration{{ID: &id, Name: "wrong", Delete: &yes}})
	if _, err := ReconcileRulesets(observed, &wrong); err == nil {
		t.Fatal("name mismatch accepted")
	}
	*tomb.Delete = false
	if !*desired.Values()[0].Delete {
		t.Fatal("tombstone aliases caller")
	}
}
