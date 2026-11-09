package cli_test

import (
	"bytes"
	"github.com/daiksud/gh-settings/internal/cli"
	"reflect"
	"testing"
)

func TestActionsGeneralExportFlags(t *testing.T) {
	tests := []struct {
		args  []string
		want  cli.ExportOptions
		valid bool
	}{
		{[]string{"export", "--cache-retention"}, cli.ExportOptions{CacheRetention: true}, true},
		{[]string{"export", "--cache-storage"}, cli.ExportOptions{CacheStorage: true}, true},
		{[]string{"export", "--actions-access"}, cli.ExportOptions{}, false},
		{[]string{"export", "--cache-retention", "--cache-storage", "--retention", "--actions"}, cli.ExportOptions{CacheRetention: true, CacheStorage: true, Retention: true, Actions: true}, true},
	}
	for _, flag := range []string{"--cache-retention", "--cache-storage"} {
		tests = append(tests, struct {
			args  []string
			want  cli.ExportOptions
			valid bool
		}{[]string{"export", flag, flag}, cli.ExportOptions{}, false})
	}
	for _, tt := range tests {
		called := false
		var got cli.ExportOptions
		var out, errOut bytes.Buffer
		status := cli.Run(tt.args, &out, &errOut, "dev", cli.Operations{Export: func(o cli.ExportOptions) error { called = true; got = o; return nil }})
		if (status == 0) != tt.valid || called != tt.valid || tt.valid && !reflect.DeepEqual(got, tt.want) {
			t.Fatal(tt.args, status, got, errOut.String())
		}
	}
}
