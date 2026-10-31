package actions_test

import (
	"html"
	"strings"
	"testing"

	"github.com/daiksud/gh-settings/internal/actions"
)

func TestWorkflowPermissionSummaryTypes(t *testing.T) {
	r := actions.PlanReport{Outcome: "changed", Target: "owner/current", Managed: 2, Changes: []actions.PlanChange{}, Actions: []actions.PlanChange{{Field: "actions.default_workflow_permissions", Before: "read", After: "write"}, {Field: "actions.can_approve_pull_request_reviews", Before: false, After: true}}}
	content, err := actions.RenderSummary(r, summaryMetadata())
	if err != nil || !strings.Contains(html.UnescapeString(content), `repository.settings.actions.general.workflow_permissions.default_permissions: "read_repository_contents_and_packages_permissions" → "read_and_write_permissions"`) || !strings.Contains(html.UnescapeString(content), `repository.settings.actions.general.workflow_permissions.allow_github_actions_to_create_and_approve_pull_requests: false → true`) {
		t.Fatal(content, err)
	}
	for _, value := range []any{nil, false, "READ", "unknown", float64(0)} {
		bad := r
		bad.Actions = append([]actions.PlanChange{}, r.Actions...)
		bad.Actions[0].After = value
		if _, err := actions.RenderSummary(bad, summaryMetadata()); err == nil {
			t.Fatal(value)
		}
	}
	for _, value := range []any{nil, "false", float64(0)} {
		bad := r
		bad.Actions = append([]actions.PlanChange{}, r.Actions...)
		bad.Actions[1].Before = value
		if _, err := actions.RenderSummary(bad, summaryMetadata()); err == nil {
			t.Fatal(value)
		}
	}
}
