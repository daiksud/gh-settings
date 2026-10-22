package actions_test

import (
	"html"
	"strings"
	"testing"

	"github.com/daiksud/gh-settings/internal/actions"
)

func TestSponsorshipsReportNativeBoolean(t *testing.T) {
	report := actions.PlanReport{Outcome: "changed", Target: "owner/current", Managed: 1, Changes: []actions.PlanChange{{Field: "repository.has_sponsorships", Before: true, After: false}}}
	content, err := actions.RenderSummary(report, summaryMetadata())
	if err != nil || !strings.Contains(html.UnescapeString(content), "settings.repository.has_sponsorships: true → false") {
		t.Fatal(content, err)
	}
	for _, value := range []any{nil, "false", float64(0)} {
		report.Changes[0].After = value
		if _, err := actions.RenderSummary(report, summaryMetadata()); err == nil {
			t.Fatal("non-boolean report accepted", value)
		}
	}
}
