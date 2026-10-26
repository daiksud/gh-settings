package settings

import (
	"strings"
	"testing"
)

func TestUnknownDismissalBlocksSelectedWholeObjectOnly(t *testing.T) {
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
	// Ordinary YAML/declaration defaults remain whole-object policy, not a
	// hidden instruction to retain unknown nested state on update.
	desired := rsDesired(t, values[0])
	if restriction := ruleOf(t, desired.Values()[0], "pull_request").DismissalRestriction; restriction == nil || *restriction.Enabled {
		t.Fatal("declaration default changed")
	}
	changes, err := ReconcileRulesets(observed, desired)
	if err == nil || !strings.Contains(err.Error(), "id 7") || !strings.Contains(err.Error(), "complete API observation") || len(changes) != 0 {
		t.Fatalf("unknown was presented as a disabled/default-derived difference: %v %v", changes, err)
	}
	// Even selecting removal of the whole PR rule cannot bypass the guard.
	remove := basicPolicy("default")
	remove.ID = rsID(7)
	if _, err := ReconcileRulesets(observed, rsDesired(t, remove)); err == nil {
		t.Fatal("whole-policy update bypassed unknown protection")
	}
	// Inventory of an unselected object does not manage or default its fields.
	other := basicPolicy("new")
	changes, err = ReconcileRulesets(observed, rsDesired(t, other))
	if err != nil || len(changes) != 1 || changes[0].Operation() != "create" {
		t.Fatalf("unselected optional absence blocked independent object: %v %v", changes, err)
	}
	other.Name = "aaa-new"
	changes, err = ReconcileRulesets(observed, rsDesired(t, other, exported.Values()[0]))
	if err == nil || len(changes) != 0 {
		t.Fatalf("incomplete collection preparation leaked an earlier create: %v %v", changes, err)
	}
	*values[0].ID = 99
	if *exported.Values()[0].ID != 7 {
		t.Fatal("exported identity pointer escaped")
	}
}
