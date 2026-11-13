package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestExportObservesAllWithoutSelectors(t *testing.T) {
	var out, diag bytes.Buffer
	calls := 0
	if Run([]string{"export"}, &out, &diag, "dev", Operations{Export: func() error { calls++; return nil }}) != 0 || calls != 1 {
		t.Fatal(calls, diag.String())
	}
	for _, flag := range []string{"--rulesets", "--actions", "--actions-policies", "--workflow-permissions", "--cache-retention", "--cache-storage", "--retention", "--fork-approval", "--archive", "--pages", "--environments", "--variable", "--agent-variable", "--environment-variable", "--copilot-custom-instructions", "--copilot-mcp", "--copilot-review-effort", "--pull-request-limits", "--pull-request-limit-exemptions"} {
		diag.Reset()
		status := Run([]string{"export", flag}, &out, &diag, "dev", Operations{Export: func() error { t.Fatal("retired selector invoked API work"); return nil }})
		if status != 1 || !strings.Contains(diag.String(), "observes supported settings automatically") {
			t.Fatal(flag, status, diag.String())
		}
	}
}
