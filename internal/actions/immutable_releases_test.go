package actions_test

import (
	"strings"
	"testing"

	"github.com/daiksud/gh-settings/internal/actions"
)

func TestImmutableReleasesNativeSummaryAndIndependentSecurityFields(t *testing.T) {
	report := actions.PlanReport{Outcome: "changed", Target: "owner/current", Managed: 2, Changes: []actions.PlanChange{}, Security: []actions.PlanChange{{Field: "security.secret_scanning", Before: true, After: false}, {Field: "security.immutable_releases", Before: false, After: true}}}
	text, err := actions.RenderSummary(report, summaryMetadata())
	if err != nil || !strings.Contains(text, "repository.settings.general.releases.enable_release_immutability: false → true") {
		t.Fatal(text, err)
	}
	for _, value := range []any{nil, "true", 1} {
		invalid := report
		invalid.Security = append([]actions.PlanChange{}, report.Security...)
		invalid.Security[1].After = value
		if _, err := actions.RenderSummary(invalid, summaryMetadata()); err == nil {
			t.Fatal("invalid PVR value accepted", value)
		}
	}
}
