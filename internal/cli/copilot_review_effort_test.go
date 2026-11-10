package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestCopilotReviewEffortExportFlag(t *testing.T) {
	for _, tc := range []struct {
		args           []string
		selected, fail bool
	}{
		{[]string{"export"}, false, false},
		{[]string{"export", "--actions"}, false, false},
		{[]string{"export", "--copilot-custom-instructions"}, false, false},
		{[]string{"export", "--copilot-custom-instructions", "--copilot-review-effort"}, true, false},
		{[]string{"export", "--copilot-review-effort"}, true, false},
		{[]string{"export", "--actions", "--copilot-review-effort", "--variable", "BUILD_CHANNEL"}, true, false},
		{[]string{"export", "--copilot-review-effort", "--copilot-review-effort"}, false, true},
		{[]string{"plan", "--copilot-review-effort"}, false, true},
	} {
		var out, diag bytes.Buffer
		calls := 0
		code := Run(tc.args, &out, &diag, "dev", Operations{Export: func(o ExportOptions) error {
			calls++
			if o.CopilotReviewEffort != tc.selected {
				t.Fatal(o)
			}
			return nil
		}})
		if (code != 0) != tc.fail || calls != map[bool]int{false: 1, true: 0}[tc.fail] {
			t.Fatal(tc.args, code, calls, diag.String())
		}
	}
	var out, diag bytes.Buffer
	if Run([]string{"--help"}, &out, &diag, "dev", Operations{}) != 0 || !strings.Contains(out.String(), "--copilot-review-effort") {
		t.Fatal(out.String(), diag.String())
	}
}
