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

func TestRepositoryVariablesPlanPresentation(t *testing.T) {
	before := "old <value>"
	change, err := settings.NewVariableChange("DISPLAY_NAME", &before, "日本語-v2")
	if err != nil {
		t.Fatal(err)
	}
	comparison := settings.Comparison{Managed: 1, Variables: []settings.VariableChange{change}}
	output := string(cli.RenderComparison(comparison))
	if !strings.Contains(output, `repository.settings.actions.variables: update "DISPLAY_NAME"`) || !strings.Contains(output, `"old <value>" → "日本語-v2"`) {
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
	if !strings.Contains(string(data), `"variables":[{"name":"DISPLAY_NAME","before":"old \u003cvalue\u003e","after":"日本語-v2"}]`) {
		t.Fatal(string(data))
	}
	var report PlanReport
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatal(err)
	}
	metadata := RunMetadata{Actor: "owner", TriggeringActor: "owner", Event: "push", RunID: "1", RunAttempt: "1", WorkflowRef: "owner/current/.github/workflows/settings.yml@refs/heads/main", WorkflowSHA: strings.Repeat("a", 40), ConfigSHA: strings.Repeat("b", 40), Target: "owner/current", CLIVersion: "dev", CLISHA256: strings.Repeat("c", 64)}
	summary, err := RenderSummary(report, metadata)
	if err != nil || !strings.Contains(summary, "DISPLAY_NAME") || !strings.Contains(summary, "&lt;value&gt;") || strings.Contains(summary, "<value>") {
		t.Fatalf("summary=%s err=%v", summary, err)
	}
	t.Setenv("SETTINGS_TARGET", "owner/current")
	privatePath := filepath.Join(t.TempDir(), "status.md")
	var status bytes.Buffer
	if err := writeSummaryMode(path, privatePath, "status-only", &status); err != nil {
		t.Fatal(err)
	}
	private, err := os.ReadFile(privatePath)
	if err != nil || strings.Contains(string(private)+status.String(), "DISPLAY_NAME") || strings.Contains(string(private)+status.String(), "日本語-v2") || strings.Contains(string(private)+status.String(), "old") {
		t.Fatalf("status escaped values: %s %s %v", private, status.String(), err)
	}
}
