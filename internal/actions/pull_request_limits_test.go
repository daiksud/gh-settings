package actions

import (
	"github.com/daiksud/gh-settings/internal/settings"
	"testing"
)

func TestPullRequestLimitsReportValidation(t *testing.T) {
	good := []PlanChange{{Field: settings.PullRequestLimitEnabledPath, Before: true, After: false}, {Field: settings.PullRequestLimitMaximumPath, Before: float64(1), After: float64(2)}, {Field: settings.PullRequestLimitDraftsPath, Before: false, After: true}}
	if c, err := validatePullRequestLimitsReport(good); err != nil || len(c) != 3 {
		t.Fatal(c, err)
	}
	for _, bad := range [][]PlanChange{{good[0], good[0]}, {{Field: settings.PullRequestLimitMaximumPath, Before: 1.5, After: 2.0}}, {{Field: settings.PullRequestLimitDraftsPath, Before: "false", After: true}}, {{Field: "unknown", Before: false, After: true}}, {{Field: settings.PullRequestLimitEnabledPath, Before: false, After: false}}} {
		if _, err := validatePullRequestLimitsReport(bad); err == nil {
			t.Fatal(bad)
		}
	}
}

func TestPullRequestLimitsManagedBound(t *testing.T) {
	report := PlanReport{Outcome: "no-difference", Target: "owner/current", Changes: []PlanChange{}, Managed: settings.PullRequestLimitsFieldCount + 3 + settings.RepositoryFieldCount() + 7 + settings.ActionsPolicyFieldCount + settings.SecurityFieldCount + settings.CodeQLConfigurationFieldCount + settings.PagesFieldCount}
	if _, err := validatePlanReport(report, "owner/current"); err != nil {
		t.Fatal(err)
	}
	report.Managed++
	if _, err := validatePlanReport(report, "owner/current"); err == nil {
		t.Fatal("out of range managed count accepted")
	}
}
