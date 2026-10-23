package cli

import (
	"bytes"
	"testing"
)

func TestPagesExportIsExplicitAndComposable(t *testing.T) {
	for _, tc := range []struct {
		args  []string
		valid bool
		pages bool
	}{{[]string{"export"}, true, false}, {[]string{"export", "--pages"}, true, true}, {[]string{"export", "--pages", "--archive", "--actions"}, true, true}, {[]string{"export", "--pages", "--pages"}, false, false}, {[]string{"plan", "--pages"}, false, false}} {
		var out, errout bytes.Buffer
		called := false
		code := Run(tc.args, &out, &errout, "dev", Operations{Export: func(o ExportOptions) error {
			called = true
			if o.Pages != tc.pages {
				t.Fatal(o)
			}
			return nil
		}})
		if (code == 0) != tc.valid || called != tc.valid {
			t.Fatal(tc, code, errout.String())
		}
	}
}
