package actions

import (
	"encoding/json"
	"github.com/daiksud/gh-settings/internal/settings"
	"strings"
	"testing"
)

func TestCodeQLConfigurationReportNativeRoundTripAndClosedValues(t *testing.T) {
	r := PlanReport{Outcome: "changed", Target: "owner/current", Managed: 4, Changes: []PlanChange{}, CodeQLConfiguration: []PlanChange{
		{Field: "threat_model", Before: "remote_sources", After: "remote_and_local_sources"},
		{Field: "query_suite", Before: "default", After: "extended"},
		{Field: "languages", Before: []string{"go"}, After: []string{"github_actions", "go"}},
		{Field: "runner", Before: map[string]any{"type": "standard_github_runner"}, After: map[string]any{"type": "labeled_runner", "label": "<&pool>"}},
	}}
	b, _ := json.Marshal(r)
	var decoded PlanReport
	if err := json.Unmarshal(b, &decoded); err != nil {
		t.Fatal(err)
	}
	v, err := validatePlanReport(decoded, "owner/current")
	if err != nil || len(v.comparison.CodeQLConfiguration) != 4 {
		t.Fatal(v, err)
	}
	if _, err := validateCodeQLConfigurationReport(append(r.CodeQLConfiguration, r.CodeQLConfiguration[0])); err == nil {
		t.Fatal("duplicate accepted")
	}
	for _, c := range []PlanChange{{Field: "threat_model", Before: "remote_sources", After: "remote_and_local_sources", Normalization: &settings.ObservationNormalization{}}, {Field: "threat_model", Before: "remote_sources", After: "remote_and_local_sources", SecurityStatusNormalization: &settings.SecurityStatusNormalization{}}, {Field: "threat_model", Before: nil, After: "remote_sources"}, {Field: "threat_model", Before: "remote", After: "remote_and_local_sources"}, {Field: "threat_model", Before: "remote_sources", After: "remote_and_local"}, {Field: "threat_model", Before: "remote_sources", After: "future"}, {Field: "threat_model", Before: true, After: "remote_sources"}, {Field: "threat_model", Before: "remote_sources", After: "remote_sources"}, {Field: "languages", Before: []any{"go"}, After: []any{"actions"}}, {Field: "unknown", Before: "a", After: "b"}, {Field: "runner", Before: map[string]any{"type": "standard_github_runner"}, After: map[string]any{"type": "labeled_runner", "label": true}}, {Field: "query_suite", Before: "default", After: "default"}} {
		if _, err := validateCodeQLConfigurationReport([]PlanChange{c}); err == nil {
			t.Fatal(c)
		}
	}
	decoded.Managed = 3
	if _, err := validatePlanReport(decoded, "owner/current"); err == nil {
		t.Fatal("accepted insufficient managed count")
	}
	if !strings.Contains(string(b), "codeql_configuration") {
		t.Fatal(string(b))
	}
}
