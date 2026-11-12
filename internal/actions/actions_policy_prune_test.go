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

func TestActionsPolicyPruneReportsRenderAbsentAndPreservePrivacy(t *testing.T) {
	c, _ := settings.NewActionsEventPolicyDelete(settings.ActionsEventPolicy{ID: 7, Name: "<before>", Enforcement: "active", Events: []string{"push"}})
	path := filepath.Join(t.TempDir(), "report.json")
	err := writePlanReport(path, func() (app.PlanResult, error) {
		return app.PlanResult{Target: repository.Repository{Owner: "owner", Name: "current"}, Comparison: settings.Comparison{Managed: 1, ActionsPolicies: []settings.ActionsEventPolicyChange{c}}}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var report PlanReport
	if err := json.Unmarshal(data, &report); err != nil || report.ActionsPolicies[0].Operation != "delete" || report.ActionsPolicies[0].After != nil {
		t.Fatal(string(data), err)
	}
	metadata := RunMetadata{Actor: "owner", TriggeringActor: "owner", Event: "push", RunID: "1", RunAttempt: "1", WorkflowRef: "owner/current/.github/workflows/settings.yml@refs/heads/main", WorkflowSHA: strings.Repeat("a", 40), ConfigSHA: strings.Repeat("b", 40), Target: "owner/current", CLIVersion: "dev", CLISHA256: strings.Repeat("c", 64)}
	rendered, err := RenderSummary(report, metadata)
	if err != nil || strings.Count(rendered, "absent") != 1 || strings.Contains(rendered, "<before>") || !strings.Contains(rendered, "actions.policies: delete") {
		t.Fatal(rendered, err)
	}
	t.Setenv("SETTINGS_TARGET", "owner/current")
	var output bytes.Buffer
	summary := filepath.Join(t.TempDir(), "private.md")
	if err := writeSummaryMode(path, summary, "status-only", &output); err != nil {
		t.Fatal(err)
	}
	private, _ := os.ReadFile(summary)
	if strings.Contains(string(private)+output.String(), "before") || !strings.Contains(string(private), "changed") {
		t.Fatal(string(private), output.String())
	}
	for _, entry := range []ActionsEventPolicyPlanChange{{Operation: "delete"}, {Operation: "delete", Before: actionsEventPolicyReport(*c.Before()), After: actionsEventPolicyReport(*c.Before())}} {
		if _, err := validateActionsEventPolicyReport([]ActionsEventPolicyPlanChange{entry}); err == nil {
			t.Fatal(entry)
		}
	}
}
