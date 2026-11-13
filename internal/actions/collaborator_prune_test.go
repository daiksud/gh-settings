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

func TestCollaboratorPruneReportAndPrivacy(t *testing.T) {
	path := filepath.Join(t.TempDir(), "plan.json")
	var out, errout bytes.Buffer
	status := actions.Run([]string{"plan", path}, &out, &errout, "dev", func() (app.PlanResult, error) {
		return app.PlanResult{Target: repository.Repository{Owner: "owner", Name: "current"}, Comparison: settings.Comparison{Managed: 1, CollaboratorDeletions: []settings.CollaboratorDeletion{{Person: settings.CollaboratorIdentity{ID: 3, Username: "removed-canary"}}, {Person: settings.CollaboratorIdentity{ID: 4, Username: "cancelled-canary"}, InvitationID: 7}}}}, nil
	})
	if status != 0 {
		t.Fatal(errout.String())
	}
	data, _ := os.ReadFile(path)
	var report actions.PlanReport
	json.Unmarshal(data, &report)
	rendered, err := actions.RenderSummary(report, summaryMetadata())
	if err != nil || report.Managed != 1 || report.Outcome != "changed" || !strings.Contains(html.UnescapeString(rendered), "cancel pending invitation id 7") || !strings.Contains(rendered, "remove collaborator grant") {
		t.Fatal(rendered, err)
	}
	for _, mutate := range []func(*actions.PlanReport){func(r *actions.PlanReport) { r.Managed = 0 }, func(r *actions.PlanReport) { r.Outcome = "no-difference" }, func(r *actions.PlanReport) { r.CollaboratorDeletions[0].Operation = "delete" }, func(r *actions.PlanReport) { r.CollaboratorDeletions[1].InvitationID = 0 }, func(r *actions.PlanReport) { r.CollaboratorDeletions[0].ID = 0 }, func(r *actions.PlanReport) {
		r.CollaboratorDeletions = append(r.CollaboratorDeletions, r.CollaboratorDeletions[0])
	}, func(r *actions.PlanReport) {
		v := false
		r.Collaborators = []actions.CollaboratorPlanEntry{{Username: "removed-canary", ID: 3, State: "active", BindIdentity: &v}}
	}} {
		var invalid actions.PlanReport
		json.Unmarshal(data, &invalid)
		mutate(&invalid)
		if _, err := actions.RenderSummary(invalid, summaryMetadata()); err == nil {
			t.Fatal("invalid deletion report accepted", invalid)
		}
	}
	t.Setenv("SETTINGS_TARGET", "owner/current")
	summary := filepath.Join(t.TempDir(), "summary.md")
	if status := actions.Run([]string{"summary", path, summary, "status-only"}, &out, &errout, "dev", nil); status != 0 {
		t.Fatal(errout.String())
	}
	private, _ := os.ReadFile(summary)
	if strings.Contains(string(private)+out.String()+errout.String(), "canary") || strings.Contains(string(private), "invitation") {
		t.Fatal(string(private))
	}
}
