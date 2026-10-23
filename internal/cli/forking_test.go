package cli_test

import (
	"bytes"
	"testing"

	"github.com/daiksud/gh-settings/internal/cli"
)

func TestForkingExportFlagIsOptInAndComposable(t *testing.T) {
	for _, tc := range []struct {
		args  []string
		valid bool
		want  cli.ExportOptions
	}{
		{[]string{"export"}, true, cli.ExportOptions{}},
		{[]string{"export", "--forking"}, true, cli.ExportOptions{Forking: true}},
		{[]string{"export", "--forking", "--rulesets", "--retention"}, true, cli.ExportOptions{Forking: true, Rulesets: true, Retention: true}},
		{[]string{"export", "--forking", "--forking"}, false, cli.ExportOptions{}},
		{[]string{"plan", "--forking"}, false, cli.ExportOptions{}},
		{[]string{"export", "--forking", "--archive"}, false, cli.ExportOptions{}},
	} {
		called := false
		var got cli.ExportOptions
		var out, errOut bytes.Buffer
		status := cli.Run(tc.args, &out, &errOut, "dev", cli.Operations{Export: func(o cli.ExportOptions) error { called = true; got = o; return nil }})
		if (status == 0) != tc.valid || called != tc.valid || (tc.valid && got != tc.want) {
			t.Fatal(tc.args, status, called, got, errOut.String())
		}
	}
}
