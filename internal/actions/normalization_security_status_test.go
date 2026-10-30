package actions_test

import (
	"bytes"
	"encoding/json"
	"html"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/actions"
	"github.com/daiksud/gh-settings/internal/app"
	"github.com/daiksud/gh-settings/internal/cli"
	"github.com/daiksud/gh-settings/internal/settings"
)

func TestSecurityStatusReportRetainsTypedProvenance(t *testing.T) {
	for _, control := range []settings.SecurityStatusControl{settings.SecretProtectionControl, settings.PushProtectionControl} {
		for _, desired := range []bool{false, true} {
			record := settings.SecurityStatusNormalization{Path: control.Path(), RawValue: "", Fallback: false}
			change, err := settings.NewSecurityChangeWithNormalization(control.Field(), false, desired, record)
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(t.TempDir(), "plan.json")
			var stdout, stderr bytes.Buffer
			status := actions.Run([]string{"plan", path}, &stdout, &stderr, "dev", func() (app.PlanResult, error) {
				return app.PlanResult{Target: repository.Repository{Host: "github.com", Owner: "owner", Name: "current"}, Comparison: settings.Comparison{Managed: 1, Security: []settings.SecurityChange{change}}}, nil
			})
			if status != 0 || stdout.Len() != 0 || stderr.Len() != 0 {
				t.Fatal(status, stdout.String(), stderr.String())
			}
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			var report actions.PlanReport
			if err := json.Unmarshal(data, &report); err != nil {
				t.Fatal(err)
			}
			if report.Outcome != "changed" || report.Managed != 1 || len(report.Security) != 1 || report.Security[0].Before != false || report.Security[0].After != desired || report.Security[0].SecurityStatusNormalization == nil || *report.Security[0].SecurityStatusNormalization != record || report.Security[0].Normalization != nil {
				t.Fatal("typed evidence lost", string(data))
			}
			if _, err := actions.RenderSummary(report, summaryMetadata()); err != nil {
				t.Fatal(err)
			}
			if !desired {
				report.Security[0].SecurityStatusNormalization = nil
				if _, err := actions.RenderSummary(report, summaryMetadata()); err == nil {
					t.Fatal("ordinary equal pair accepted")
				}
			}
		}
	}
}

func TestSecurityStatusReportRejectsWrongScopeAndForgedEvidence(t *testing.T) {
	record := settings.SecurityStatusNormalization{Path: settings.SecretProtectionControl.Path(), RawValue: "FUTURE", Fallback: false}
	entry := actions.PlanChange{Field: "security.secret_scanning", Before: false, After: true, SecurityStatusNormalization: &record}
	base := actions.PlanReport{Outcome: "changed", Target: "owner/current", Managed: 1, Changes: []actions.PlanChange{}, Security: []actions.PlanChange{entry}}
	for _, mutate := range []func(*actions.PlanReport){
		func(r *actions.PlanReport) { r.Security[0].Field = "security.private_vulnerability_reporting" },
		func(r *actions.PlanReport) { r.Security[0].Before = true },
		func(r *actions.PlanReport) { r.Security[0].Before = nil },
		func(r *actions.PlanReport) { r.Security[0].After = "true" },
		func(r *actions.PlanReport) {
			forged := record
			forged.RawValue = "disabled"
			r.Security[0].SecurityStatusNormalization = &forged
		},
		func(r *actions.PlanReport) {
			forged := record
			forged.Fallback = true
			r.Security[0].SecurityStatusNormalization = &forged
		},
		func(r *actions.PlanReport) {
			r.Security[0].Normalization = &settings.ObservationNormalization{Path: settings.WorkflowPermissionsPath, RawValue: "FUTURE", Fallback: "read_repository_contents_and_packages_permissions"}
		},
		func(r *actions.PlanReport) { r.Managed = 2; r.Security = append(r.Security, entry) },
		func(r *actions.PlanReport) {
			r.Managed = 2
			r.Security[0].After = false
			r.Security = append(r.Security, actions.PlanChange{Field: "security.secret_scanning_push_protection", Before: false, After: true})
		},
	} {
		report := base
		report.Security = append([]actions.PlanChange{}, base.Security...)
		mutate(&report)
		if _, err := actions.RenderSummary(report, summaryMetadata()); err == nil {
			t.Fatalf("invalid security provenance accepted: %+v", report)
		}
	}
	for _, section := range []string{"repository", "actions"} {
		report := base
		report.Security = nil
		if section == "repository" {
			change := entry
			change.Field = "repository.has_issues"
			report.Changes = []actions.PlanChange{change}
		} else {
			change := entry
			change.Field = "actions.enabled"
			report.Actions = []actions.PlanChange{change}
		}
		if _, err := actions.RenderSummary(report, summaryMetadata()); err == nil || !strings.Contains(err.Error(), "normalization") {
			t.Fatalf("wrong-section metadata ignored: %s %v", section, err)
		}
	}
}

func TestSecurityStatusPresentationKeepsBooleanDecisionsPrivate(t *testing.T) {
	raw := "<script>SECURITY_CANARY</script>\n::warning::untrusted&value"
	for _, control := range []settings.SecurityStatusControl{settings.SecretProtectionControl, settings.PushProtectionControl} {
		for _, desired := range []bool{false, true} {
			record := settings.SecurityStatusNormalization{Path: control.Path(), RawValue: raw, Fallback: false}
			change, err := settings.NewSecurityChangeWithNormalization(control.Field(), false, desired, record)
			if err != nil {
				t.Fatal(err)
			}
			comparison := settings.Comparison{Managed: 1, Security: []settings.SecurityChange{change}}
			output := string(cli.RenderComparison(comparison))
			warning := "Warning: " + control.Path() + ": observed " + strconv.Quote(raw) + " is outside UI choices; using false (projection only)."
			decision := control.Path() + ": false (normalized observation) → " + strconv.FormatBool(desired) + " (write required)"
			if output != warning+"\n"+decision+"\n" || strings.Contains(output, "\n::warning::") {
				t.Fatal("unsafe or misleading boolean decision", output)
			}
			report := actions.PlanReport{Outcome: "changed", Target: "owner/current", Managed: 1, Changes: []actions.PlanChange{}, Security: []actions.PlanChange{{Field: control.Field(), Before: false, After: desired, SecurityStatusNormalization: &record}}}
			detailed, err := actions.RenderSummary(report, summaryMetadata())
			if err != nil || !strings.Contains(html.UnescapeString(detailed), strings.TrimSuffix(output, "\n")) || strings.Contains(detailed, "<script>") {
				t.Fatal(detailed, err)
			}
			path, summary := filepath.Join(t.TempDir(), "plan.json"), filepath.Join(t.TempDir(), "summary.md")
			data, _ := json.Marshal(report)
			if err := os.WriteFile(path, data, 0600); err != nil {
				t.Fatal(err)
			}
			t.Setenv("SETTINGS_TARGET", "owner/current")
			var stdout, stderr bytes.Buffer
			status := actions.Run([]string{"summary", path, summary, "status-only"}, &stdout, &stderr, "dev", nil)
			data, err = os.ReadFile(summary)
			if err != nil || status != 0 || stderr.Len() != 0 || stdout.String() != "plan-outcome=changed\n" || !strings.Contains(string(data), "Outcome: changed") || strings.Contains(string(data), "SECURITY_CANARY") || strings.Contains(string(data), control.Path()) {
				t.Fatal(status, stdout.String(), stderr.String(), string(data), err)
			}
		}
	}
}
