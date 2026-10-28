package settings

import "testing"

func TestRulesetCodeAnalysisValidationAndCanonicalCopies(t *testing.T) {
	tools := []CodeScanningTool{{Tool: "z", AlertsThreshold: "none", SecurityAlertsThreshold: "none"}, {Tool: "CodeQL", AlertsThreshold: "errors", SecurityAlertsThreshold: "high_or_higher"}}
	policy := basicPolicy("analysis")
	policy.Rules = &[]RulesetRule{{Type: "code_scanning", Parameters: &RuleParameters{CodeScanningTools: &tools}}, {Type: "code_quality", Parameters: &RuleParameters{Severity: rsText("errors")}}}
	d, err := NewRulesetDeclarations([]RulesetDeclaration{policy})
	if err != nil {
		t.Fatal(err)
	}
	values := d.Values()
	if (*ruleOf(t, values[0], "code_scanning").CodeScanningTools)[0].Tool != "CodeQL" {
		t.Fatal("tool identities not canonical")
	}
	tools[0].Tool = "changed"
	*ruleOf(t, values[0], "code_quality").Severity = "all"
	if *ruleOf(t, d.Values()[0], "code_quality").Severity != "errors" {
		t.Fatal("aliased severity")
	}
	for _, alerts := range []string{"none", "errors", "errors_and_warnings", "all"} {
		for _, security := range []string{"none", "critical", "high_or_higher", "medium_or_higher", "all"} {
			r := RulesetRule{Type: "code_scanning", Parameters: &RuleParameters{CodeScanningTools: &[]CodeScanningTool{{Tool: "CodeQL", AlertsThreshold: alerts, SecurityAlertsThreshold: security}}}}
			if err := validateRule(r, true); err != nil {
				t.Fatal(err)
			}
		}
	}
	for _, severity := range []string{"errors", "warnings", "notes", "all"} {
		if err := validateRule(RulesetRule{Type: "code_quality", Parameters: &RuleParameters{Severity: &severity}}, true); err != nil {
			t.Fatal(err)
		}
	}
	valid := CodeScanningTool{Tool: "CodeQL", AlertsThreshold: "errors", SecurityAlertsThreshold: "high_or_higher"}
	for _, p := range []*RuleParameters{nil, {}, {Severity: rsText("errors")}, {CodeScanningTools: &[]CodeScanningTool{{}}}, {CodeScanningTools: &[]CodeScanningTool{valid, valid}}, {CodeScanningTools: &[]CodeScanningTool{{Tool: "CodeQL", AlertsThreshold: "warning", SecurityAlertsThreshold: "high_or_higher"}}}, {CodeScanningTools: &[]CodeScanningTool{valid}, Severity: rsText("errors")}, {CodeScanningTools: &[]CodeScanningTool{valid}, RequiredApprovingReviewCount: paramInt(1)}} {
		if err := validateRule(RulesetRule{Type: "code_scanning", Parameters: p}, false); err == nil {
			t.Fatal("accepted incomplete/mixed scanning rule", p)
		}
	}
	for _, p := range []*RuleParameters{nil, {}, {Severity: rsText("none")}, {Severity: rsText("errors"), RequiredStatusChecks: &[]RequiredStatusCheck{}}, {Severity: rsText("errors"), CodeScanningTools: &[]CodeScanningTool{valid}}} {
		if err := validateRule(RulesetRule{Type: "code_quality", Parameters: p}, false); err == nil {
			t.Fatal("accepted incomplete/mixed quality rule", p)
		}
	}
	for _, kind := range []string{"pull_request", "required_status_checks", "deletion"} {
		if err := validateRule(RulesetRule{Type: kind, Parameters: &RuleParameters{Severity: rsText("errors")}}, false); err == nil {
			t.Fatal("analysis parameter accepted for", kind)
		}
	}
	empty := []CodeScanningTool{}
	if err := validateRule(RulesetRule{Type: "code_scanning", Parameters: &RuleParameters{CodeScanningTools: &empty}}, true); err != nil {
		t.Fatal(err)
	}
	if (*ruleOf(t, d.Values()[0], "code_scanning").CodeScanningTools)[1].Tool != "z" {
		t.Fatal("aliased tools")
	}
}
