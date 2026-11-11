package actions

import (
	"bytes"
	"encoding/json"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/app"
	"github.com/daiksud/gh-settings/internal/cli"
	"github.com/daiksud/gh-settings/internal/settings"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCopilotInstructionsPlanReport(t *testing.T) {
	change, err := settings.NewCopilotInstructionsChange(true, false)
	if err != nil {
		t.Fatal(err)
	}
	comparison := settings.Comparison{Managed: 1, CopilotInstructions: []settings.CopilotInstructionsChange{change}}
	line := "repository.settings.copilot.code_review.use_custom_instructions: true → false"
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
	if len(report.CopilotInstructions) != 1 || report.CopilotInstructions[0].Before != true || report.CopilotInstructions[0].After != false {
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
	if err != nil || strings.Contains(string(saved)+output.String(), "use_custom_instructions") || strings.Contains(string(saved), "true → false") {
		t.Fatal(string(saved), output.String(), err)
	}
	for _, bad := range [][]PlanChange{
		{{Field: "custom_instructions_enabled", Before: true, After: false}},
		{{Field: settings.CopilotInstructionsPath, Before: "true", After: false}},
		{{Field: settings.CopilotInstructionsPath, Before: true}},
		{{Field: settings.CopilotInstructionsPath, Before: true, After: true}},
		{{Field: settings.CopilotInstructionsPath, Before: true, After: false}, {Field: settings.CopilotInstructionsPath, Before: true, After: false}},
	} {
		report.CopilotInstructions = bad
		if _, err := RenderSummary(report, metadata); err == nil {
			t.Fatal("accepted malformed report", bad)
		}
	}
}
