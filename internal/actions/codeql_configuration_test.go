package actions

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestCodeQLConfigurationReportNativeRoundTripAndClosedValues(t *testing.T) {
	r := PlanReport{Outcome: "changed", Target: "owner/current", Managed: 2, Changes: []PlanChange{}, CodeQLConfiguration: []PlanChange{
		{Field: "languages", Before: []string{"go"}, After: []string{"github_actions", "go"}},
		{Field: "runner", Before: map[string]any{"type": "standard_github_runner"}, After: map[string]any{"type": "labeled_runner", "label": "<&pool>"}},
	}}
	b, _ := json.Marshal(r)
	var decoded PlanReport
	if err := json.Unmarshal(b, &decoded); err != nil {
		t.Fatal(err)
	}
	v, err := validatePlanReport(decoded, "owner/current")
	if err != nil || len(v.comparison.CodeQLConfiguration) != 2 {
		t.Fatal(v, err)
	}
	if _, err := validateCodeQLConfigurationReport(append(r.CodeQLConfiguration, r.CodeQLConfiguration[0])); err == nil {
		t.Fatal("duplicate accepted")
	}
	for _, c := range []PlanChange{{Field: "languages", Before: []any{"go"}, After: []any{"actions"}}, {Field: "unknown", Before: "a", After: "b"}, {Field: "runner", Before: map[string]any{"type": "standard_github_runner"}, After: map[string]any{"type": "labeled_runner", "label": true}}, {Field: "query_suite", Before: "default", After: "default"}} {
		if _, err := validateCodeQLConfigurationReport([]PlanChange{c}); err == nil {
			t.Fatal(c)
		}
	}
	if !strings.Contains(string(b), "codeql_configuration") {
		t.Fatal(string(b))
	}
}
