package cli

import (
	"bytes"
	"reflect"
	"strings"
	"testing"
)

func TestRepositoryVariableExportOptions(t *testing.T) {
	for _, tc := range []struct {
		args []string
		want []string
		fail bool
	}{
		{[]string{"export"}, nil, false},
		{[]string{"export", "--variable", "display_name", "--actions", "--variable", "EMPTY"}, []string{"DISPLAY_NAME", "EMPTY"}, false},
		{[]string{"export", "--variable"}, nil, true},
		{[]string{"export", "--variable", "A", "--variable", "a"}, nil, true},
		{[]string{"export", "--variable", "GITHUB_X"}, nil, true},
		{[]string{"export", "--variables-all"}, nil, true},
	} {
		var out, diag bytes.Buffer
		calls := 0
		code := Run(tc.args, &out, &diag, "dev", Operations{Export: func(o ExportOptions) error {
			calls++
			if !reflect.DeepEqual(o.Variables, tc.want) {
				t.Fatal(o)
			}
			return nil
		}})
		if (code != 0) != tc.fail || calls != map[bool]int{false: 1, true: 0}[tc.fail] {
			t.Fatal(tc.args, code, calls, diag.String())
		}
	}
	var out, diag bytes.Buffer
	if Run([]string{"--help"}, &out, &diag, "dev", Operations{}) != 0 || !strings.Contains(out.String(), "--variable NAME") {
		t.Fatal(out.String(), diag.String())
	}
}
