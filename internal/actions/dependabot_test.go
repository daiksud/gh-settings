package actions_test

import (
	"strings"
	"testing"

	"github.com/daiksud/gh-settings/internal/actions"
)

func TestDependabotSummaryUsesClosedNativeBooleanFields(t *testing.T) {
	r := actions.PlanReport{Outcome: "changed", Target: "owner/current", Managed: 2, Changes: []actions.PlanChange{}, Security: []actions.PlanChange{{Field: "security.dependabot_alerts", Before: false, After: true}, {Field: "security.dependabot_security_updates", Before: false, After: true}}}
	text, err := actions.RenderSummary(r, summaryMetadata())
	if err != nil || !strings.Contains(text, "repository.settings.advanced_security.dependabot.alerts: false → true") || !strings.Contains(text, "dependabot.security_updates: false → true") {
		t.Fatal(text, err)
	}
	for _, c := range []actions.PlanChange{{Field: "security.dependabot_paused", Before: false, After: true}, {Field: "security.dependabot_alerts", Before: "false", After: true}, {Field: "security.dependabot_alerts", Before: false, After: false}} {
		r.Managed = 1
		r.Security = []actions.PlanChange{c}
		if _, err := actions.RenderSummary(r, summaryMetadata()); err == nil {
			t.Fatal("invalid report accepted", c)
		}
	}
}
