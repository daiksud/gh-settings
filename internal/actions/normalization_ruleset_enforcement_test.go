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

func enforcementReportChange(t *testing.T, raw, desired string) settings.RulesetChange {
	t.Helper()
	var before, after settings.RulesetDeclaration
	policy := `{"id":7,"name":"<ENFORCEMENT_OBJECT>","target":"branch","enforcement":"disabled","conditions":{"ref_name":{"include":[],"exclude":[]}},"bypass_actors":[],"rules":[]}`
	if err := json.Unmarshal([]byte(policy), &before); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(policy), &after); err != nil {
		t.Fatal(err)
	}
	after.Enforcement = &desired
	change, err := settings.NewRulesetChangeWithEnforcementNormalization("update", 7, &before, &after, settings.ObservationNormalization{Path: settings.RulesetEnforcementPath, RawValue: raw, Fallback: "disabled"})
	if err != nil {
		t.Fatal(err)
	}
	return change
}

func TestRulesetEnforcementPrivateReportRetainsClosedEvidence(t *testing.T) {
	for _, raw := range []string{"evaluate", "", "<script>ENFORCEMENT_RAW</script>\n::warning::untrusted"} {
		for _, desired := range []string{"disabled", "active"} {
			change := enforcementReportChange(t, raw, desired)
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
			if report.Outcome != "changed" || report.Managed != 1 || len(report.Rulesets) != 1 || report.Rulesets[0].EnforcementNormalization == nil || *report.Rulesets[0].EnforcementNormalization != change.EnforcementNormalization() {
				t.Fatal("provenance lost", string(data))
			}
			content, err := actions.RenderSummary(report, summaryMetadata())
			if err != nil || !strings.Contains(html.UnescapeString(content), presentation.RulesetLine(change.Display())) || strings.Contains(content, "<script>") || strings.Contains(content, "<ENFORCEMENT_OBJECT>") {
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
			if err != nil || status != 0 || stdout.String() != "plan-outcome=changed\n" || stderr.Len() != 0 || strings.Contains(string(data), "ENFORCEMENT_") || strings.Contains(string(data), settings.RulesetEnforcementPath) || strings.Contains(string(data), "Ruleset 7") {
				t.Fatal(status, string(data), stdout.String(), stderr.String(), err)
			}
		}
	}
}

func TestRulesetEnforcementReportRejectsForgedProof(t *testing.T) {
	for _, mutate := range []func(*actions.PlanReport){
		func(r *actions.PlanReport) {
			r.Rulesets[0].Operation = "create"
			r.Rulesets[0].Before = nil
			r.Rulesets[0].After.ID = nil
		},
		func(r *actions.PlanReport) { r.Rulesets[0].Before.ID = nil },
		func(r *actions.PlanReport) { *r.Rulesets[0].After.ID = 8 },
		func(r *actions.PlanReport) { *r.Rulesets[0].After.Enforcement = "evaluate" },
		func(r *actions.PlanReport) { *r.Rulesets[0].Before.Enforcement = "active" },
		func(r *actions.PlanReport) { r.Rulesets[0].EnforcementNormalization = nil },
		func(r *actions.PlanReport) { r.Rulesets[0].EnforcementNormalization.Path = "other" },
		func(r *actions.PlanReport) { r.Rulesets[0].EnforcementNormalization.RawValue = "active" },
		func(r *actions.PlanReport) { r.Rulesets[0].EnforcementNormalization.Fallback = "active" },
		func(r *actions.PlanReport) { r.Rulesets = append(r.Rulesets, r.Rulesets[0]) },
	} {
		c := enforcementReportChange(t, "", "disabled")
		record := c.EnforcementNormalization()
		r := actions.PlanReport{Outcome: "changed", Target: "owner/current", Managed: 1, Changes: []actions.PlanChange{}, Rulesets: []actions.RulesetPlanChange{{Operation: "update", Before: c.Display().Before(), After: c.Display().After(), EnforcementNormalization: &record}}}
		if _, err := actions.RenderSummary(r, summaryMetadata()); err != nil {
			t.Fatal("empty acquired raw rejected", err)
		}
		mutate(&r)
		if _, err := actions.RenderSummary(r, summaryMetadata()); err == nil {
			t.Fatal("forged report accepted", r)
		}
	}
	for _, raw := range []string{
		`null`, `{}`, `{"path":"x","fallback":"disabled"}`, `{"path":"x","raw_value":null,"fallback":"disabled"}`, `{"path":"x","raw_value":false,"fallback":"disabled"}`, `{"path":"x","raw_value":"","fallback":"disabled","extra":true}`,
	} {
		var entry actions.RulesetPlanChange
		if err := json.Unmarshal([]byte(`{"operation":"update","enforcement_normalization":`+raw+`}`), &entry); err == nil {
			t.Fatal("malformed evidence decoded", raw)
		}
	}
	var entry actions.RulesetPlanChange
	if err := json.Unmarshal([]byte(`{"operation":"update","enforcement_normalization":{"path":"x","raw_value":"","fallback":"disabled"},"extra":true}`), &entry); err == nil {
		t.Fatal("unknown entry key decoded")
	}
}
