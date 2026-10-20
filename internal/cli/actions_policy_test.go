package cli

import (
	"bytes"
	"reflect"
	"testing"
)

func TestExportPolicyFlagsComposeAndRejectDuplicates(t *testing.T) {
	for _, test := range []struct {
		args []string
		want []bool
	}{
		{[]string{"export"}, []bool{false, false}}, {[]string{"export", "--actions"}, []bool{false, true}},
		{[]string{"export", "--rulesets"}, []bool{true, false}}, {[]string{"export", "--rulesets", "--actions"}, []bool{true, true}},
		{[]string{"export", "--actions", "--rulesets"}, []bool{true, true}},
		{[]string{"export", "--actions", "--actions"}, nil}, {[]string{"export", "--rulesets", "--rulesets"}, nil}, {[]string{"export", "--unknown"}, nil},
	} {
		var got []bool
		var stdout, stderr bytes.Buffer
		status := Run(test.args, &stdout, &stderr, "dev", Operations{Export: func(rulesets, actions bool) error { got = []bool{rulesets, actions}; return nil }})
		if !reflect.DeepEqual(got, test.want) || (status == 0) != (test.want != nil) || stdout.Len() != 0 {
			t.Fatal(test.args, status, got, stderr.String())
		}
	}
}
