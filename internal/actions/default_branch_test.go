package actions_test

import (
	"html"
	"strings"
	"testing"

	"github.com/daiksud/gh-settings/internal/actions"
)

func TestDefaultBranchSummary(t *testing.T) {
	r := actions.PlanReport{Outcome: "changed", Target: "owner/current", Managed: 21, Changes: []actions.PlanChange{{Field: "repository.default_branch", Before: "main", After: "release/v2"}}}
	text, err := actions.RenderSummary(r, summaryMetadata())
	if err != nil || !strings.Contains(html.UnescapeString(text), `repository.settings.general.default_branch: "main" → "release/v2"`) {
		t.Fatalf("summary=%s error=%v", text, err)
	}
	for _, value := range []any{nil, false, float64(0), "", "a..b", []string{}} {
		for _, before := range []bool{false, true} {
			r.Changes[0] = actions.PlanChange{Field: "repository.default_branch", Before: "main", After: value}
			if before {
				r.Changes[0].Before, r.Changes[0].After = value, "main"
			}
			if _, err := actions.RenderSummary(r, summaryMetadata()); err == nil {
				t.Fatalf("accepted %+v", r.Changes[0])
			}
		}
	}
}
