//go:build acceptance

package acceptance

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"go.yaml.in/yaml/v4"
)

func assertExportRetains(t *testing.T, fixture *apiFixture, before string, path ...string) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(fixture.work, ".github/settings.yml"))
	if err != nil {
		t.Fatal(err)
	}
	get := func(raw []byte) any {
		var m map[string]any
		if err := yaml.Unmarshal(raw, &m); err != nil {
			t.Fatal(err)
		}
		var v any = m
		for _, key := range path {
			if v == nil {
				return nil
			}
			mapping, ok := v.(map[string]any)
			if !ok {
				t.Fatal("unexpected YAML path", path)
			}
			v = mapping[key]
		}
		return v
	}
	var includes func(any, any) bool
	includes = func(want, actual any) bool {
		switch w := want.(type) {
		case map[string]any:
			a, ok := actual.(map[string]any)
			if !ok {
				return false
			}
			for k, v := range w {
				if !includes(v, a[k]) {
					return false
				}
			}
			return true
		case []any:
			actuals, ok := actual.([]any)
			if !ok || len(actuals) != len(w) {
				return false
			}
			used := make([]bool, len(actuals))
			for _, value := range w {
				found := false
				for i, candidate := range actuals {
					if !used[i] && includes(value, candidate) {
						used[i] = true
						found = true
						break
					}
				}
				if !found {
					return false
				}
			}
			return true
		default:
			return reflect.DeepEqual(want, actual)
		}
	}
	if !includes(get([]byte(before)), get(data)) {
		t.Fatal("unavailable family intent changed", path)
	}
}
