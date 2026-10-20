package actions_test

import (
	"strings"
	"testing"

	"github.com/daiksud/gh-settings/internal/actions"
)

func TestRetentionSummaryHasNativeIntegerValues(t *testing.T) {
	r := actions.PlanReport{Outcome: "changed", Target: "owner/current", Managed: 1, Changes: []actions.PlanChange{}, Actions: []actions.PlanChange{{Field: "actions.artifact_and_log_retention_days", Before: float64(90), After: float64(30)}}}
	text, err := actions.RenderSummary(r, summaryMetadata())
	if err != nil || !strings.Contains(text, "settings.actions.artifact_and_log_retention_days: 90 → 30") {
		t.Fatal(text, err)
	}
	for _, value := range []any{nil, "30", true, 0, 401, 1.5} {
		bad := r
		bad.Actions = append([]actions.PlanChange{}, r.Actions...)
		bad.Actions[0].After = value
		if _, err := actions.RenderSummary(bad, summaryMetadata()); err == nil {
			t.Fatal("invalid integer accepted", value)
		}
	}
	bad := r
	bad.Actions = append([]actions.PlanChange{}, r.Actions...)
	bad.Actions[0].Field = "actions.maximum_allowed_days"
	if _, err := actions.RenderSummary(bad, summaryMetadata()); err == nil {
		t.Fatal("readonly field accepted")
	}
}
