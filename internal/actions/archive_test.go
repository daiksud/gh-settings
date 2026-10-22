package actions_test

import (
	"strings"
	"testing"

	"github.com/daiksud/gh-settings/internal/actions"
)

func TestArchiveOrderIsExplicitInActionSummary(t *testing.T) {
	for _, tc := range []struct {
		before, after bool
		want          string
	}{
		{true, false, "Lifecycle order: unarchive before other settings."},
		{false, true, "Lifecycle order: archive after other settings."},
	} {
		report := actions.PlanReport{Outcome: "changed", Target: "owner/current", Managed: 1, Changes: []actions.PlanChange{{Field: "repository.archived", Before: tc.before, After: tc.after}}}
		text, err := actions.RenderSummary(report, summaryMetadata())
		if err != nil || !strings.Contains(text, tc.want) {
			t.Fatal(err, text)
		}
	}
}
