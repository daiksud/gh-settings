package settings

import (
	"strings"
	"testing"
)

func TestRemovedRulesetTombstonesGivePruneMigration(t *testing.T) {
	id := int64(7)
	for _, value := range []bool{true, false} {
		_, err := NewRulesetDeclarations([]RulesetDeclaration{{ID: &id, Name: "basic", Delete: &value}})
		if err == nil || !strings.Contains(err.Error(), "every Ruleset to retain") || !strings.Contains(err.Error(), "--prune rulesets") {
			t.Fatal(err)
		}
	}
}

func TestRulesetPruneRejectsUnmanagedAndNormalizedPolicy(t *testing.T) {
	id := int64(7)
	policy := basicPolicy("basic")
	policy.ID = &id
	desired, _ := NewRulesetDeclarations(nil)
	current, _ := NewRulesetObservation([]RulesetState{{ID: id, Policy: policy}}, 0)
	changes, err := ReconcileRulesetsWithPrune(current, &desired, true)
	if err != nil || len(changes) != 1 || changes[0].ID() != id || changes[0].Operation() != "delete" || !changes[0].MatchesDeletionPolicy(policy) {
		t.Fatal(changes, err)
	}
	policy.Name = "changed"
	if changes[0].MatchesDeletionPolicy(policy) {
		t.Fatal("drift accepted")
	}
	for _, args := range []struct {
		push int
		ids  []int64
	}{{1, nil}, {0, []int64{9}}} {
		unowned, _ := NewRulesetObservationWithUnmanagedIDs(nil, args.push, args.ids)
		if _, err := ReconcileRulesetsWithPrune(unowned, &desired, true); err == nil {
			t.Fatal("unowned inventory accepted")
		}
	}
	policy = basicPolicy("basic")
	policy.ID = &id
	disabled := "disabled"
	policy.Enforcement = &disabled
	current, err = NewRulesetObservation([]RulesetState{{ID: id, Policy: policy, EnforcementNormalization: ObservationNormalization{Path: RulesetEnforcementPath, RawValue: "FUTURE", Fallback: "disabled"}}}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ReconcileRulesetsWithPrune(current, &desired, true); err == nil {
		t.Fatal("normalized protection authorized deletion")
	}
}
