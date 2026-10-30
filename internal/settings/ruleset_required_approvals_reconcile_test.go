package settings

import (
	"reflect"
	"testing"
)

func TestRulesetRequiredApprovalsReconciliationKeepsSelectedWholeObjectIntent(t *testing.T) {
	for _, both := range []bool{false, true} {
		for _, count := range []int{0, 1, 10, -1} {
			state := normalizedRequiredApprovalsState(7, "selected", 11, both)
			omitDismissalRestriction(&state.Policy)
			unlisted := normalizedRequiredApprovalsState(9, "UNMANAGED", 99, true)
			observed := rsObserved(t, state, unlisted)
			after := cloneRuleset(state.Policy)
			if count == -1 {
				rules := []RulesetRule{}
				for _, r := range *after.Rules {
					if r.Type != "pull_request" {
						rules = append(rules, r)
					}
				}
				after.Rules = &rules
			} else {
				ruleOf(t, after, "pull_request").RequiredApprovingReviewCount = paramInt(count)
			}
			changes, err := ReconcileRulesets(observed, rsDesired(t, after))
			want := RulesetNormalizations{Enforcement: state.EnforcementNormalization, QualitySeverity: state.QualitySeverityNormalization, RequiredApprovals: state.RequiredApprovalsNormalization}
			if err != nil || len(changes) != 1 || changes[0].ID() != 7 || !reflect.DeepEqual(changes[0].Normalizations(), want) || !reflect.DeepEqual(changes[0].Display().Normalizations(), want) {
				t.Fatal("selected whole update/evidence lost", changes, err)
			}
			got, err := (RulesetState{ID: 7, Policy: *changes[0].After()}).RawRequiredApprovals()
			if err != nil || (got == nil) != (count == -1) || got != nil && *got != count {
				t.Fatal("desired count changed", got, err)
			}
			for _, desired := range []*RulesetDeclarations{nil, rsDesired(t)} {
				if got, err := ReconcileRulesets(observed, desired); err != nil || len(got) != 0 {
					t.Fatal("unmanaged selection changed", got, err)
				}
			}
		}
	}
}

func TestRulesetRequiredApprovalsReconciliationKeepsIDOrderAndCanonicalNoops(t *testing.T) {
	z := normalizedRequiredApprovalsState(7, "z", 11, true)
	a := normalizedRequiredApprovalsState(5, "a", 12, true)
	omitDismissalRestriction(&z.Policy)
	omitDismissalRestriction(&a.Policy)
	changes, err := ReconcileRulesets(rsObserved(t, z, a), rsDesired(t, z.Policy, a.Policy))
	if err != nil || len(changes) != 2 || changes[0].ID() != 5 || changes[1].ID() != 7 || changes[0].Normalizations().RequiredApprovals.RawValue != 12 || changes[1].Normalizations().RequiredApprovals.RawValue != 11 {
		t.Fatal("ID order/evidence crossed", changes, err)
	}
	canonical := RulesetState{ID: 8, Policy: reviewPolicy()}
	canonical.Policy.ID = rsID(8)
	omitDismissalRestriction(&canonical.Policy)
	changes, err = ReconcileRulesets(rsObserved(t, canonical, z), rsDesired(t, canonical.Policy))
	if err != nil || len(changes) != 0 {
		t.Fatal("canonical or unlisted became actionable", changes, err)
	}
}
