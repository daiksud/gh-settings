package actions

import (
	"bytes"
	"encoding/json"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/app"
	"github.com/daiksud/gh-settings/internal/settings"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVariablePruneReportsRenderAbsentAndPreservePrivacy(t *testing.T) {
	c, _ := settings.NewVariableDelete(settings.RepositoryVariable{Name: "SAME", Value: "<before>"})
	env, _ := settings.NewEnvironmentVariableChange("prod/blue", c)
	path := filepath.Join(t.TempDir(), "report.json")
	err := writePlanReport(path, func() (app.PlanResult, error) {
		return app.PlanResult{Target: repository.Repository{Owner: "owner", Name: "current"}, Comparison: settings.Comparison{Managed: 3, Variables: []settings.VariableChange{c}, AgentsVariables: []settings.VariableChange{c}, EnvironmentVariables: []settings.EnvironmentVariableChange{env}}}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var report PlanReport
	if err := json.Unmarshal(data, &report); err != nil || report.Variables[0].Operation != "delete" || report.Variables[0].After != nil {
		t.Fatal(string(data), err)
	}
	metadata := RunMetadata{Actor: "owner", TriggeringActor: "owner", Event: "push", RunID: "1", RunAttempt: "1", WorkflowRef: "owner/current/.github/workflows/settings.yml@refs/heads/main", WorkflowSHA: strings.Repeat("a", 40), ConfigSHA: strings.Repeat("b", 40), Target: "owner/current", CLIVersion: "dev", CLISHA256: strings.Repeat("c", 64)}
	rendered, err := RenderSummary(report, metadata)
	if err != nil || strings.Count(rendered, "absent") != 3 || strings.Contains(rendered, "<before>") || !strings.Contains(rendered, "agents.variables: delete") {
		t.Fatal(rendered, err)
	}
	t.Setenv("SETTINGS_TARGET", "owner/current")
	var output bytes.Buffer
	summary := filepath.Join(t.TempDir(), "private.md")
	if err := writeSummaryMode(path, summary, "status-only", &output); err != nil {
		t.Fatal(err)
	}
	private, _ := os.ReadFile(summary)
	if strings.Contains(string(private)+output.String(), "SAME") || strings.Contains(string(private)+output.String(), "before") || !strings.Contains(string(private), "changed") {
		t.Fatal(string(private), output.String())
	}
	value := "value"
	for _, entry := range []VariablePlanChange{{Name: "SAME", Operation: "delete"}, {Name: "SAME", Operation: "delete", Before: &value, After: &value}, {Name: "SAME", Before: &value}, {Name: "SAME", Operation: "unknown", Before: &value}} {
		if _, err := validateVariableReport([]VariablePlanChange{entry}); err == nil {
			t.Fatal(entry)
		}
	}
}
