package settings

import "testing"

func TestOptionalDismissalRetainsObservedWireValue(t *testing.T) {
	policy := reviewPolicy()
	ruleOf(t, policy, "pull_request").DismissalRestriction = nil
	observed, err := NewRulesetObservation([]RulesetState{{ID: 7, Policy: policy}}, 0)
	if err != nil {
		t.Fatal(err)
	}
	exported, err := observed.ExportDeclarations()
	if err != nil {
		t.Fatal(err)
	}
	values := exported.Values()
	if *values[0].ID != 7 || ruleOf(t, values[0], "pull_request").DismissalRestriction != nil {
		t.Fatal("export changed observation", values)
	}
	// The documented optional-field exception preserves absence, not false.
	desired := rsDesired(t, values[0])
	if restriction := ruleOf(t, desired.Values()[0], "pull_request").DismissalRestriction; restriction != nil {
		t.Fatal("declaration invented a restriction")
	}
	changes, err := ReconcileRulesets(observed, desired)
	if err != nil || len(changes) != 0 {
		t.Fatalf("unchanged observable policy differs: %v %v", changes, err)
	}
	changed := values[0]
	changed.Enforcement = rsText("disabled")
	changes, err = ReconcileRulesets(observed, rsDesired(t, changed))
	if err != nil || len(changes) != 1 || ruleOf(t, *changes[0].After(), "pull_request").DismissalRestriction != nil {
		t.Fatalf("update invented absent protection: %v %v", changes, err)
	}
	known := reviewPolicy()
	knownObserved := rsObserved(t, RulesetState{ID: 7, Policy: known})
	if changes, err := ReconcileRulesets(knownObserved, desired); err != nil || len(changes) != 0 {
		t.Fatalf("omission cleared explicitly observed protection: %v %v", changes, err)
	}
	explicit := values[0]
	ruleOf(t, explicit, "pull_request").DismissalRestriction = &ReviewDismissalRestriction{Enabled: paramBool(false), AllowedActors: &[]ReviewActor{}}
	changes, err = ReconcileRulesets(observed, rsDesired(t, explicit))
	if err != nil || len(changes) != 1 || *ruleOf(t, *changes[0].After(), "pull_request").DismissalRestriction.Enabled {
		t.Fatalf("explicit YAML did not select its requested value: %v %v", changes, err)
	}
	remove := basicPolicy("default")
	remove.ID = rsID(7)
	if _, err := ReconcileRulesets(observed, rsDesired(t, remove)); err != nil {
		t.Fatal("explicit whole-rule removal rejected", err)
	}
	// Inventory of an unselected object does not manage or default its fields.
	other := basicPolicy("new")
	changes, err = ReconcileRulesets(observed, rsDesired(t, other))
	if err != nil || len(changes) != 1 || changes[0].Operation() != "create" {
		t.Fatalf("unselected optional absence blocked independent object: %v %v", changes, err)
	}
	other.Name = "aaa-new"
	changes, err = ReconcileRulesets(observed, rsDesired(t, other, exported.Values()[0]))
	if err != nil || len(changes) != 1 {
		t.Fatalf("unchanged omitted wire value blocked independent create: %v %v", changes, err)
	}
	*values[0].ID = 99
	if *exported.Values()[0].ID != 7 {
		t.Fatal("exported identity pointer escaped")
	}
}
