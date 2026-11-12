package cli_test

import (
	"bytes"
	"github.com/daiksud/gh-settings/internal/cli"
	"strings"
	"testing"
)

func TestSecuritySelectionRemovedBeforeExport(t *testing.T) {
	for _, family := range []string{"", "dependabot", "secret-scanning", "codeql-configuration", "codeql-default-setup", "ai-scan-pull-requests", "private-vulnerability-reporting", "immutable-releases", "all"} {
		args := []string{"export", "--security"}
		if family != "" {
			args = append(args, family)
		}
		called := false
		var out, errout bytes.Buffer
		status := cli.Run(args, &out, &errout, "dev", cli.Operations{Export: func(cli.ExportOptions) error { called = true; return nil }})
		if status != 1 || called || !strings.Contains(errout.String(), "observes supported Security families automatically") {
			t.Fatal(args, status, called, errout.String())
		}
	}
}
