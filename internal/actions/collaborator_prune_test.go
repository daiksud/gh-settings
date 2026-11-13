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

	// Both families survive integration and retain separate managed counts.
	combined := report
	id, yes, no := int64(77), true, false
	before := settings.ClassicRule{ID: &id, BranchNamePattern: "classic/*", RequireConversationResolutionBeforeMerging: &no, RequireSignedCommits: &no, RequireLinearHistory: &no, AllowDeletions: &no, LockBranch: &no}
	combined.Managed = 2
	combined.BranchProtectionRules = []settings.ClassicChange{{Before: &before, After: settings.ClassicRule{ID: &id, BranchNamePattern: "classic/*", RequireSignedCommits: &yes}}}
	both, err := actions.RenderSummary(combined, summaryMetadata())
	if err != nil || !strings.Contains(both, "require_signed_commits") || !strings.Contains(both, "remove collaborator grant") {
		t.Fatal(both, err)
	}
	combined.Managed = 1
	if _, err := actions.RenderSummary(combined, summaryMetadata()); err == nil {
		t.Fatal("classic rules and collaborator deletions collapsed into one family")
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
