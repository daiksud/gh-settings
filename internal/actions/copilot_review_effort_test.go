package actions

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/app"
	"github.com/daiksud/gh-settings/internal/cli"
	"github.com/daiksud/gh-settings/internal/settings"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCopilotReviewEffortPlanReport(t *testing.T) {
	change, err := settings.NewCopilotReviewEffortChange("default", "max")
	if err != nil {
		t.Fatal(err)
	}
	comparison := settings.Comparison{Managed: 1, CopilotReviewEffort: []settings.CopilotReviewEffortChange{change}}
	line := "repository.settings.copilot.code_review.review_effort_level: default → max"
	if output := string(cli.RenderComparison(comparison)); output != line+"\n" {
		t.Fatal(output)
	}
	path := filepath.Join(t.TempDir(), "report.json")
	if err := writePlanReport(path, func() (app.PlanResult, error) {
		return app.PlanResult{Target: repository.Repository{Owner: "owner", Name: "current"}, Comparison: comparison}, nil
	}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var report PlanReport
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatal(err)
	}
	if len(report.CopilotReviewEffort) != 1 || report.CopilotReviewEffort[0].Before != "default" || report.CopilotReviewEffort[0].After != "max" {
		t.Fatal(string(data))
	}
	metadata := RunMetadata{Actor: "owner", TriggeringActor: "owner", Event: "push", RunID: "1", RunAttempt: "1", WorkflowRef: "owner/current/.github/workflows/settings.yml@refs/heads/main", WorkflowSHA: strings.Repeat("a", 40), ConfigSHA: strings.Repeat("b", 40), Target: "owner/current", CLIVersion: "dev", CLISHA256: strings.Repeat("c", 64)}
	summary, err := RenderSummary(report, metadata)
	if err != nil || !strings.Contains(summary, line) {
		t.Fatal(summary, err)
	}
	t.Setenv("SETTINGS_TARGET", "owner/current")
	private := filepath.Join(t.TempDir(), "private.md")
	var output bytes.Buffer
	if err := writeSummaryMode(path, private, "status-only", &output); err != nil {
		t.Fatal(err)
	}
	saved, err := os.ReadFile(private)
	if err != nil || strings.Contains(string(saved)+output.String(), "review_effort_level") || strings.Contains(string(saved), "default → max") {
		t.Fatal(string(saved), output.String(), err)
	}
	for _, bad := range [][]PlanChange{
		{{Field: "review_effort", Before: "default", After: "max"}},
		{{Field: settings.CopilotReviewEffortPath, Before: true, After: "max"}},
		{{Field: settings.CopilotReviewEffortPath, Before: "default"}},
		{{Field: settings.CopilotReviewEffortPath, Before: "max", After: "max"}},
		{{Field: settings.CopilotReviewEffortPath, Before: "default", After: "future"}},
		{{Field: settings.CopilotReviewEffortPath, Before: "future", After: "max"}},
		{{Field: settings.CopilotReviewEffortPath, Before: "default", After: "max"}, {Field: settings.CopilotReviewEffortPath, Before: "default", After: "lite"}},
	} {
		report.CopilotReviewEffort = bad
		if _, err := RenderSummary(report, metadata); err == nil {
			t.Fatal("accepted malformed report", bad)
		}
	}
}

func TestCopilotReviewEffortFailureReport(t *testing.T) {
	path := filepath.Join(t.TempDir(), "failure.json")
	failure := &app.PlanError{Stage: "read-failure", Path: settings.CopilotReviewEffortPath, Target: repository.Repository{Owner: "owner", Name: "current"}, Err: fmt.Errorf("private response")}
	if err := writePlanReport(path, func() (app.PlanResult, error) { return app.PlanResult{}, failure }); err == nil {
		t.Fatal("lost failure")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var report PlanReport
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatal(err)
	}
	if report.Outcome != "read-failure" || report.Resource != settings.CopilotReviewEffortPath || len(report.CopilotReviewEffort) != 0 || strings.Contains(string(data), "private response") {
		t.Fatal(string(data))
	}
	if _, err := validatePlanReport(report, "owner/current"); err != nil {
		t.Fatal(err)
	}
}
