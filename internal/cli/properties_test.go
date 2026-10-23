package cli

import (
	"bytes"
	"testing"
)

func TestPropertiesExportIsExplicitAndComposable(t *testing.T) {
	for _, tc := range []struct {
		args       []string
		valid      bool
		properties bool
	}{{[]string{"export"}, true, false}, {[]string{"export", "--custom-properties"}, true, true}, {[]string{"export", "--custom-properties", "--archive", "--actions"}, true, true}, {[]string{"export", "--custom-properties", "--custom-properties"}, false, false}, {[]string{"plan", "--custom-properties"}, false, false}} {
		var out, errout bytes.Buffer
		called := false
		code := Run(tc.args, &out, &errout, "dev", Operations{Export: func(o ExportOptions) error {
			called = true
			if o.Properties != tc.properties {
				t.Fatal(o)
			}
			return nil
		}})
		if (code == 0) != tc.valid || called != tc.valid {
			t.Fatal(tc, code, errout.String())
		}
	}
}
