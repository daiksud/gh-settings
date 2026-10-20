package cli_test

import (
	"bytes"
	"testing"

	"github.com/daiksud/gh-settings/internal/cli"
)

func TestPrivateReportingExportSelectionAndFamilyComposition(t *testing.T) {
	for _, test := range []struct {
		args          []string
		valid, secret bool
	}{
		{[]string{"export", "--security", "private-vulnerability-reporting"}, true, false},
		{[]string{"export", "--security", "private-vulnerability-reporting", "--security", "secret-scanning"}, true, true},
		{[]string{"export", "--security", "secret-scanning", "--security", "private-vulnerability-reporting"}, true, true},
		{[]string{"export", "--security", "private-vulnerability-reporting", "--security", "private-vulnerability-reporting"}, false, false},
		{[]string{"export", "--security", "private-vulnerability-reporting", "--security", "all"}, false, false},
	} {
		called := false
		var got cli.ExportOptions
		var out, errOut bytes.Buffer
		status := cli.Run(test.args, &out, &errOut, "dev", cli.Operations{Export: func(o cli.ExportOptions) error { called = true; got = o; return nil }})
		if (status == 0) != test.valid || called != test.valid || test.valid && (!got.PrivateReporting || got.SecretScanning != test.secret) {
			t.Fatal(test.args, status, called, got, errOut.String())
		}
	}
}
