package actions_test

import (
	"bytes"
	"encoding/json"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/actions"
	"github.com/daiksud/gh-settings/internal/app"
	"github.com/daiksud/gh-settings/internal/settings"
	"html"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCollaboratorsPendingReportAndPrivateSummary(t *testing.T) {
	selected := settings.CollaboratorSelection{Person: settings.CollaboratorIdentity{ID: 3123, Username: "recipient-canary"}, DeclaredUsername: "recipient-canary", Status: "pending", InvitationID: 7123}
	path := filepath.Join(t.TempDir(), "plan.json")
	var out, errout bytes.Buffer
	status := actions.Run([]string{"plan", path}, &out, &errout, "dev", func() (app.PlanResult, error) {
		return app.PlanResult{Target: repository.Repository{Owner: "owner", Name: "current"}, Comparison: settings.Comparison{Managed: 1, Collaborators: []settings.CollaboratorSelection{selected}}}, nil
	})
	if status != 0 {
		t.Fatal(errout.String())
	}
	data, _ := os.ReadFile(path)
	var report actions.PlanReport
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatal(err)
	}
	content, err := actions.RenderSummary(report, summaryMetadata())
	if err != nil || report.Outcome != "no-difference" || !strings.Contains(html.UnescapeString(content), "acceptance pending; no new invitation") {
		t.Fatal(content, err)
	}
	for _, mutate := range []func(*actions.PlanReport){func(r *actions.PlanReport) { r.Managed = 0 }, func(r *actions.PlanReport) { r.Collaborators[0].BindIdentity = nil }, func(r *actions.PlanReport) { r.Outcome = "changed" }, func(r *actions.PlanReport) { r.Collaborators[0].State = "accepted" }, func(r *actions.PlanReport) { r.Collaborators[0].InvitationID = 0 }, func(r *actions.PlanReport) { r.Collaborators = append(r.Collaborators, r.Collaborators[0]) }, func(r *actions.PlanReport) { r.Collaborators[0].ID = 0 }} {
		var invalid actions.PlanReport
		json.Unmarshal(data, &invalid)
		mutate(&invalid)
		if _, err := actions.RenderSummary(invalid, summaryMetadata()); err == nil {
			t.Fatal("invalid report accepted", invalid)
		}
	}
	// Status-only retains only the configured target and outcome.
	t.Setenv("SETTINGS_TARGET", "owner/current")
	summary := filepath.Join(t.TempDir(), "summary.md")
	status = actions.Run([]string{"summary", path, summary, "status-only"}, &out, &errout, "dev", nil)
	private, _ := os.ReadFile(summary)
	if status != 0 || strings.Contains(string(private)+out.String()+errout.String(), "recipient-canary") || strings.Contains(string(private), "3123") || strings.Contains(string(private), "7123") {
		t.Fatal(status, string(private), out.String(), errout.String())
	}
}
