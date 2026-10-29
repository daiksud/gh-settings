package actions_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/actions"
	"github.com/daiksud/gh-settings/internal/app"
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
