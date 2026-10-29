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

func TestForkApprovalNormalizationReportRetainsEvidence(t *testing.T) {
	for _, raw := range []string{"", "FUTURE"} {
		for _, desired := range []string{"first_time_contributors", "first_time_contributors_new_to_github", "all_external_contributors"} {
			record := settings.ObservationNormalization{Path: settings.ForkApprovalPath, RawValue: raw, Fallback: string(settings.DefaultForkApprovalChoice)}
			observation, err := settings.NewForkApprovalObservationWithNormalization("first_time_contributors", record)
			if err != nil {
				t.Fatal(err)
			}
			managed, changes, err := settings.CompareForkApproval(observation, settings.ActionsPolicyDeclaration{ForkApprovalPolicy: &desired})
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(t.TempDir(), "plan.json")
			var stdout, stderr bytes.Buffer
			status := actions.Run([]string{"plan", path}, &stdout, &stderr, "dev", func() (app.PlanResult, error) {
				return app.PlanResult{Target: repository.Repository{Host: "github.com", Owner: "owner", Name: "current"}, Comparison: settings.Comparison{Managed: managed, Actions: changes}}, nil
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
			if report.Outcome != "changed" || report.Managed != 1 || len(report.Actions) != 1 || report.Actions[0].Normalization == nil || *report.Actions[0].Normalization != record || report.Actions[0].After != desired {
				t.Fatalf("lost fork evidence: %+v", report)
			}
			if _, err := actions.RenderSummary(report, summaryMetadata()); err != nil {
				t.Fatal(err)
			}
			for _, invalidRecord := range []*settings.ObservationNormalization{nil, {}, {Path: "unmanaged", RawValue: raw, Fallback: record.Fallback}, {Path: record.Path, RawValue: "all_external_contributors", Fallback: record.Fallback}} {
				invalid := report
				invalid.Actions = append([]actions.PlanChange{}, report.Actions...)
				invalid.Actions[0].Normalization = invalidRecord
				if _, err := actions.RenderSummary(invalid, summaryMetadata()); err == nil && desired == "first_time_contributors" {
					t.Fatalf("forged equal decision accepted: %+v", invalidRecord)
				}
				if invalidRecord != nil {
					if _, err := actions.RenderSummary(invalid, summaryMetadata()); err == nil {
						t.Fatalf("forged record accepted: %+v", invalidRecord)
					}
				}
			}
		}
	}
}

func TestForkApprovalNormalizationPresentationShowsProjection(t *testing.T) {
	raw := "<script>CANARY</script>\n::warning::untrusted&value"
	record := settings.ObservationNormalization{Path: settings.ForkApprovalPath, RawValue: raw, Fallback: string(settings.DefaultForkApprovalChoice)}
	observed, err := settings.NewForkApprovalObservationWithNormalization("first_time_contributors", record)
	if err != nil {
		t.Fatal(err)
	}
	for _, desired := range []struct{ api, ui string }{{"first_time_contributors", "require_approval_for_first_time_contributors"}, {"first_time_contributors_new_to_github", "require_approval_for_first_time_contributors_who_are_new_to_github"}, {"all_external_contributors", "require_approval_for_all_external_contributors"}} {
		managed, changes, err := settings.CompareForkApproval(observed, settings.ActionsPolicyDeclaration{ForkApprovalPolicy: &desired.api})
		if err != nil {
			t.Fatal(err)
		}
		comparison := settings.Comparison{Managed: managed, Actions: changes}
		output := string(cli.RenderComparison(comparison))
		decision := strconv.Quote(record.Fallback) + " (normalized observation) → " + strconv.Quote(desired.ui) + " (write required)"
		if strings.Count(output, "Warning:") != 1 || !strings.Contains(output, decision) {
			t.Fatalf("missing exact fork decision %q: %s", decision, output)
		}
		if !strings.Contains(output, "Warning:") || !strings.Contains(output, settings.ForkApprovalPath) || !strings.Contains(output, strconv.Quote(raw)) || !strings.Contains(output, `"require_approval_for_first_time_contributors"`) || !strings.Contains(output, strconv.Quote(desired.ui)) || !strings.Contains(output, "write required") || strings.Contains(output, "No differences") || strings.Contains(output, "\n::warning::") {
			t.Fatalf("projection was presented as exact equality or unsafe text: %s", output)
		}
		before, after := changes[0].Values()
		report := actions.PlanReport{Outcome: "changed", Target: "owner/current", Managed: managed, Changes: []actions.PlanChange{}, Actions: []actions.PlanChange{{Field: changes[0].Field(), Before: before, After: after, Normalization: &record}}}
		summary, err := actions.RenderSummary(report, summaryMetadata())
		if err != nil || !strings.Contains(html.UnescapeString(summary), output[:len(output)-1]) || strings.Contains(summary, "<script>") || !strings.Contains(summary, "&lt;script&gt;") {
			t.Fatal("detailed Summary lost safe projection warning", summary, err)
		}
		path := filepath.Join(t.TempDir(), "plan.json")
		data, _ := json.Marshal(report)
		if err := os.WriteFile(path, data, 0600); err != nil {
			t.Fatal(err)
		}
		t.Setenv("SETTINGS_TARGET", "owner/current")
		var stdout, stderr bytes.Buffer
		privatePath := filepath.Join(t.TempDir(), "summary.md")
		status := actions.Run([]string{"summary", path, privatePath, "status-only"}, &stdout, &stderr, "dev", nil)
		private, err := os.ReadFile(privatePath)
		if err != nil || status != 0 || stderr.Len() != 0 || stdout.String() != "plan-outcome=changed\n" || !strings.Contains(string(private), "Outcome: changed") || strings.Contains(string(private), "CANARY") || strings.Contains(string(private), record.Path) || strings.Contains(string(private), record.Fallback) {
			t.Fatal("status-only Summary disclosed normalized policy", status, string(private), stdout.String(), stderr.String(), err)
		}
	}
}
