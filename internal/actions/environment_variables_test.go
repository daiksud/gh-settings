package actions

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/app"
	"github.com/daiksud/gh-settings/internal/settings"
)

func TestEnvironmentVariablesReportAndPrivacy(t *testing.T) {
	for _, mixed := range []bool{false, true} {
		t.Run(map[bool]string{false: "environment", true: "mixed"}[mixed], func(t *testing.T) {
			before := "environment <before>"
			change, _ := settings.NewVariableChange("SHARED", &before, "environment-after")
			scoped, _ := settings.NewEnvironmentVariableChange("prod/<blue>", change)
			comparison := settings.Comparison{Managed: 1, EnvironmentVariables: []settings.EnvironmentVariableChange{scoped}}
			if mixed {
				workflow, _ := settings.NewVariableChange("SHARED", nil, "workflow")
				comparison.Variables = []settings.VariableChange{workflow}
				comparison.Managed = 2
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
			if report.Outcome != "changed" || len(report.EnvironmentVariables) != 1 || report.EnvironmentVariables[0].Name != "SHARED" || *report.EnvironmentVariables[0].Before != before || *report.EnvironmentVariables[0].After != "environment-after" || len(report.Variables) != comparison.Managed-1 || !strings.Contains(string(data), `"environment_variables":`) {
				t.Fatal(string(data))
			}
			if mixed && (report.Variables[0].Before != nil || *report.Variables[0].After != "workflow") {
				t.Fatal(report.Variables)
			}
			metadata := RunMetadata{Actor: "owner", TriggeringActor: "owner", Event: "push", RunID: "1", RunAttempt: "1", WorkflowRef: "owner/current/.github/workflows/settings.yml@refs/heads/main", WorkflowSHA: strings.Repeat("a", 40), ConfigSHA: strings.Repeat("b", 40), Target: "owner/current", CLIVersion: "dev", CLISHA256: strings.Repeat("c", 64)}
			summary, err := RenderSummary(report, metadata)
			if err != nil || !strings.Contains(summary, "repository.settings.environments[&#34;prod/&lt;blue&gt;&#34;].variables: update") || !strings.Contains(summary, "environment &lt;before&gt;") || strings.Contains(summary, "<before>") || strings.Contains(summary, "repository.settings.actions.variables") != mixed {
				t.Fatal(summary, err)
			}
			t.Setenv("SETTINGS_TARGET", "owner/current")
			privatePath := filepath.Join(t.TempDir(), "status.md")
			var output bytes.Buffer
			if err := writeSummaryMode(path, privatePath, "status-only", &output); err != nil {
				t.Fatal(err)
			}
			private, err := os.ReadFile(privatePath)
			if err != nil {
				t.Fatal(err)
			}
			for _, s := range []string{"prod/<blue>", "SHARED", "environment-after", "environment <before>", "workflow", "repository.settings.environments.variables"} {
				if strings.Contains(string(private)+output.String(), s) {
					t.Fatalf("status-only leaked %s", s)
				}
			}
		})
	}
}

func TestEnvironmentVariablesReportValidation(t *testing.T) {
	after := "environment"
	entry := EnvironmentVariablePlanChange{Environment: "prod", Name: "SHARED", After: &after}
	base := PlanReport{Outcome: "changed", Target: "owner/current", Managed: 1, Changes: []PlanChange{}, EnvironmentVariables: []EnvironmentVariablePlanChange{entry}}
	if _, err := validatePlanReport(base, "owner/current"); err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"duplicate", "missing", "empty", "unchanged", "count", "environment", "different-environment"} {
		t.Run(mode, func(t *testing.T) {
			report := base
			report.EnvironmentVariables = append([]EnvironmentVariablePlanChange{}, base.EnvironmentVariables...)
			switch mode {
			case "duplicate":
				report.EnvironmentVariables = append(report.EnvironmentVariables, EnvironmentVariablePlanChange{Environment: "PROD", Name: "shared", After: &after})
			case "missing":
				report.EnvironmentVariables[0].After = nil
			case "empty":
				empty := ""
				report.EnvironmentVariables[0].After = &empty
			case "unchanged":
				report.EnvironmentVariables[0].Before = &after
			case "environment":
				report.EnvironmentVariables[0].Environment = ""
			case "different-environment":
				report.EnvironmentVariables = append(report.EnvironmentVariables, EnvironmentVariablePlanChange{Environment: "staging", Name: "SHARED", After: &after})
				if _, err := validatePlanReport(report, "owner/current"); err != nil {
					t.Fatal(err)
				}
				return
			case "count":
				report.Variables = []VariablePlanChange{{Name: "SHARED", After: &after}}
			}
			if _, err := validatePlanReport(report, "owner/current"); err == nil {
				t.Fatal("accepted invalid Environment report", mode)
			}
		})
	}
	base.Managed = 2
	base.Variables = []VariablePlanChange{{Name: "SHARED", After: &after}}
	if _, err := validatePlanReport(base, "owner/current"); err != nil {
		t.Fatal("same names across families rejected", err)
	}
}

func TestEnvironmentVariablesFailureResourcePrivacy(t *testing.T) {
	for _, stage := range []string{"read-failure", "invalid-config"} {
		t.Run(stage, func(t *testing.T) {
			failure := &app.PlanError{Stage: stage, Path: "repository.settings.environments.variables", Target: repository.Repository{Owner: "owner", Name: "current"}, Err: errors.New("repository.settings.environments.variables: private-name private-value api-secret")}
			path := filepath.Join(t.TempDir(), "report.json")
			if err := writePlanReport(path, func() (app.PlanResult, error) { return app.PlanResult{}, failure }); err == nil {
				t.Fatal("failure lost")
			}
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			var report PlanReport
			if err := json.Unmarshal(data, &report); err != nil {
				t.Fatal(err)
			}
			if report.Resource != "repository.settings.environments.variables" {
				t.Fatal(string(data))
			}
			valid, err := validatePlanReport(report, "owner/current")
			if err != nil {
				t.Fatal(err)
			}
			summary := renderSummary(valid, RunMetadata{Target: "owner/current"}, []string{"owner", "current", ".github/workflows/settings.yml"})
			if !strings.Contains(summary, "repository.settings.environments.variables") {
				t.Fatal(summary)
			}
			for _, private := range []string{"private-name", "private-value", "api-secret"} {
				if strings.Contains(string(data)+summary, private) {
					t.Fatal("failure leaked", private)
				}
			}
			t.Setenv("SETTINGS_TARGET", "owner/current")
			privatePath := filepath.Join(t.TempDir(), "status.md")
			if err := writeSummaryMode(path, privatePath, "status-only", nil); err == nil {
				t.Fatal("failure status lost")
			}
			private, err := os.ReadFile(privatePath)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(private), "repository.settings.environments.variables") {
				t.Fatal(string(private))
			}
		})
	}
	if validFailureResource("read-failure", "repository.settings.environments.variables private-name") || validFailureResource("changed", "repository.settings.environments.variables") {
		t.Fatal("accepted invalid resource context")
	}
}

func TestEnvironmentVariableReportRejectsUnknownFields(t *testing.T) {
	var change EnvironmentVariablePlanChange
	if err := json.Unmarshal([]byte(`{"environment":"prod","name":"A","after":"value","secret":"unexpected"}`), &change); err == nil {
		t.Fatal("accepted unknown field")
	}
}
