package cli_test

import (
	"bytes"
	"testing"

	"github.com/daiksud/gh-settings/internal/cli"
)

func TestSecurityExportSelectionIsExplicitAndComposable(t *testing.T) {
	for _, test := range []struct {
		args  []string
		valid bool
		want  cli.ExportOptions
	}{
		{[]string{"export"}, true, cli.ExportOptions{}},
		{[]string{"export", "--security", "secret-scanning"}, true, cli.ExportOptions{SecretScanning: true}},
		{[]string{"export", "--security", "secret-scanning", "--actions", "--rulesets", "--workflow-permissions"}, true, cli.ExportOptions{SecretScanning: true, Actions: true, Rulesets: true, WorkflowPermissions: true}},
		{[]string{"export", "--rulesets", "--security", "secret-scanning", "--actions"}, true, cli.ExportOptions{SecretScanning: true, Actions: true, Rulesets: true}},
		{[]string{"export", "--security"}, false, cli.ExportOptions{}},
		{[]string{"export", "--security", "all"}, false, cli.ExportOptions{}},
		{[]string{"export", "--security", "dependabot"}, false, cli.ExportOptions{}},
		{[]string{"export", "--security", "secret-scanning", "--security", "secret-scanning"}, false, cli.ExportOptions{}},
	} {
		called := false
		var got cli.ExportOptions
		var out, errOut bytes.Buffer
		status := cli.Run(test.args, &out, &errOut, "dev", cli.Operations{Export: func(o cli.ExportOptions) error { called = true; got = o; return nil }})
		if (status == 0) != test.valid || called != test.valid || test.valid && got != test.want {
			t.Fatal(test.args, status, called, got, test.want, errOut.String())
		}
	}
}
