package settings

import (
	"math"
	"testing"
)

func coveragePolicy(minimum, drop *float64) RulesetDeclaration {
	policy := basicPolicy("coverage")
	policy.Rules = &[]RulesetRule{{Type: "code_coverage", Parameters: &RuleParameters{MinimumCoverage: minimum, MaxCoverageDrop: drop}}}
	return policy
}

func TestRulesetCoverageDefaultsAreDeclarationOnly(t *testing.T) {
	for _, value := range []float64{0, 0.25, 80.5, 100} {
		for _, omit := range []string{"neither", "minimum", "drop", "both"} {
			minimum, drop := &value, &value
			if omit == "minimum" || omit == "both" {
				minimum = nil
			}
			if omit == "drop" || omit == "both" {
				drop = nil
			}
			policy := coveragePolicy(minimum, drop)
			declared, err := NewRulesetDeclarations([]RulesetDeclaration{policy})
			if err != nil {
				t.Fatal(err)
			}
			normalized := declared.Values()[0]
			p := (*normalized.Rules)[0].Parameters
			wantMin, wantDrop := value, value
			if minimum == nil {
				wantMin = 0
			}
			if drop == nil {
				wantDrop = 0
			}
			if *p.MinimumCoverage != wantMin || *p.MaxCoverageDrop != wantDrop {
				t.Fatal("wrong declaration defaults", p)
			}
			if _, err := NewRulesetObservation([]RulesetState{{ID: 7, Policy: policy}}, 0); (err != nil) != (omit != "neither") {
				t.Fatal("observation fabricated missing threshold", omit, err)
			}
			*p.MinimumCoverage = 17
			if *(*declared.Values()[0].Rules)[0].Parameters.MinimumCoverage != wantMin {
				t.Fatal("coverage clone aliases declaration")
			}
		}
	}
}

func TestRulesetCoverageRejectsInvalidAndForeignValues(t *testing.T) {
	zero := 0.0
	for _, invalid := range []float64{-0.01, 100.01, math.NaN(), math.Inf(1), math.Inf(-1)} {
		for _, p := range []RulesetDeclaration{coveragePolicy(&invalid, &zero), coveragePolicy(&zero, &invalid)} {
			if _, err := NewRulesetDeclarations([]RulesetDeclaration{p}); err == nil {
				t.Fatal("accepted invalid declared threshold", invalid)
			}
			if _, err := NewRulesetObservation([]RulesetState{{ID: 7, Policy: p}}, 0); err == nil {
				t.Fatal("accepted invalid observed threshold", invalid)
			}
		}
	}
	policy := coveragePolicy(&zero, &zero)
	policy.Target = rsText("tag")
	if _, err := NewRulesetDeclarations([]RulesetDeclaration{policy}); err == nil {
		t.Fatal("coverage accepted on tags")
	}
	policy = coveragePolicy(&zero, &zero)
	*policy.Rules = append(*policy.Rules, (*policy.Rules)[0])
	if _, err := NewRulesetDeclarations([]RulesetDeclaration{policy}); err == nil {
		t.Fatal("duplicate coverage accepted")
	}
}

func TestRulesetCoverageWholeObjectReconciliation(t *testing.T) {
	minimum, drop := 80.5, 2.25
	before := coveragePolicy(&minimum, &drop)
	observation := rsObserved(t, RulesetState{ID: 7, Policy: before}, RulesetState{ID: 9, Policy: basicPolicy("unlisted")})
	for _, operation := range []string{"matching", "minimum", "drop", "zero", "removed"} {
		after := cloneRuleset(before)
		after.ID = rsID(7)
		switch operation {
		case "minimum":
			*(*after.Rules)[0].Parameters.MinimumCoverage = 90
		case "drop":
			*(*after.Rules)[0].Parameters.MaxCoverageDrop = 1.5
		case "zero":
			(*after.Rules)[0].Parameters = &RuleParameters{}
		case "removed":
			after.Rules = &[]RulesetRule{}
		}
		changes, err := ReconcileRulesets(observation, rsDesired(t, after))
		if err != nil || (len(changes) == 0) != (operation == "matching") {
			t.Fatal(operation, changes, err)
		}
		if len(changes) != 0 {
			if changes[0].ID() != 7 {
				t.Fatal("unlisted object changed")
			}
			fresh := rsObserved(t, RulesetState{ID: 7, Policy: *changes[0].After()})
			again, err := ReconcileRulesets(fresh, rsDesired(t, after))
			if err != nil || len(again) != 0 {
				t.Fatal("repeat is not matching", again, err)
			}
		}
	}
	if changes, err := ReconcileRulesets(observation, nil); err != nil || len(changes) != 0 {
		t.Fatal("omitted collection was managed", changes, err)
	}
}
