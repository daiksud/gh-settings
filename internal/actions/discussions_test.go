package actions_test

import (
	"html"
	"strings"
	"testing"

	"github.com/daiksud/gh-settings/internal/actions"
)

func TestDiscussionsSummaryRequiresNativeBooleans(t *testing.T) {
	r := actions.PlanReport{Outcome: "changed", Target: "owner/current", Managed: 18, Changes: []actions.PlanChange{{Field: "repository.has_discussions", Before: true, After: false}}}
	text, err := actions.RenderSummary(r, summaryMetadata())
	if err != nil || !strings.Contains(html.UnescapeString(text), "repository.settings.general.features.discussions: true → false") {
		t.Fatalf("summary=%s err=%v", text, err)
	}
	for _, value := range []any{nil, "false", float64(0), []string{}} {
		r.Changes[0].After = value
		if _, err := actions.RenderSummary(r, summaryMetadata()); err == nil {
			t.Fatalf("accepted %v", value)
		}
	}
}
