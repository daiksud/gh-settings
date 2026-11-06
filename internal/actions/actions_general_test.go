package actions_test

import (
	"github.com/daiksud/gh-settings/internal/actions"
	"strings"
	"testing"
)

func TestActionsGeneralSummaryNativeValuesAndUIPaths(t *testing.T) {
	r := actions.PlanReport{Outcome: "changed", Target: "owner/current", Managed: 3, Changes: []actions.PlanChange{}, Actions: []actions.PlanChange{
		{Field: "actions.cache_retention_days", Before: float64(7), After: float64(30)},
		{Field: "actions.cache_size_gb", Before: float64(10), After: float64(20)},
		{Field: "actions.access_level", Before: "none", After: "user"},
	}}
	text, err := actions.RenderSummary(r, summaryMetadata())
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"repository.settings.actions.general.cache.cache_retention: 7 → 30", "repository.settings.actions.general.cache.cache_size_eviction_limit: 10 → 20", "not_accessible", "accessible_from_repositories_owned_by_user"} {
		if !strings.Contains(text, want) {
			t.Fatal(text, want)
		}
	}
	for i, values := range [][]any{{nil, "30", true, 0, 366, 1.5}, {nil, "20", false, 0, 10001, 1.5}, {nil, "enterprise", true, 1}} {
		for _, value := range values {
			bad := r
			bad.Actions = append([]actions.PlanChange{}, r.Actions...)
			bad.Actions[i].After = value
			if _, err := actions.RenderSummary(bad, summaryMetadata()); err == nil {
				t.Fatal("invalid comparison accepted", i, value)
			}
		}
	}
}
