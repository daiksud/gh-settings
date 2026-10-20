package actions_test

import (
	"github.com/daiksud/gh-settings/internal/actions"
	"strings"
	"testing"
)

func TestForkSummaryShowsCredentialChoicesAndClosedEnums(t *testing.T) {
	r := actions.PlanReport{Outcome: "changed", Target: "owner/current", Managed: 2, Changes: []actions.PlanChange{}, Actions: []actions.PlanChange{{Field: "actions.fork_pr_approval_policy", Before: "first_time_contributors", After: "all_external_contributors"}, {Field: "actions.private_fork_workflows.send_secrets_and_variables", Before: false, After: true}}}
	text, err := actions.RenderSummary(r, summaryMetadata())
	if err != nil || !strings.Contains(text, "send_secrets_and_variables: false → true") || !strings.Contains(text, "all_external_contributors") {
		t.Fatal(text, err)
	}
	for _, bad := range []any{nil, true, "never", 1} {
		b := r
		b.Actions = append([]actions.PlanChange{}, r.Actions...)
		b.Actions[0].After = bad
		if _, err := actions.RenderSummary(b, summaryMetadata()); err == nil {
			t.Fatal("invalid policy accepted", bad)
		}
	}
}
