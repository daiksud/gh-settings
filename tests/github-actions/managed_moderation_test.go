package githubactions

import (
	"testing"

	"github.com/daiksud/gh-settings/internal/settings"
)

func TestManagedModerationMatchesObservedPolicy(t *testing.T) {
	d := readManagedDeclaration(t)
	observed, _ := settings.NewPullRequestLimitsObservation(settings.PullRequestLimitsValues{Enabled: false, Maximum: 1, IncludeDrafts: false})
	managed, changes, err := settings.ComparePullRequestLimits(observed, d.PullRequestLimits)
	if err != nil || managed != 3 || len(changes) != 0 || d.PullRequestExemptions == nil || len(d.PullRequestExemptions.Logins()) != 0 {
		t.Fatal("operational moderation must retain the independently observed cap and empty exemption policy")
	}
}
