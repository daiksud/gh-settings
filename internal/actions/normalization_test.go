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
	"github.com/daiksud/gh-settings/internal/github"
	"github.com/daiksud/gh-settings/internal/settings"
)

func TestWorkflowNormalizationReportRetainsEvidence(t *testing.T) {
	observed := github.NormalizeWorkflowPermissions(settings.WorkflowPermissionsState{DefaultWorkflowPermissions: "FUTURE"})
	desired := "read"
	managed, changes, err := settings.CompareWorkflowPermissions(observed, settings.ActionsPolicyDeclaration{DefaultWorkflowPermissions: &desired})
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
	if report.Outcome != "changed" || len(report.Actions) != 1 || report.Actions[0].Normalization == nil || *report.Actions[0].Normalization != observed.Normalization {
		t.Fatalf("report lost normalization evidence: %+v", report)
	}
	if _, err := actions.RenderSummary(report, summaryMetadata()); err != nil {
		t.Fatal("valid normalization failed reconstruction", err)
	}
	for _, provenance := range []*settings.ObservationNormalization{
		nil,
		{},
		{Path: "unmanaged", RawValue: "FUTURE", Fallback: observed.Normalization.Fallback},
		{Path: observed.Normalization.Path, RawValue: "read", Fallback: observed.Normalization.Fallback},
	} {
		invalid := report
		invalid.Actions = append([]actions.PlanChange{}, report.Actions...)
		invalid.Actions[0].Normalization = provenance
		if _, err := actions.RenderSummary(invalid, summaryMetadata()); err == nil {
			t.Fatalf("forged or missing normalization accepted: %+v", provenance)
		}
	}
}

func TestWorkflowNormalizationPresentationShowsProjection(t *testing.T) {
	raw := "<script>CANARY</script>\n::warning::untrusted&value"
	observed := github.NormalizeWorkflowPermissions(settings.WorkflowPermissionsState{DefaultWorkflowPermissions: raw})
	for _, desired := range []struct{ api, ui string }{{"read", "read_repository_contents_and_packages_permissions"}, {"write", "read_and_write_permissions"}} {
		managed, changes, err := settings.CompareWorkflowPermissions(observed, settings.ActionsPolicyDeclaration{DefaultWorkflowPermissions: &desired.api})
		if err != nil {
			t.Fatal(err)
		}
		comparison := settings.Comparison{Managed: managed, Actions: changes}
		output := string(cli.RenderComparison(comparison))
		if !strings.Contains(output, "Warning:") || !strings.Contains(output, "repository.settings.actions.general.workflow_permissions") || !strings.Contains(output, strconv.Quote(raw)) || !strings.Contains(output, `"read_repository_contents_and_packages_permissions"`) || !strings.Contains(output, strconv.Quote(desired.ui)) || !strings.Contains(output, "write required") || strings.Contains(output, "No differences") || strings.Contains(output, "\n::warning::") {
			t.Fatalf("projection was presented as exact equality or unsafe text: %s", output)
		}
		before, after := changes[0].Values()
		report := actions.PlanReport{Outcome: "changed", Target: "owner/current", Managed: managed, Changes: []actions.PlanChange{}, Actions: []actions.PlanChange{{Field: changes[0].Field(), Before: before, After: after, Normalization: &observed.Normalization}}}
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
		if err != nil || status != 0 || stderr.Len() != 0 || stdout.String() != "plan-outcome=changed\n" || !strings.Contains(string(private), "Outcome: changed") || strings.Contains(string(private), "CANARY") || strings.Contains(string(private), observed.Normalization.Path) || strings.Contains(string(private), observed.Normalization.Fallback) {
			t.Fatal("status-only Summary disclosed normalized policy", status, string(private), stdout.String(), stderr.String(), err)
		}
	}
}

func TestWorkflowNormalizationReportRequiresAcquiredStrings(t *testing.T) {
	for _, member := range []string{"path", "raw_value", "fallback"} {
		for _, invalid := range []any{nil, false, float64(1), []any{}, map[string]any{}} {
			fields := map[string]any{"path": "repository.settings.actions.general.workflow_permissions", "raw_value": "", "fallback": "read_repository_contents_and_packages_permissions"}
			fields[member] = invalid
			data, _ := json.Marshal(fields)
			var normalization settings.ObservationNormalization
			if err := json.Unmarshal(data, &normalization); err == nil {
				t.Fatalf("non-string provenance accepted: %s", data)
			}
		}
		fields := map[string]any{"path": "repository.settings.actions.general.workflow_permissions", "raw_value": "", "fallback": "read_repository_contents_and_packages_permissions"}
		delete(fields, member)
		data, _ := json.Marshal(fields)
		var normalization settings.ObservationNormalization
		if err := json.Unmarshal(data, &normalization); err == nil {
			t.Fatalf("missing provenance member accepted: %s", data)
		}
	}
	var normalization settings.ObservationNormalization
	if err := json.Unmarshal([]byte(`{"path":"repository.settings.actions.general.workflow_permissions","raw_value":"","fallback":"read_repository_contents_and_packages_permissions"}`), &normalization); err != nil || normalization.RawValue != "" || normalization.Path == "" {
		t.Fatalf("explicit acquired empty token rejected: %+v %v", normalization, err)
	}
}

func TestWorkflowNormalizationReportRejectsUnrelatedSections(t *testing.T) {
	normalization := github.NormalizeWorkflowPermissions(settings.WorkflowPermissionsState{DefaultWorkflowPermissions: "FUTURE"}).Normalization
	for _, section := range []string{"repository", "security"} {
		report := actions.PlanReport{Outcome: "changed", Target: "owner/current", Managed: 1, Changes: []actions.PlanChange{}}
		change := actions.PlanChange{Field: "repository.has_issues", Before: false, After: true, Normalization: &normalization}
		if section == "repository" {
			report.Changes = append(report.Changes, change)
		} else {
			change.Field = "security.private_vulnerability_reporting"
			report.Security = append(report.Security, change)
		}
		if _, err := actions.RenderSummary(report, summaryMetadata()); err == nil || !strings.Contains(err.Error(), "normalization") {
			t.Fatalf("workflow provenance on %s was ignored: %v", section, err)
		}
	}
}
