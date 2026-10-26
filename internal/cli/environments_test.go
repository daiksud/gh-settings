package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestEnvironmentExportOptionIsExplicitAndComposable(t *testing.T) {
	for _, args := range [][]string{{"export"}, {"export", "--environments"}, {"export", "--rulesets", "--environments"}, {"export", "--environments", "--environments"}} {
		var stdout, stderr bytes.Buffer
		calls := 0
		var got ExportOptions
		status := Run(args, &stdout, &stderr, "dev", Operations{Export: func(options ExportOptions) error { calls++; got = options; return nil }})
		duplicate := len(args) == 3 && args[1] == args[2]
		if duplicate {
			if status == 0 || calls != 0 {
				t.Fatal(args, status, calls)
			}
			continue
		}
		if status != 0 || calls != 1 || got.Environments != (len(args) > 1) || got.Rulesets != (len(args) == 3) {
			t.Fatal(args, status, got, stderr.String())
		}
	}
	var stdout, stderr bytes.Buffer
	if Run([]string{"--help"}, &stdout, &stderr, "dev", Operations{}) != 0 || !strings.Contains(stdout.String(), "--environments") {
		t.Fatal(stdout.String(), stderr.String())
	}
}
