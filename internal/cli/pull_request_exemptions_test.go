package cli

import (
	"errors"
	"github.com/daiksud/gh-settings/internal/app"
	"github.com/daiksud/gh-settings/internal/settings"

	"testing"
)

func TestPullRequestExemptionsPresentation(t *testing.T) {
	change, err := settings.NewPullRequestExemptionsChange([]string{"old-user"}, []string{})
	if err != nil {
		t.Fatal(err)
	}
	got := string(RenderComparison(settings.Comparison{Managed: 1, PullRequestExemptions: []settings.PullRequestExemptionsChange{change}}))
	if got != settings.PullRequestExemptionsPath+": [\"old-user\"] → []\n" {
		t.Fatal(got)
	}
	for _, status := range []app.ApplyOperationStatus{app.OperationFailedRead, app.OperationFailedUnknown} {
		err := applyDiagnostic(errors.New("failed"), app.ApplyResult{Operations: []app.ApplyOperation{{Kind: app.PullRequestExemptionsUpdate, Status: status}}})
		want := settings.PullRequestExemptionsPath + ": failed"
		if status == app.OperationFailedUnknown {
			want += " (write effects are unconfirmed)"
		}
		if err.Error() != want {
			t.Fatal(err)
		}
	}
}
