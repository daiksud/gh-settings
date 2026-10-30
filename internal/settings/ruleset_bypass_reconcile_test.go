package settings

import (
	"fmt"
	"reflect"
	"testing"
)

func TestRulesetBypassReconciliationRetainsSelectedWholeObjectEvidence(t *testing.T) {
	for _, companions := range []bool{false, true} {
		for _, actors := range []string{"retained", "empty", "omitted"} {
			t.Run(fmt.Sprintf("scalars=%t/actors=%s", companions, actors), func(t *testing.T) {
				state := normalizedBypassState("FUTURE_MODE", companions)
				state.Policy.ID = rsID(state.ID)
				omitDismissalRestriction(&state.Policy)
				unlisted := normalizedBypassState("UNLISTED_MODE", false)
				unlisted.ID, unlisted.Policy.ID, unlisted.Policy.Name = 9, rsID(9), "unlisted"
				observed := rsObserved(t, unlisted, state)
				after := cloneRuleset(state.Policy)
				switch actors {
				case "empty":
					after.BypassActors = &[]BypassActor{}
				case "omitted":
					after.BypassActors = nil
				}
				changes, err := ReconcileRulesets(observed, rsDesired(t, after))
				if err != nil || len(changes) != 1 || changes[0].ID() != 7 || changes[0].Operation() != "update" {
					t.Fatal("selected normalized policy lost its whole-object update", changes, err)
				}
				independent := normalizedBypassState("FUTURE_MODE", companions)
				want := RulesetNormalizations{Enforcement: independent.EnforcementNormalization, QualitySeverity: independent.QualitySeverityNormalization, RequiredApprovals: independent.RequiredApprovalsNormalization, BypassModes: independent.BypassModeNormalizations}
				if !reflect.DeepEqual(changes[0].Normalizations(), want) || !reflect.DeepEqual(changes[0].Display().Normalizations(), want) {
					t.Fatal("reconciliation dropped actor or scalar evidence", changes[0].Normalizations())
				}
				got := changes[0].After().BypassActors
				if got == nil || actors != "retained" && len(*got) != 0 || actors == "retained" && len(*got) != 4 {
					t.Fatal("whole-object actor intent changed", got)
				}
				returned := changes[0].Normalizations()
				*returned.BypassModes[1].ActorID = 99
				returned.BypassModes[0].Normalization.RawValue = "mutated accessor"
				if !reflect.DeepEqual(changes[0].Normalizations(), want) {
					t.Fatal("reconciled actor evidence aliases returned values")
				}
				for _, desired := range []*RulesetDeclarations{nil, rsDesired(t)} {
					if got, err := ReconcileRulesets(observed, desired); err != nil || len(got) != 0 {
						t.Fatal("unmanaged selection became actionable", got, err)
					}
				}
			})
		}
	}
}

func TestRulesetBypassReconciliationKeepsNameOrderAndCanonicalNoops(t *testing.T) {
	z, a := normalizedBypassState("Z_MODE", false), normalizedBypassState("A_MODE", false)
	z.Policy.ID, z.Policy.Name = rsID(7), "z"
	a.ID, a.Policy.ID, a.Policy.Name = 5, rsID(5), "a"
	changes, err := ReconcileRulesets(rsObserved(t, z, a), rsDesired(t, z.Policy, a.Policy))
	if err != nil || len(changes) != 2 || changes[0].ID() != 5 || changes[1].ID() != 7 {
		t.Fatal("selected normalized policies lost stable name order", changes, err)
	}
	if changes[0].Normalizations().BypassModes[1].Normalization.RawValue != "A_MODE" || changes[1].Normalizations().BypassModes[1].Normalization.RawValue != "Z_MODE" {
		t.Fatal("actor evidence crossed selected IDs")
	}
	canonical := normalizedBypassState("unused", false)
	canonical.ID, canonical.Policy.ID, canonical.Policy.Name = 8, rsID(8), "canonical"
	canonical.BypassModeNormalizations = nil
	changes, err = ReconcileRulesets(rsObserved(t, canonical, z), rsDesired(t, canonical.Policy))
	if err != nil || len(changes) != 0 {
		t.Fatal("canonical or unlisted policies became actionable", changes, err)
	}
}
