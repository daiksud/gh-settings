package githubactions

import (
	"testing"

	"github.com/daiksud/gh-settings/internal/settings"
)

func TestManagedModerationMatchesObservedPolicy(t *testing.T) {
	d := readManagedDeclaration(t)
	observed, _ := settings.NewPullRequestLimitsObservation(settings.PullRequestLimitsValues{Enabled: false, Maximum: 1, IncludeDrafts: false})
	managed, changes, err := settings.ComparePullRequestLimits(observed, d.PullRequestLimits)
	if err != nil || managed != 1 || len(changes) != 0 || d.PullRequestLimits.Enabled == nil || *d.PullRequestLimits.Enabled || d.PullRequestExemptions != nil {
		t.Fatal("operational moderation must manage only the disabled cap switch")
	}
}
