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

func TestSquashCommitReportAndSummary(t *testing.T) {
	path := filepath.Join(t.TempDir(), "plan.json")
	var stdout, stderr bytes.Buffer
	status := actions.Run([]string{"plan", path}, &stdout, &stderr, "dev", func() (app.PlanResult, error) {
		return app.PlanResult{Target: repository.Repository{Owner: "owner", Name: "current"}, Comparison: settings.Comparison{Managed: 2, Changes: []settings.Change{comparisonChange(t, "repository.squash_merge_commit_title", "COMMIT_OR_PR_TITLE", "PR_TITLE"), comparisonChange(t, "repository.squash_merge_commit_message", "COMMIT_MESSAGES", "BLANK")}}}, nil
	})
	data, err := os.ReadFile(path)
	if err != nil || status != 0 || stderr.Len() != 0 {
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
	if !strings.Contains(html.UnescapeString(rendered), `settings.repository.squash_merge_commit_title: "COMMIT_OR_PR_TITLE" → "PR_TITLE"`) || !strings.Contains(html.UnescapeString(rendered), `settings.repository.squash_merge_commit_message: "COMMIT_MESSAGES" → "BLANK"`) {
		t.Fatalf("summary=%s", rendered)
	}
	for _, field := range []string{"repository.squash_merge_commit_title", "repository.squash_merge_commit_message"} {
		before := "COMMIT_OR_PR_TITLE"
		if strings.HasSuffix(field, "message") {
			before = "COMMIT_MESSAGES"
		}
		for _, wrong := range []any{nil, true, float64(0), "", "UNKNOWN", "pr_title"} {
			for _, badBefore := range []bool{false, true} {
				c := actions.PlanChange{Field: field, Before: before, After: wrong}
				if badBefore {
					c.Before, c.After = c.After, c.Before
				}
				report.Managed = 1
				report.Changes = []actions.PlanChange{c}
				if _, err := actions.RenderSummary(report, summaryMetadata()); err == nil {
					t.Fatalf("accepted=%+v", c)
				}
			}
		}
	}
	for _, count := range []int{37, 38} {
		_, err := actions.RenderSummary(actions.PlanReport{Outcome: "no-difference", Target: "owner/current", Managed: count, Changes: []actions.PlanChange{}}, summaryMetadata())
		if (err == nil) != (count == 37) {
			t.Fatalf("count=%d err=%v", count, err)
		}
	}
}
