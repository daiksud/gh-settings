package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestPullRequestLimitsExportFlag(t *testing.T) {
	for _, tc := range []struct {
		args           []string
		selected, fail bool
	}{
		{[]string{"export"}, false, false},
		{[]string{"export", "--actions"}, false, false},
		{[]string{"export", "--pull-request-limits"}, true, false},
		{[]string{"export", "--actions", "--pull-request-limits", "--variable", "BUILD_CHANNEL"}, true, false},
		{[]string{"export", "--pull-request-limits", "--pull-request-limits"}, false, true},
		{[]string{"plan", "--pull-request-limits"}, false, true},
	} {
		var out, diag bytes.Buffer
		calls := 0
		code := Run(tc.args, &out, &diag, "dev", Operations{Export: func(o ExportOptions) error {
			calls++
			if o.PullRequestLimits != tc.selected {
				t.Fatal(o)
			}
			return nil
		}})
		if (code != 0) != tc.fail || calls != map[bool]int{false: 1, true: 0}[tc.fail] {
			t.Fatal(tc.args, code, calls, diag.String())
		}
	}
	var out, diag bytes.Buffer
	if Run([]string{"--help"}, &out, &diag, "dev", Operations{}) != 0 || !strings.Contains(out.String(), "--pull-request-limits") || strings.Count(out.String(), "Usage: gh settings export") != 1 {
		t.Fatal(out.String(), diag.String())
	}
}
