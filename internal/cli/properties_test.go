package cli

import (
	"bytes"
	"reflect"
	"testing"
)

func TestPropertiesExportIsExplicitAndComposable(t *testing.T) {
	for _, tc := range []struct {
		args       []string
		valid      bool
		properties bool
		names      []string
	}{
		{args: []string{"export"}, valid: true},
		{args: []string{"export", "--custom-properties"}, valid: true, properties: true},
		{args: []string{"export", "--custom-properties", "--archive", "--actions"}, valid: true, properties: true},
		{args: []string{"export", "--custom-property", "service", "--custom-property", "regions", "--archive"}, valid: true, names: []string{"service", "regions"}},
		{args: []string{"export", "--custom-properties", "--custom-properties"}},
		{args: []string{"export", "--custom-properties", "--custom-property", "service"}},
		{args: []string{"export", "--custom-property", "service", "--custom-properties"}},
		{args: []string{"export", "--custom-property", "service", "--custom-property", "service"}},
		{args: []string{"export", "--custom-property"}},
		{args: []string{"export", "--custom-property", "bad.name"}},
		{args: []string{"plan", "--custom-properties"}},
		{args: []string{"plan", "--custom-property", "service"}},
	} {
		var out, errout bytes.Buffer
		called := false
		code := Run(tc.args, &out, &errout, "dev", Operations{Export: func(o ExportOptions) error {
			called = true
			if o.Properties != tc.properties || !reflect.DeepEqual(o.PropertyNames, tc.names) {
				t.Fatal(o)
			}
			return nil
		}})
		if (code == 0) != tc.valid || called != tc.valid {
			t.Fatal(tc, code, errout.String())
		}
	}
}
