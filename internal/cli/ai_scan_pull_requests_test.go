package cli_test

import (
	"bytes"
	"testing"

	"github.com/daiksud/gh-settings/internal/cli"
)

func TestAIScanPullRequestsExportSelectionAndFamilyComposition(t *testing.T) {
	for _, test := range []struct {
		args          []string
		valid, secret bool
	}{
		{[]string{"export", "--security", "ai-scan-pull-requests"}, true, false},
		{[]string{"export", "--security", "ai-scan-pull-requests", "--security", "secret-scanning"}, true, true},
		{[]string{"export", "--security", "secret-scanning", "--security", "ai-scan-pull-requests"}, true, true},
		{[]string{"export", "--security", "ai-scan-pull-requests", "--security", "ai-scan-pull-requests"}, false, false},
		{[]string{"export", "--security", "ai-scan-pull-requests", "--security", "all"}, false, false},
	} {
		called := false
		var got cli.ExportOptions
		var out, errOut bytes.Buffer
		status := cli.Run(test.args, &out, &errOut, "dev", cli.Operations{Export: func(o cli.ExportOptions) error { called = true; got = o; return nil }})
		if (status == 0) != test.valid || called != test.valid || test.valid && (!got.AIScanPullRequests || got.SecretScanning != test.secret) {
			t.Fatal(test.args, status, called, got, errOut.String())
		}
	}
}
