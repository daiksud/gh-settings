package workspace

import (
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

// The matrix groups descendants only where the documented grants match. New
// public codec fields must receive a reviewed row, not disappear from coverage.
func permissionControlPaths(typ reflect.Type, prefix string) []string {
	for typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	if typ == reflect.TypeFor[uimodel.PagesCustomDomain]() {
		return []string{prefix}
	}
	if typ.Kind() == reflect.Slice {
		return permissionControlPaths(typ.Elem(), prefix+"[]")
	}
	if typ.Kind() != reflect.Struct {
		return []string{prefix}
	}
	var result []string
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		result = append(result, permissionControlPaths(field.Type, prefix+"."+uiControlName(field.Name))...)
	}
	return result
}

func TestSettingsPermissionMatrixCoversPublicCodec(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "docs", "settings", "permissions.md"))
	if err != nil {
		t.Fatal(err)
	}
	entries := regexp.MustCompile(`<!-- permission-path (v1|v0): ([a-zA-Z0-9_.\[\]-]+) -->`).FindAllStringSubmatch(string(data), -1)
	if len(entries) == 0 {
		t.Fatal("permission matrix has no path coverage records")
	}
	paths := permissionControlPaths(reflect.TypeFor[uiSettingsDocument](), "repository.settings")
	paths = append(paths, permissionControlPaths(reflect.TypeFor[uiAboutDocument](), "repository.about")...)
	paths = append(paths, permissionControlPaths(reflect.TypeFor[uimodel.Label](), "repository.labels[]")...)
	matched := make([]bool, len(entries))
	for _, path := range paths {
		covered := false
		legacy := strings.HasPrefix(path, "repository.settings.autolinks[") || strings.HasPrefix(path, "repository.settings.branch_protection_rules[")
		for i, e := range entries {
			prefix := e[2]
			if path != prefix && !strings.HasPrefix(path, prefix+".") && !strings.HasPrefix(path, prefix+"[]") {
				continue
			}
			covered = true
			matched[i] = true
			if legacy != (e[1] == "v0") {
				t.Errorf("%s has incorrect v1/v0 permission classification", path)
			}
		}
		if !covered {
			t.Errorf("missing permission row for %s", path)
		}
	}
	for i, yes := range matched {
		if !yes {
			t.Errorf("stale permission row path %s", entries[i][2])
		}
	}
	t.Logf("covered %d public codec leaf paths with %d explicit permission groups", len(paths), len(entries))
}
