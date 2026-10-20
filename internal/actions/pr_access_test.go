package actions_test

import (
	"github.com/daiksud/gh-settings/internal/actions"
	"html"
	"strings"
	"testing"
)

func TestPRAccessReportRequiresNativeBooleanAndKnownPolicy(t *testing.T) {
	report := actions.PlanReport{Outcome: "changed", Target: "owner/current", Managed: 2, Changes: []actions.PlanChange{{Field: "repository.has_pull_requests", Before: true, After: false}, {Field: "repository.pull_request_creation_policy", Before: "ALL", After: "COLLABORATORS_ONLY"}}}
	content, err := actions.RenderSummary(report, summaryMetadata())
	if err != nil {
		t.Fatal(err)
	}
	visible := html.UnescapeString(content)
	for _, want := range []string{"settings.repository.has_pull_requests: true → false", `settings.repository.pull_request_creation_policy: "ALL" → "COLLABORATORS_ONLY"`} {
		if !strings.Contains(visible, want) {
			t.Fatal(visible)
		}
	}
	for _, tt := range []struct {
		index int
		value any
	}{{0, "false"}, {0, nil}, {0, float64(0)}, {1, "all"}, {1, "UNKNOWN"}, {1, ""}, {1, false}, {1, nil}} {
		changed := report
		changed.Changes = append([]actions.PlanChange{}, report.Changes...)
		changed.Changes[tt.index].After = tt.value
		if _, err := actions.RenderSummary(changed, summaryMetadata()); err == nil {
			t.Fatal("invalid native PR access report accepted", tt)
		}
	}
}
