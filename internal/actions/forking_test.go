package actions_test

import (
	"encoding/json"
	"html"
	"strings"
	"testing"

	"github.com/daiksud/gh-settings/internal/actions"
)

func TestForkingNativeBooleanReportAndSummary(t *testing.T) {
	for _, before := range []bool{true, false} {
		report := actions.PlanReport{Outcome: "changed", Target: "owner/current", Managed: 1, Changes: []actions.PlanChange{{Field: "repository.allow_forking", Before: before, After: !before}}}
		data, err := json.Marshal(report)
		if err != nil {
			t.Fatal(err)
		}
		var round actions.PlanReport
		if err := json.Unmarshal(data, &round); err != nil {
			t.Fatal(err)
		}
		got, err := actions.RenderSummary(round, summaryMetadata())
		want := "settings.repository.allow_forking: true → false"
		if !before {
			want = "settings.repository.allow_forking: false → true"
		}
		if err != nil || !strings.Contains(html.UnescapeString(got), want) {
			t.Fatal(got, err)
		}
		for _, bad := range []any{nil, "false", 0, []string{}} {
			report.Changes[0].After = bad
			if _, err := actions.RenderSummary(report, summaryMetadata()); err == nil {
				t.Fatal("nonboolean report accepted", bad)
			}
		}
	}
}
