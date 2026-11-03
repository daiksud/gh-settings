package settings

import (
	"math"
	"testing"
)

func coverageValue(value float64) *float64 { return &value }
func TestCodeCoverageRequiresExplicitBoundedThresholds(t *testing.T) {
	for _, value := range []float64{0, 0.25, 80, 100} {
		p := &RuleParameters{MinimumCoverage: coverageValue(value), MaxCoverageDrop: coverageValue(value)}
		rule := RulesetRule{Type: "code_coverage", Parameters: p}
		for _, complete := range []bool{false, true} {
			if err := validateRule(rule, complete); err != nil {
				t.Fatal(err)
			}
		}
		owned := cloneRulesetRule(rule)
		*p.MinimumCoverage = 51
		*p.MaxCoverageDrop = 52
		if *owned.Parameters.MinimumCoverage != value || *owned.Parameters.MaxCoverageDrop != value {
			t.Fatal("aliased threshold")
		}
	}
	for _, p := range []*RuleParameters{nil, {}, {MinimumCoverage: coverageValue(0)}, {MaxCoverageDrop: coverageValue(0)}, {MinimumCoverage: coverageValue(-1), MaxCoverageDrop: coverageValue(1)}, {MinimumCoverage: coverageValue(101), MaxCoverageDrop: coverageValue(1)}, {MinimumCoverage: coverageValue(math.NaN()), MaxCoverageDrop: coverageValue(1)}, {MinimumCoverage: coverageValue(1), MaxCoverageDrop: coverageValue(math.Inf(1))}, {MinimumCoverage: coverageValue(1), MaxCoverageDrop: coverageValue(1), Severity: rsText("errors")}} {
		if err := validateRule(RulesetRule{Type: "code_coverage", Parameters: p}, true); err == nil {
			t.Fatal(p)
		}
	}
	for _, kind := range []string{"deletion", "pull_request", "required_status_checks", "code_quality", "code_scanning"} {
		if err := validateRule(RulesetRule{Type: kind, Parameters: &RuleParameters{MinimumCoverage: coverageValue(1), MaxCoverageDrop: coverageValue(1)}}, false); err == nil {
			t.Fatal("cross-kind threshold", kind)
		}
	}
	policy := basicPolicy("coverage")
	policy.Rules = &[]RulesetRule{{Type: "code_coverage", Parameters: &RuleParameters{MinimumCoverage: coverageValue(80), MaxCoverageDrop: coverageValue(0)}}}
	policy.Target = rsText("tag")
	if err := ValidateRuleset(policy, true); err == nil {
		t.Fatal("tag accepted")
	}
}
