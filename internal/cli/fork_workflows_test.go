package cli_test

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/daiksud/gh-settings/internal/cli"
)

func TestForkExportOptInAndComposition(t *testing.T) {
	for _, test := range []struct {
		args  []string
		valid bool
		want  cli.ExportOptions
	}{
		{[]string{"export"}, true, cli.ExportOptions{}},
		{[]string{"export", "--fork-approval"}, true, cli.ExportOptions{ForkApproval: true}},
		{[]string{"export", "--workflow-permissions", "--fork-approval", "--actions", "--retention", "--private-fork-workflows", "--rulesets", "--security", "secret-scanning", "--security", "private-vulnerability-reporting"}, true, cli.ExportOptions{Rulesets: true, Actions: true, WorkflowPermissions: true, SecretScanning: true, PrivateReporting: true, ForkApproval: true, Retention: true, PrivateFork: true}},
		{[]string{"export", "--private-fork-workflows"}, true, cli.ExportOptions{PrivateFork: true}},
		{[]string{"export", "--private-fork-workflows", "--private-fork-workflows"}, false, cli.ExportOptions{}},
		{[]string{"export", "--fork-approval", "--fork-approval"}, false, cli.ExportOptions{}},
	} {
		called := false
		var got cli.ExportOptions
		var out, errOut bytes.Buffer
		status := cli.Run(test.args, &out, &errOut, "dev", cli.Operations{Export: func(o cli.ExportOptions) error { called = true; got = o; return nil }})
		if (status == 0) != test.valid || called != test.valid || test.valid && !reflect.DeepEqual(got, test.want) {
			t.Fatal(test.args, status, got, test.want, errOut.String())
		}
	}
}
