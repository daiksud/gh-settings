package settings

import (
	"reflect"
	"testing"
)

func TestNormalizedRulesetEnforcementReconciliationOwnsOnlySelectedIDs(t *testing.T) {
	for _, raw := range []string{"evaluate", "FUTURE_MODE", "", " \t"} {
		for _, status := range []string{"active", "disabled"} {
			first := normalizedEnforcementState(7, "z", raw)
			unlisted := normalizedEnforcementState(8, "unmanaged", "OTHER_MODE")
			second := normalizedEnforcementState(9, "a", "THIRD_MODE")
			observed := rsObserved(t, first, unlisted, second)
			wanted := cloneRuleset(first.Policy)
			wanted.ID = rsID(7)
			wanted.Enforcement = rsText(status)
			otherWanted := cloneRuleset(second.Policy)
			otherWanted.ID = rsID(9)
			desired := rsDesired(t, wanted, otherWanted)
			changes, err := ReconcileRulesets(observed, desired)
			if err != nil || len(changes) != 2 || changes[0].ID() != 9 || changes[1].ID() != 7 {
				t.Fatal(changes, err)
			}
			for i, state := range []RulesetState{second, first} {
				if changes[i].Operation() != "update" || changes[i].EnforcementNormalization() != state.EnforcementNormalization || changes[i].Display().EnforcementNormalization() != state.EnforcementNormalization {
					t.Fatal("selected evidence lost", changes[i])
				}
			}
			if !reflect.DeepEqual(changes[1].After(), &desired.Values()[1]) {
				t.Fatal("whole selected payload changed", changes[1].After(), desired.Values())
			}
			*changes[1].After().Enforcement = "evaluate"
			if *changes[1].After().Enforcement != status {
				t.Fatal("returned payload alias")
			}
			canonical := RulesetState{ID: 10, Policy: basicPolicy("canonical")}
			canonicalDesired := cloneRuleset(canonical.Policy)
			canonicalDesired.ID = rsID(10)
			withUnlisted := rsObserved(t, canonical, unlisted)
			if got, err := ReconcileRulesets(withUnlisted, rsDesired(t, canonicalDesired)); err != nil || len(got) != 0 {
				t.Fatal("unlisted provenance forced update", got, err)
			}
			if got, err := ReconcileRulesets(observed, nil); err != nil || len(got) != 0 {
				t.Fatal("omission managed objects", got, err)
			}
			if got, err := ReconcileRulesets(observed, rsDesired(t)); err != nil || len(got) != 0 {
				t.Fatal("empty collection managed objects", got, err)
			}
		}
	}
}

func TestNormalizedRulesetEnforcementReconciliationRetainsIdentityAndDismissalRules(t *testing.T) {
	state := normalizedEnforcementState(7, "first", "evaluate")
	observed := rsObserved(t, state, normalizedEnforcementState(8, "collision", "OTHER_MODE"))
	for _, after := range []RulesetDeclaration{
		func() RulesetDeclaration { d := cloneRuleset(state.Policy); d.ID = rsID(99); return d }(),
		func() RulesetDeclaration {
			d := cloneRuleset(state.Policy)
			d.ID = rsID(7)
			d.Name = "collision"
			return d
		}(),
		cloneRuleset(state.Policy),
	} {
		if got, err := ReconcileRulesets(observed, rsDesired(t, after)); err == nil || len(got) != 0 {
			t.Fatal("identity guard bypassed", got, err)
		}
	}
	state.Policy = reviewPolicy()
	state.Policy.Enforcement = rsText("disabled")
	observed = rsObserved(t, state)
	exported, err := observed.ExportDeclarations()
	if err != nil {
		t.Fatal(err)
	}
	changes, err := ReconcileRulesets(observed, &exported)
	if err != nil || len(changes) != 1 || ruleOf(t, *changes[0].After(), "pull_request").DismissalRestriction != nil || ruleOf(t, *changes[0].Display().Before(), "pull_request").DismissalRestriction != nil {
		t.Fatal("omitted write-only setting changed", changes, err)
	}
}
