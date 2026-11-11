package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestCopilotInstructionsExportFlag(t *testing.T) {
	for _, tc := range []struct {
		args           []string
		selected, fail bool
	}{
		{[]string{"export"}, false, false},
		{[]string{"export", "--actions"}, false, false},
		{[]string{"export", "--copilot-custom-instructions"}, true, false},
		{[]string{"export", "--actions", "--copilot-custom-instructions", "--variable", "BUILD_CHANNEL"}, true, false},
		{[]string{"export", "--copilot-custom-instructions", "--copilot-custom-instructions"}, false, true},
		{[]string{"plan", "--copilot-custom-instructions"}, false, true},
	} {
		var out, diag bytes.Buffer
		calls := 0
		code := Run(tc.args, &out, &diag, "dev", Operations{Export: func(o ExportOptions) error {
			calls++
			if o.CopilotInstructions != tc.selected {
				t.Fatal(o)
			}
			return nil
		}})
		if (code != 0) != tc.fail || calls != map[bool]int{false: 1, true: 0}[tc.fail] {
			t.Fatal(tc.args, code, calls, diag.String())
		}
	}
	var out, diag bytes.Buffer
	if Run([]string{"--help"}, &out, &diag, "dev", Operations{}) != 0 || !strings.Contains(out.String(), "--copilot-custom-instructions") {
		t.Fatal(out.String(), diag.String())
	}
}
