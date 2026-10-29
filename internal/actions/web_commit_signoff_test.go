package actions_test

import (
	"html"
	"strings"
	"testing"

	"github.com/daiksud/gh-settings/internal/actions"
)

func TestWebCommitSignoffSummaryRequiresNativeBooleans(t *testing.T) {
	r := actions.PlanReport{Outcome: "changed", Target: "owner/current", Managed: 19, Changes: []actions.PlanChange{{Field: "repository.web_commit_signoff_required", Before: true, After: false}}}
	text, err := actions.RenderSummary(r, summaryMetadata())
	if err != nil || !strings.Contains(html.UnescapeString(text), "repository.settings.general.commits.require_contributors_to_sign_off_on_web_based_commits: true → false") {
		t.Fatalf("summary=%s err=%v", text, err)
	}
	for _, value := range []any{nil, "false", float64(0), []string{}} {
		r.Changes[0].After = value
		if _, err := actions.RenderSummary(r, summaryMetadata()); err == nil {
			t.Fatalf("accepted %v", value)
		}
	}
}
