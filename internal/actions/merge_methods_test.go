package actions_test

import (
	"bytes"
	"encoding/json"
	"html"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/actions"
	"github.com/daiksud/gh-settings/internal/app"
	"github.com/daiksud/gh-settings/internal/settings"
)

func TestMergeMethodTypedReportAndSummary(t *testing.T) {
	path := filepath.Join(t.TempDir(), "plan.json")
	var stdout, stderr bytes.Buffer
	changes := []settings.Change{comparisonChange(t, "repository.allow_merge_commit", true, false), comparisonChange(t, "repository.allow_squash_merge", false, true), comparisonChange(t, "repository.allow_rebase_merge", true, false), comparisonChange(t, "repository.delete_branch_on_merge", true, false), comparisonChange(t, "repository.allow_auto_merge", true, false), comparisonChange(t, "repository.allow_update_branch", false, true)}
	status := actions.Run([]string{"plan", path}, &stdout, &stderr, "dev", func() (app.PlanResult, error) {
		return app.PlanResult{Target: repository.Repository{Owner: "owner", Name: "current"}, Comparison: settings.Comparison{Managed: 6, Changes: changes}}, nil
	})
	data, err := os.ReadFile(path)
	if status != 0 || err != nil || stderr.Len() != 0 {
		t.Fatalf("status=%d err=%v stderr=%s", status, err, &stderr)
	}
	var report actions.PlanReport
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatal(err)
	}
	rendered, err := actions.RenderSummary(report, summaryMetadata())
	if err != nil {
		t.Fatal(err)
	}
	wantPaths := []string{
		"repository.settings.general.pull_requests.allow_merge_commits",
		"repository.settings.general.pull_requests.allow_squash_merging",
		"repository.settings.general.pull_requests.allow_rebase_merging",
		"repository.settings.general.pull_requests.automatically_delete_head_branches",
		"repository.settings.general.pull_requests.allow_auto_merge",
		"repository.settings.general.pull_requests.always_suggest_updating_pull_request_branches",
	}
	if len(report.Changes) != len(wantPaths) {
		t.Fatalf("lost report fields: %s", data)
	}
	for i, change := range report.Changes {
		if _, ok := change.Before.(bool); !ok {
			t.Fatalf("lost before bool: %s", data)
		}
		if _, ok := change.After.(bool); !ok {
			t.Fatalf("lost after bool: %s", data)
		}
		if !strings.Contains(html.UnescapeString(rendered), wantPaths[i]+": ") {
			t.Fatalf("missing field: %s", rendered)
		}
	}
	for _, field := range []string{"repository.allow_merge_commit", "repository.allow_squash_merge", "repository.allow_rebase_merge", "repository.delete_branch_on_merge", "repository.allow_auto_merge", "repository.allow_update_branch"} {
		for _, wrong := range []any{"false", nil, float64(0)} {
			report.Changes = []actions.PlanChange{{Field: field, Before: true, After: wrong}}
			report.Managed = 1
			if _, err := actions.RenderSummary(report, summaryMetadata()); err == nil {
				t.Fatalf("accepted non-boolean: %v", wrong)
			}
		}
	}
	if _, err := actions.RenderSummary(actions.PlanReport{Outcome: "no-difference", Target: "owner/current", Managed: 15, Changes: []actions.PlanChange{}}, summaryMetadata()); err != nil {
		t.Fatalf("new maximum managed count: %v", err)
	}
}
