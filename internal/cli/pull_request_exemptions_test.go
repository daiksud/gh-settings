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
		if err.Error() != settings.PullRequestExemptionsPath+": failed" {
			t.Fatal(err)
		}
	}
}
