package settings

import (
	"reflect"
	"testing"
)

func TestRulesetQualityReconciliationRetainsWholeObjectIntent(t *testing.T) {
	for _, simultaneous := range []bool{false, true} {
		for _, desired := range []string{"errors", "warnings", "notes", "all", "removed"} {
			t.Run(desired, func(t *testing.T) {
				state := normalizedQualityState(7, "selected", "FUTURE_SEVERITY")
				if !simultaneous {
					state.EnforcementNormalization = ObservationNormalization{}
					state.Policy.Enforcement = rsText("active")
				}
				unlisted := normalizedQualityState(9, "UNMANAGED", "OTHER_SEVERITY")
				observed := rsObserved(t, state, unlisted)
				after := cloneRuleset(state.Policy)
				after.ID = rsID(7)
				if desired == "removed" {
					after.Rules = &[]RulesetRule{}
				} else {
					ruleOf(t, after, "code_quality").Severity = rsText(desired)
				}
				changes, err := ReconcileRulesets(observed, rsDesired(t, after))
				want := RulesetNormalizations{Enforcement: state.EnforcementNormalization, QualitySeverity: state.QualitySeverityNormalization}
				if err != nil || len(changes) != 1 || changes[0].ID() != 7 || !reflect.DeepEqual(changes[0].Normalizations(), want) || !reflect.DeepEqual(changes[0].Display().Normalizations(), want) {
					t.Fatal("whole update/evidence lost", changes, err)
				}
				actual, err := (RulesetState{ID: 7, Policy: *changes[0].After()}).RawQualitySeverity()
				if err != nil || (actual == nil) != (desired == "removed") || actual != nil && *actual != desired {
					t.Fatal("after quality intent changed", actual, err)
				}
				if *changes[0].After().Enforcement != *state.Policy.Enforcement {
					t.Fatal("quality invented enforcement intent")
				}
				if got, err := ReconcileRulesets(observed, nil); err != nil || len(got) != 0 {
					t.Fatal("omission managed state", got, err)
				}
				if got, err := ReconcileRulesets(observed, rsDesired(t)); err != nil || len(got) != 0 {
					t.Fatal("empty selection managed state", got, err)
				}
			})
		}
	}
}

func TestRulesetQualityReconciliationKeepsIDOrderAndCanonicalNoops(t *testing.T) {
	z := normalizedQualityState(7, "z", "Z_MODE")
	a := normalizedQualityState(5, "a", "A_MODE")
	unlisted := normalizedQualityState(9, "unlisted", "UNMANAGED_MODE")
	zDesired, aDesired := cloneRuleset(z.Policy), cloneRuleset(a.Policy)
	zDesired.ID, aDesired.ID = rsID(7), rsID(5)
	changes, err := ReconcileRulesets(rsObserved(t, z, unlisted, a), rsDesired(t, zDesired, aDesired))
	if err != nil || len(changes) != 2 || changes[0].ID() != 5 || changes[1].ID() != 7 || changes[0].Normalizations().QualitySeverity.RawValue != "A_MODE" || changes[1].Normalizations().QualitySeverity.RawValue != "Z_MODE" {
		t.Fatal("IDs/order/evidence crossed", changes, err)
	}
	canonical := normalizedQualityState(8, "canonical", "")
	canonical.EnforcementNormalization, canonical.QualitySeverityNormalization = ObservationNormalization{}, ObservationNormalization{}
	canonicalDesired := cloneRuleset(canonical.Policy)
	canonicalDesired.ID = rsID(8)
	changes, err = ReconcileRulesets(rsObserved(t, canonical, unlisted), rsDesired(t, canonicalDesired))
	if err != nil || len(changes) != 0 {
		t.Fatal("canonical or unlisted state became actionable", changes, err)
	}
}
