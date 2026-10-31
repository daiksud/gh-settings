package actions_test

import (
	"encoding/json"
	"html"
	"os"
	"strings"
	"testing"

	"github.com/daiksud/gh-settings/internal/actions"
)

func TestRulesetCodeAnalysisReportRetainsThresholdsAndEscapesTools(t *testing.T) {
	data, err := os.ReadFile("../../tests/fixtures/ruleset-code-analysis.json")
	if err != nil {
		t.Fatal(err)
	}
	var policy map[string]any
	json.Unmarshal(data, &policy)
	delete(policy, "source")
	delete(policy, "source_type")
	data, err = json.Marshal([]any{policy})
	if err != nil {
		t.Fatal(err)
	}
	d, err := rulesetReportFixture(data)
	if err != nil {
		t.Fatal(err)
	}
	before, after := d.Rulesets.Values()[0], d.Rulesets.Values()[0]
	for i := range *after.Rules {
		r := &(*after.Rules)[i]
		if r.Type == "code_quality" {
			severity := "warnings"
			r.Parameters.Severity = &severity
		}
		if r.Type == "code_scanning" {
			(*r.Parameters.CodeScanningTools)[0].SecurityAlertsThreshold = "critical"
			(*r.Parameters.CodeScanningTools)[0].Tool = "<CodeQL>"
		}
	}
	report := actions.PlanReport{Outcome: "changed", Target: "owner/current", Managed: 1, Changes: []actions.PlanChange{}, Rulesets: []actions.RulesetPlanChange{{Operation: "update", Before: &before, After: &after}}}
	data, err = json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	var round actions.PlanReport
	if err := json.Unmarshal(data, &round); err != nil {
		t.Fatal(err)
	}
	rendered, err := actions.RenderSummary(round, summaryMetadata())
	if err != nil {
		t.Fatal(err)
	}
	visible := html.UnescapeString(rendered)
	for _, want := range []string{`"severity":"warnings_and_higher"`, `"security_alerts":"critical"`, `"tool_name":"\u003cCodeQL\u003e"`} {
		if !strings.Contains(visible, want) {
			t.Fatal("missing analysis policy", want, visible)
		}
	}
	if strings.Contains(rendered, "<CodeQL>") {
		t.Fatal("unescaped tool", rendered)
	}
	for _, pair := range [][2]string{{`"severity":"warnings"`, `"severity":"none"`}, {`"security_alerts_threshold":"critical"`, `"security_alerts_threshold":null`}, {`"code_scanning_tools":`, `"Code_scanning_tools":`}, {`"alerts_threshold":"errors"`, `"alerts_threshold":"errors","future":true`}} {
		bad := strings.Replace(string(data), pair[0], pair[1], 1)
		if bad == string(data) {
			t.Fatal(pair)
		}
		if err := json.Unmarshal([]byte(bad), &round); err == nil {
			if _, err := actions.RenderSummary(round, summaryMetadata()); err == nil {
				t.Fatal("invalid analysis report accepted", pair)
			}
		}
	}
}
