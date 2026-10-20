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
		status := Run(test.args, &stdout, &stderr, "dev", Operations{Export: func(options ExportOptions) error { got = []bool{options.Rulesets, options.Actions}; return nil }})
		if !reflect.DeepEqual(got, test.want) || (status == 0) != (test.want != nil) || stdout.Len() != 0 {
			t.Fatal(test.args, status, got, stderr.String())
		}
	}
}

func TestWorkflowExportOptionsAndDuplicateRejection(t *testing.T) {
	for _, args := range [][]string{{"export", "--workflow-permissions"}, {"export", "--rulesets", "--actions", "--workflow-permissions"}, {"export", "--workflow-permissions", "--actions", "--rulesets"}, {"export", "--workflow-permissions", "--workflow-permissions"}} {
		calls := 0
		var got ExportOptions
		var stdout, stderr bytes.Buffer
		status := Run(args, &stdout, &stderr, "dev", Operations{Export: func(options ExportOptions) error { calls++; got = options; return nil }})
		if len(args) == 3 {
			if status == 0 || calls != 0 {
				t.Fatal(args, status, calls)
			}
			continue
		}
		if status != 0 || calls != 1 || !got.WorkflowPermissions || ((len(args) == 4) != (got.Rulesets && got.Actions)) {
			t.Fatal(args, status, got, stderr.String())
		}
	}
}
