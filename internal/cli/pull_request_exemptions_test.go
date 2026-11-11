package cli

import (
	"bytes"
	"errors"
	"github.com/daiksud/gh-settings/internal/app"
	"github.com/daiksud/gh-settings/internal/settings"
	"strings"
	"testing"
)

func TestPullRequestExemptionsExportFlag(t *testing.T) {
	for _, tc := range []struct {
		args                     []string
		exemptions, limits, fail bool
	}{
		{args: []string{"export"}},
		{args: []string{"export", "--pull-request-limit-exemptions"}, exemptions: true},
		{args: []string{"export", "--pull-request-limits"}, limits: true},
		{args: []string{"export", "--pull-request-limit-exemptions", "--pull-request-limits"}, exemptions: true, limits: true},
		{args: []string{"export", "--pull-request-limit-exemptions", "--pull-request-limit-exemptions"}, fail: true},
		{args: []string{"plan", "--pull-request-limit-exemptions"}, fail: true},
	} {
		var out, diag bytes.Buffer
		calls := 0
		code := Run(tc.args, &out, &diag, "dev", Operations{Export: func(o ExportOptions) error {
			calls++
			if o.PullRequestExemptions != tc.exemptions || o.PullRequestLimits != tc.limits {
				t.Fatal(o)
			}
			return nil
		}})
		if (code != 0) != tc.fail || calls != map[bool]int{false: 1, true: 0}[tc.fail] {
			t.Fatal(tc, code, diag.String())
		}
	}
	var out, diag bytes.Buffer
	if Run([]string{"--help"}, &out, &diag, "dev", Operations{}) != 0 || !strings.Contains(out.String(), "--pull-request-limit-exemptions") || !strings.Contains(out.String(), "explicit lists replace membership and [] clears it. Administration write is required.") {
		t.Fatal(out.String(), diag.String())
	}
}
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
