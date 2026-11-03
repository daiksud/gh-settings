package settings

import (
	"reflect"
	"testing"
)

func TestRequiredDeploymentsValidationOwnershipAndCanonicalization(t *testing.T) {
	environments := []string{"staging", "Production"}
	policy := basicPolicy("deployments")
	policy.Rules = &[]RulesetRule{{Type: "required_deployments", Parameters: &RuleParameters{RequiredDeploymentEnvironments: &environments}}}
	declaration, err := NewRulesetDeclarations([]RulesetDeclaration{policy})
	if err != nil {
		t.Fatal(err)
	}
	environments[0] = "changed"
	values := declaration.Values()
	got := ruleOf(t, values[0], "required_deployments").RequiredDeploymentEnvironments
	if !reflect.DeepEqual(*got, []string{"Production", "staging"}) {
		t.Fatal(got)
	}
	(*got)[0] = "mutated"
	if (*ruleOf(t, declaration.Values()[0], "required_deployments").RequiredDeploymentEnvironments)[0] != "Production" {
		t.Fatal("aliased list")
	}
	for _, p := range []*RuleParameters{nil, {}, {RequiredDeploymentEnvironments: rsList("")}, {RequiredDeploymentEnvironments: rsList(" ")}, {RequiredDeploymentEnvironments: rsList("a\n")}, {RequiredDeploymentEnvironments: rsList("staging", "staging")}, {RequiredDeploymentEnvironments: rsList("staging"), Severity: rsText("errors")}} {
		if err := validateRule(RulesetRule{Type: "required_deployments", Parameters: p}, false); err == nil {
			t.Fatal("accepted incomplete/mixed deployment rule", p)
		}
	}
	for _, kind := range []string{"pull_request", "required_status_checks", "code_scanning", "code_quality", "deletion"} {
		if err := validateRule(RulesetRule{Type: kind, Parameters: &RuleParameters{RequiredDeploymentEnvironments: rsList("staging")}}, false); err == nil {
			t.Fatal("accepted deployments on", kind)
		}
	}
	*policy.Target = "tag"
	if err := ValidateRuleset(policy, false); err == nil {
		t.Fatal("accepted deployment rule for tags")
	}
	if err := validateRule(RulesetRule{Type: "required_deployments", Parameters: &RuleParameters{RequiredDeploymentEnvironments: rsList()}}, true); err != nil {
		t.Fatal("explicit empty list rejected", err)
	}
}
