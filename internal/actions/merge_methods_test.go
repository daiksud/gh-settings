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
	changes := []settings.Change{{Field: "repository.allow_merge_commit", Before: true, After: false}, {Field: "repository.allow_squash_merge", Before: false, After: true}, {Field: "repository.allow_rebase_merge", Before: true, After: false}, {Field: "repository.delete_branch_on_merge", Before: true, After: false}}
	status := actions.Run([]string{"plan", path}, &stdout, &stderr, "dev", func() (app.PlanResult, error) {
		return app.PlanResult{Target: repository.Repository{Owner: "owner", Name: "current"}, Comparison: settings.Comparison{Managed: 4, Changes: changes}}, nil
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
	for _, change := range report.Changes {
		if _, ok := change.Before.(bool); !ok {
			t.Fatalf("lost before bool: %s", data)
		}
		if _, ok := change.After.(bool); !ok {
			t.Fatalf("lost after bool: %s", data)
		}
		if !strings.Contains(html.UnescapeString(rendered), "settings."+change.Field+": ") {
			t.Fatalf("missing field: %s", rendered)
		}
	}
	for _, field := range []string{"repository.allow_merge_commit", "repository.allow_squash_merge", "repository.allow_rebase_merge", "repository.delete_branch_on_merge"} {
		for _, wrong := range []any{"false", nil, float64(0)} {
			report.Changes = []actions.PlanChange{{Field: field, Before: true, After: wrong}}
			report.Managed = 1
			if _, err := actions.RenderSummary(report, summaryMetadata()); err == nil {
				t.Fatalf("accepted non-boolean: %v", wrong)
			}
		}
	}
	if _, err := actions.RenderSummary(actions.PlanReport{Outcome: "no-difference", Target: "owner/current", Managed: 11, Changes: []actions.PlanChange{}}, summaryMetadata()); err != nil {
		t.Fatalf("new maximum managed count: %v", err)
	}
}
