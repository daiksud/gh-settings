package actions

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/app"
	"github.com/daiksud/gh-settings/internal/settings"
)

func TestRunnerDeletionReportBoundaryAndPrivacy(t *testing.T) {
	before, after := []string{"old"}, []string{"new"}
	update, err := settings.NewRunnerLabelsChange(settings.RunnerLabels{ID: 5, CustomLabels: &before}, settings.RunnerLabels{ID: 5, CustomLabels: &after})
	if err != nil {
		t.Fatal(err)
	}
	deletion := settings.RunnerDeletion{ID: 9007199254740993, Name: "runner-canary", OS: "os-canary", CustomLabels: []string{"label-canary"}}
	path := filepath.Join(t.TempDir(), "plan.json")
	var out, diag bytes.Buffer
	plan := func() (app.PlanResult, error) {
		return app.PlanResult{Target: repository.Repository{Owner: "owner", Name: "current"}, Comparison: settings.Comparison{Managed: 1, Runners: []settings.RunnerLabelsChange{update}, RunnerDeletions: []settings.RunnerDeletion{deletion}}}, nil
	}
	if status := Run([]string{"plan", path, "status-only"}, &out, &diag, "dev", plan); status != 0 {
		t.Fatal(diag.String())
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var report PlanReport
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatal(err)
	}
	validated, err := validatePlanReport(report, "owner/current")
	if err != nil || validated.comparison.RunnerDeletions[0].ID != deletion.ID {
		t.Fatal(validated, err)
	}
	if report.Managed != 1 || report.Outcome != "changed" || !strings.Contains(string(data), `"after":null`) {
		t.Fatal(string(data))
	}
	metadata := RunMetadata{Actor: "actor", TriggeringActor: "actor", Event: "push", RunID: "1", RunAttempt: "1", WorkflowRef: "owner/current/.github/workflows/settings.yml@refs/heads/main", WorkflowSHA: strings.Repeat("a", 40), ConfigSHA: strings.Repeat("b", 40), Target: "owner/current", CLIVersion: "v0.99.0", CLISHA256: strings.Repeat("c", 64)}
	summary, err := RenderSummary(report, metadata)
	if err != nil || !strings.Contains(summary, "runner-canary") || !strings.Contains(summary, "os-canary") || !strings.Contains(summary, "label-canary") || !strings.Contains(summary, "null") {
		t.Fatal(summary, err)
	}
	t.Setenv("SETTINGS_TARGET", "owner/current")
	private := filepath.Join(t.TempDir(), "summary")
	if status := Run([]string{"summary", path, private, "status-only"}, &out, &diag, "dev", nil); status != 0 {
		t.Fatal(diag.String())
	}
	value, err := os.ReadFile(private)
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{"runner-canary", "os-canary", "label-canary", "9007199254740993"} {
		if strings.Contains(string(value)+out.String()+diag.String(), secret) {
			t.Fatal("leaked", secret)
		}
	}

	raw := `{"runner_deletions":[{"field":"repository.settings.actions.runners","before":{"id":9007199254740993,"name":"runner","os":"linux","custom_labels":[]},"after":null}]}`
	for _, bad := range []string{strings.Replace(raw, `"after":null`, `"after":{}`, 1), strings.Replace(raw, `"name":"runner"`, `"name":"runner","busy":false`, 1), strings.Replace(raw, `"custom_labels":[]`, `"custom_labels":null`, 1), strings.Replace(raw, `9007199254740993`, `1.5`, 1), strings.Replace(raw, `"after":null`, `"After":null`, 1), `{"runner_deletions":null}`} {
		var decoded PlanReport
		if json.Unmarshal([]byte(bad), &decoded) == nil {
			t.Fatal("accepted malformed report", bad)
		}
	}
	report.RunnerDeletions = append(report.RunnerDeletions, report.RunnerDeletions[0])
	if _, err := validatePlanReport(report, "owner/current"); err == nil {
		t.Fatal("duplicate deletion accepted")
	}
	report.RunnerDeletions = report.RunnerDeletions[:1]
	report.RunnerDeletions[0].Before.ID = 5
	if _, err := validatePlanReport(report, "owner/current"); err == nil {
		t.Fatal("same runner updated and deleted")
	}
}
