package cli_test

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/daiksud/gh-settings/internal/cli"
)

func TestRetentionExportOptInAndComposition(t *testing.T) {
	for _, test := range []struct {
		args  []string
		valid bool
		want  cli.ExportOptions
	}{
		{[]string{"export"}, true, cli.ExportOptions{}},
		{[]string{"export", "--retention"}, true, cli.ExportOptions{Retention: true}},
		{[]string{"export", "--workflow-permissions", "--retention", "--actions", "--rulesets"}, true, cli.ExportOptions{Rulesets: true, Actions: true, WorkflowPermissions: true, Retention: true}},
		{[]string{"export", "--retention", "--retention"}, false, cli.ExportOptions{}},
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
