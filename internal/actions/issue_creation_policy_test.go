package actions_test

import (
	"github.com/daiksud/gh-settings/internal/actions"
	"html"
	"strings"
	"testing"
)

func TestIssueCreationPolicyReportEnum(t *testing.T) {
	report := actions.PlanReport{Outcome: "changed", Target: "owner/current", Managed: 1, Changes: []actions.PlanChange{{Field: "repository.issue_creation_policy", Before: "ALL", After: "COLLABORATORS_ONLY"}}}
	content, err := actions.RenderSummary(report, summaryMetadata())
	if err != nil || !strings.Contains(html.UnescapeString(content), `repository.settings.general.features.issue_permissions.creation_allowed_by: "all_users" → "collaborators_only"`) {
		t.Fatal(content, err)
	}
	for _, value := range []any{nil, false, float64(0), "", "all", "UNKNOWN"} {
		changed := report
		changed.Changes = append([]actions.PlanChange{}, report.Changes...)
		changed.Changes[0].After = value
		if _, err := actions.RenderSummary(changed, summaryMetadata()); err == nil {
			t.Fatal("invalid Issue policy report accepted", value)
		}
	}
}
