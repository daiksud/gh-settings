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

func TestAgentsVariablesReportAndPrivacy(t *testing.T) {
	for _, mixed := range []bool{false, true} {
		t.Run(map[bool]string{false: "agents", true: "mixed"}[mixed], func(t *testing.T) {
			before := "agent <before>"
			change, _ := settings.NewVariableChange("SHARED", &before, "agent-after")
			comparison := settings.Comparison{Managed: 1, AgentsVariables: []settings.VariableChange{change}}
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
			if report.Outcome != "changed" || len(report.AgentsVariables) != 1 || report.AgentsVariables[0].Name != "SHARED" || *report.AgentsVariables[0].Before != before || *report.AgentsVariables[0].After != "agent-after" || len(report.Variables) != comparison.Managed-1 || !strings.Contains(string(data), `"agents_variables":`) {
				t.Fatal(string(data))
			}
			if mixed && (report.Variables[0].Before != nil || *report.Variables[0].After != "workflow") {
				t.Fatal(report.Variables)
			}
			metadata := RunMetadata{Actor: "owner", TriggeringActor: "owner", Event: "push", RunID: "1", RunAttempt: "1", WorkflowRef: "owner/current/.github/workflows/settings.yml@refs/heads/main", WorkflowSHA: strings.Repeat("a", 40), ConfigSHA: strings.Repeat("b", 40), Target: "owner/current", CLIVersion: "dev", CLISHA256: strings.Repeat("c", 64)}
			summary, err := RenderSummary(report, metadata)
			if err != nil || !strings.Contains(summary, "repository.settings.agents.variables: update") || !strings.Contains(summary, "agent &lt;before&gt;") || strings.Contains(summary, "<before>") || strings.Contains(summary, "repository.settings.actions.variables") != mixed {
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
			for _, s := range []string{"SHARED", "agent-after", "agent <before>", "workflow", "repository.settings.agents.variables"} {
				if strings.Contains(string(private)+output.String(), s) {
					t.Fatalf("status-only leaked %s", s)
				}
			}
		})
	}
}

func TestAgentsVariablesReportValidation(t *testing.T) {
	after := "agent"
	entry := VariablePlanChange{Name: "SHARED", After: &after}
	base := PlanReport{Outcome: "changed", Target: "owner/current", Managed: 1, Changes: []PlanChange{}, AgentsVariables: []VariablePlanChange{entry}}
	if _, err := validatePlanReport(base, "owner/current"); err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"duplicate", "missing", "empty", "unchanged", "count"} {
		t.Run(mode, func(t *testing.T) {
			report := base
			report.AgentsVariables = append([]VariablePlanChange{}, base.AgentsVariables...)
			switch mode {
			case "duplicate":
				report.AgentsVariables = append(report.AgentsVariables, VariablePlanChange{Name: "shared", After: &after})
			case "missing":
				report.AgentsVariables[0].After = nil
			case "empty":
				empty := ""
				report.AgentsVariables[0].After = &empty
			case "unchanged":
				report.AgentsVariables[0].Before = &after
			case "count":
				report.Variables = []VariablePlanChange{entry}
			}
			if _, err := validatePlanReport(report, "owner/current"); err == nil {
				t.Fatal("accepted invalid Agents report", mode)
			}
		})
	}
	base.Managed = 2
	base.Variables = []VariablePlanChange{entry}
	if _, err := validatePlanReport(base, "owner/current"); err != nil {
		t.Fatal("same names across families rejected", err)
	}
	max := settings.PullRequestLimitsFieldCount + 4 + settings.RepositoryFieldCount() + 14 + settings.ActionsPolicyFieldCount + settings.SecurityFieldCount + settings.CodeQLConfigurationFieldCount + settings.PagesFieldCount
	for _, count := range []int{max, max + 1} {
		_, err := validatePlanReport(PlanReport{Outcome: "no-difference", Target: "owner/current", Managed: count, Changes: []PlanChange{}}, "owner/current")
		if (err != nil) != (count > max) {
			t.Fatal("managed bound", count, err)
		}
	}
}

func TestAgentsVariablesFailureResourcePrivacy(t *testing.T) {
	for _, stage := range []string{"read-failure", "invalid-config"} {
		t.Run(stage, func(t *testing.T) {
			failure := &app.PlanError{Stage: stage, Path: "repository.settings.agents.variables", Target: repository.Repository{Owner: "owner", Name: "current"}, Err: errors.New("repository.settings.agents.variables: private-name private-value api-secret")}
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
			if report.Resource != "repository.settings.agents.variables" {
				t.Fatal(string(data))
			}
			valid, err := validatePlanReport(report, "owner/current")
			if err != nil {
				t.Fatal(err)
			}
			summary := renderSummary(valid, RunMetadata{Target: "owner/current"}, []string{"owner", "current", ".github/workflows/settings.yml"})
			if !strings.Contains(summary, "repository.settings.agents.variables") {
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
			if strings.Contains(string(private), "repository.settings.agents.variables") {
				t.Fatal(string(private))
			}
		})
	}
	if validFailureResource("read-failure", "repository.settings.agents.variables private-name") || validFailureResource("changed", "repository.settings.agents.variables") {
		t.Fatal("accepted invalid resource context")
	}
}
