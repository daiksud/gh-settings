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

func TestNormalizedCreationPolicyReportRetainsEvidence(t *testing.T) {
	for _, test := range []struct{ field, path string }{
		{"repository.issue_creation_policy", "repository.settings.general.features.issue_permissions.creation_allowed_by"},
		{"repository.pull_request_creation_policy", "repository.settings.general.features.pull_request_permissions.creation_allowed_by"},
	} {
		for _, after := range []string{"ALL", "COLLABORATORS_ONLY"} {
			field, err := settings.ParseFieldID(test.field)
			if err != nil {
				t.Fatal(err)
			}
			record := settings.ObservationNormalization{Path: test.path, RawValue: "", Fallback: "all_users"}
			change, err := settings.NewCreationPolicyChangeWithNormalization(field, "ALL", after, record)
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(t.TempDir(), "plan.json")
			var stdout, stderr bytes.Buffer
			status := actions.Run([]string{"plan", path}, &stdout, &stderr, "dev", func() (app.PlanResult, error) {
				return app.PlanResult{Target: repository.Repository{Host: "github.com", Owner: "owner", Name: "current"}, Comparison: settings.Comparison{Managed: 1, Changes: []settings.Change{change}}}, nil
			})
			data, err := os.ReadFile(path)
			if err != nil || status != 0 || stdout.Len() != 0 || stderr.Len() != 0 {
				t.Fatal("report write failed", status, stdout.String(), stderr.String(), err)
			}
			var report actions.PlanReport
			if err := json.Unmarshal(data, &report); err != nil {
				t.Fatal(err)
			}
			if report.Outcome != "changed" || report.Managed != 1 || len(report.Changes) != 1 || report.Changes[0].Field != test.field || report.Changes[0].Before != "ALL" || report.Changes[0].After != after || report.Changes[0].Normalization == nil || *report.Changes[0].Normalization != record {
				t.Fatal("report lost acquired evidence or actionable equality", report)
			}
			if _, err := actions.RenderSummary(report, summaryMetadata()); err != nil {
				t.Fatal("closed policy provenance failed reconstruction", err)
			}
		}
	}
}

func TestCreationPolicyReportRejectsUnrelatedOrForgedMetadata(t *testing.T) {
	for _, failure := range []string{"ordinary equality", "wrong path", "wrong fallback", "supported raw", "before", "after", "unrelated string", "boolean", "actions", "security", "duplicate"} {
		t.Run(failure, func(t *testing.T) {
			record := settings.ObservationNormalization{Path: "repository.settings.general.features.issue_permissions.creation_allowed_by", RawValue: "FUTURE", Fallback: "all_users"}
			entry := actions.PlanChange{Field: "repository.issue_creation_policy", Before: "ALL", After: "ALL", Normalization: &record}
			report := actions.PlanReport{Outcome: "changed", Target: "owner/current", Managed: 1, Changes: []actions.PlanChange{entry}}
			switch failure {
			case "ordinary equality":
				report.Changes[0].Normalization = nil
			case "wrong path":
				record.Path = "repository.settings.general.features.pull_request_permissions.creation_allowed_by"
			case "wrong fallback":
				record.Fallback = "collaborators_only"
			case "supported raw":
				record.RawValue = "COLLABORATORS_ONLY"
			case "before":
				report.Changes[0].Before = "COLLABORATORS_ONLY"
			case "after":
				report.Changes[0].After = "FUTURE"
			case "unrelated string":
				report.Changes[0].Field = "repository.description"
				report.Changes[0].Before = "before"
				report.Changes[0].After = "after"
			case "boolean":
				report.Changes[0].Field = "repository.has_issues"
				report.Changes[0].Before = false
				report.Changes[0].After = true
			case "actions":
				report.Changes = []actions.PlanChange{}
				entry.Field = "actions.default_workflow_permissions"
				entry.Before = "read"
				entry.After = "write"
				report.Actions = []actions.PlanChange{entry}
			case "security":
				report.Changes = []actions.PlanChange{}
				entry.Field = "security.private_vulnerability_reporting"
				entry.Before = true
				entry.After = false
				report.Security = []actions.PlanChange{entry}
			case "duplicate":
				report.Managed = 2
				report.Changes = append(report.Changes, entry)
			}
			if _, err := actions.RenderSummary(report, summaryMetadata()); err == nil {
				t.Fatal("invalid policy report accepted", failure, report)
			} else if (failure == "actions" || failure == "security" || failure == "unrelated string") && !strings.Contains(err.Error(), "normalization") {
				t.Fatal("wrong rejection masked metadata handling", failure, err)
			}
		})
	}
}

func TestNormalizedCreationPolicyPresentationRetainsUIAndPrivacy(t *testing.T) {
	raw := "<script>POLICY_CANARY</script>\n::warning::untrusted&value"
	for _, test := range []struct{ field, path string }{
		{"repository.issue_creation_policy", "repository.settings.general.features.issue_permissions.creation_allowed_by"},
		{"repository.pull_request_creation_policy", "repository.settings.general.features.pull_request_permissions.creation_allowed_by"},
	} {
		for _, desired := range []struct{ native, ui string }{{"ALL", "all_users"}, {"COLLABORATORS_ONLY", "collaborators_only"}} {
			field, err := settings.ParseFieldID(test.field)
			if err != nil {
				t.Fatal(err)
			}
			record := settings.ObservationNormalization{Path: test.path, RawValue: raw, Fallback: "all_users"}
			change, err := settings.NewCreationPolicyChangeWithNormalization(field, "ALL", desired.native, record)
			if err != nil {
				t.Fatal(err)
			}
			output := string(cli.RenderComparison(settings.Comparison{Managed: 1, Changes: []settings.Change{change}}))
			if strings.Count(output, "Warning:") != 1 || !strings.Contains(output, test.path) || !strings.Contains(output, strconv.Quote(raw)) || !strings.Contains(output, `"all_users" (normalized observation) → "`+desired.ui+`" (write required)`) || strings.Contains(output, "No differences") || strings.Contains(output, "\n::warning::") || strings.Contains(output, "settings."+test.field) {
				t.Fatal("policy projection presented as exact or unsafe native values", output)
			}
			report := actions.PlanReport{Outcome: "changed", Target: "owner/current", Managed: 1, Changes: []actions.PlanChange{{Field: test.field, Before: "ALL", After: desired.native, Normalization: &record}}}
			summary, err := actions.RenderSummary(report, summaryMetadata())
			if err != nil || !strings.Contains(html.UnescapeString(summary), strings.TrimSuffix(output, "\n")) || strings.Contains(summary, "<script>") || !strings.Contains(summary, "&lt;script&gt;") {
				t.Fatal("detailed Summary lost escaped policy decision", summary, err)
			}
			dir := t.TempDir()
			path, summaryPath := filepath.Join(dir, "plan.json"), filepath.Join(dir, "summary.md")
			data, err := json.Marshal(report)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, data, 0600); err != nil {
				t.Fatal(err)
			}
			t.Setenv("SETTINGS_TARGET", "owner/current")
			var stdout, stderr bytes.Buffer
			status := actions.Run([]string{"summary", path, summaryPath, "status-only"}, &stdout, &stderr, "dev", nil)
			private, err := os.ReadFile(summaryPath)
			if err != nil || status != 0 || stderr.Len() != 0 || stdout.String() != "plan-outcome=changed\n" || strings.Contains(string(private), "POLICY_CANARY") || strings.Contains(string(private), test.path) || strings.Contains(string(private), "all_users") || strings.Contains(string(private), "collaborators_only") {
				t.Fatal("status-only leaked policy", status, string(private), stdout.String(), stderr.String(), err)
			}
		}
		field, err := settings.ParseFieldID(test.field)
		if err != nil {
			t.Fatal(err)
		}
		ordinary, err := settings.NewStringChange(field, "ALL", "COLLABORATORS_ONLY")
		if err != nil {
			t.Fatal(err)
		}
		if output := string(cli.RenderComparison(settings.Comparison{Managed: 1, Changes: []settings.Change{ordinary}})); output != "settings."+test.field+": \"ALL\" → \"COLLABORATORS_ONLY\"\n" {
			t.Fatal("ordinary policy display changed", output)
		}
	}
}
