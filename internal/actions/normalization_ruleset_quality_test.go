package actions_test

import (
	"bytes"
	"encoding/json"
	"html"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/actions"
	"github.com/daiksud/gh-settings/internal/app"
	"github.com/daiksud/gh-settings/internal/presentation"
	"github.com/daiksud/gh-settings/internal/settings"
)

func qualityReportChange(t *testing.T, raw, desired string, both bool) settings.RulesetChange {
	t.Helper()
	policy := `{"id":7,"name":"<QUALITY_OBJECT>","target":"branch","enforcement":"active","conditions":{"ref_name":{"include":[],"exclude":[]}},"bypass_actors":[],"rules":[{"type":"code_quality","parameters":{"severity":"errors"}}]}`
	var before, after settings.RulesetDeclaration
	if err := json.Unmarshal([]byte(policy), &before); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(policy), &after); err != nil {
		t.Fatal(err)
	}
	records := settings.RulesetNormalizations{QualitySeverity: settings.ObservationNormalization{Path: settings.RulesetQualitySeverityPath, RawValue: raw, Fallback: "errors"}}
	if desired == "removed" {
		after.Rules = &[]settings.RulesetRule{}
	} else {
		(*after.Rules)[0].Parameters.Severity = &desired
	}
	if both {
		status := "disabled"
		before.Enforcement, after.Enforcement = &status, &status
		records.Enforcement = settings.ObservationNormalization{Path: settings.RulesetEnforcementPath, RawValue: "evaluate", Fallback: "disabled"}
	}
	change, err := settings.NewRulesetChangeWithNormalizations("update", 7, &before, &after, records)
	if err != nil {
		t.Fatal(err)
	}
	return change
}

func TestRulesetQualityPrivateReportRetainsBothRecordsAndRemoval(t *testing.T) {
	for _, both := range []bool{false, true} {
		for _, desired := range []string{"errors", "warnings", "notes", "all", "removed"} {
			for _, raw := range []string{"", "<script>QUALITY_RAW</script>\n::warning::untrusted"} {
				change := qualityReportChange(t, raw, desired, both)
				path := filepath.Join(t.TempDir(), "plan.json")
				var stdout, stderr bytes.Buffer
				status := actions.Run([]string{"plan", path}, &stdout, &stderr, "dev", func() (app.PlanResult, error) {
					return app.PlanResult{Target: repository.Repository{Owner: "owner", Name: "current"}, Comparison: settings.Comparison{Managed: 1, Rulesets: []settings.RulesetChange{change}}}, nil
				})
				data, err := os.ReadFile(path)
				if err != nil || status != 0 || stdout.Len() != 0 || stderr.Len() != 0 {
					t.Fatal(status, err, stdout.String(), stderr.String())
				}
				var report actions.PlanReport
				if err := json.Unmarshal(data, &report); err != nil {
					t.Fatal(err)
				}
				if report.Outcome != "changed" || report.Managed != 1 || len(report.Rulesets) != 1 || report.Rulesets[0].QualitySeverityNormalization == nil || *report.Rulesets[0].QualitySeverityNormalization != change.Normalizations().QualitySeverity || (report.Rulesets[0].EnforcementNormalization != nil) != both {
					t.Fatal("record lost or invented", string(data))
				}
				if both && *report.Rulesets[0].EnforcementNormalization != change.Normalizations().Enforcement {
					t.Fatal("enforcement evidence changed")
				}
				content, err := actions.RenderSummary(report, summaryMetadata())
				if err != nil || !strings.Contains(html.UnescapeString(content), presentation.RulesetLine(change.Display())) || strings.Contains(content, "<script>") || strings.Contains(content, "<QUALITY_OBJECT>") {
					t.Fatal(content, err)
				}
				invalid := report
				invalid.Managed = 0
				if _, err := actions.RenderSummary(invalid, summaryMetadata()); err == nil {
					t.Fatal("collection undercount accepted")
				}
				private := filepath.Join(t.TempDir(), "private.md")
				t.Setenv("SETTINGS_TARGET", "owner/current")
				stdout.Reset()
				stderr.Reset()
				status = actions.Run([]string{"summary", path, private, "status-only"}, &stdout, &stderr, "dev", nil)
				data, err = os.ReadFile(private)
				if err != nil || status != 0 || stdout.String() != "plan-outcome=changed\n" || stderr.Len() != 0 || strings.Contains(string(data), "QUALITY_") || strings.Contains(string(data), settings.RulesetQualitySeverityPath) || strings.Contains(string(data), "Ruleset 7") {
					t.Fatal(status, string(data), stdout.String(), stderr.String(), err)
				}
			}
		}
	}
}

func TestRulesetQualityReportRejectsCrossRecordForgery(t *testing.T) {
	for _, mutate := range []func(*actions.PlanReport){
		func(r *actions.PlanReport) {
			r.Rulesets[0].Operation = "create"
			r.Rulesets[0].Before = nil
			r.Rulesets[0].After.ID = nil
		},
		func(r *actions.PlanReport) { r.Rulesets[0].Before.ID = nil },
		func(r *actions.PlanReport) { *r.Rulesets[0].After.ID = 8 },
		func(r *actions.PlanReport) { r.Rulesets[0].Before.Rules = &[]settings.RulesetRule{} },
		func(r *actions.PlanReport) {
			(*r.Rulesets[0].Before.Rules)[0].Parameters.Severity = func() *string { v := "warnings"; return &v }()
		},
		func(r *actions.PlanReport) { (*r.Rulesets[0].After.Rules)[0].Parameters.Severity = nil },
		func(r *actions.PlanReport) {
			r.Rulesets[0].QualitySeverityNormalization = nil
			r.Rulesets[0].EnforcementNormalization = nil
		},
		func(r *actions.PlanReport) {
			*r.Rulesets[0].QualitySeverityNormalization = settings.ObservationNormalization{}
		},
		func(r *actions.PlanReport) {
			*r.Rulesets[0].EnforcementNormalization = settings.ObservationNormalization{}
		},
		func(r *actions.PlanReport) {
			r.Rulesets[0].QualitySeverityNormalization.Path = settings.RulesetEnforcementPath
		},
		func(r *actions.PlanReport) {
			r.Rulesets[0].EnforcementNormalization.Path = settings.RulesetQualitySeverityPath
		},
		func(r *actions.PlanReport) { r.Rulesets[0].QualitySeverityNormalization.RawValue = "warnings" },
		func(r *actions.PlanReport) { r.Rulesets[0].QualitySeverityNormalization.Fallback = "all" },
		func(r *actions.PlanReport) { r.Rulesets = append(r.Rulesets, r.Rulesets[0]) },
	} {
		change := qualityReportChange(t, "", "errors", true)
		records := change.Normalizations()
		r := actions.PlanReport{Outcome: "changed", Target: "owner/current", Managed: 1, Changes: []actions.PlanChange{}, Rulesets: []actions.RulesetPlanChange{{Operation: "update", Before: change.Display().Before(), After: change.Display().After(), EnforcementNormalization: &records.Enforcement, QualitySeverityNormalization: &records.QualitySeverity}}}
		if _, err := actions.RenderSummary(r, summaryMetadata()); err != nil {
			t.Fatal("valid simultaneous record rejected", err)
		}
		mutate(&r)
		if _, err := actions.RenderSummary(r, summaryMetadata()); err == nil {
			t.Fatal("forged proof accepted", r)
		}
	}
	for _, raw := range []string{`null`, `{}`, `{"path":"x","fallback":"errors"}`, `{"path":"x","raw_value":null,"fallback":"errors"}`, `{"path":"x","raw_value":false,"fallback":"errors"}`, `{"path":"x","raw_value":"","fallback":"errors","extra":true}`} {
		var entry actions.RulesetPlanChange
		if err := json.Unmarshal([]byte(`{"operation":"update","quality_severity_normalization":`+raw+`}`), &entry); err == nil {
			t.Fatal("malformed quality metadata decoded", raw)
		}
	}
}
