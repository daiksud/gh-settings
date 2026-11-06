package githubactions

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.yaml.in/yaml/v4"
)

// Go uses the standard setup-go installation and cache lifecycle.
// Lint, unit tests and build still execute regardless of cache hits.
func TestPRUsesSetupGoCache(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(root(), ".github/workflows/pr.yml"))
	if err != nil {
		t.Fatal(err)
	}
	var workflow struct {
		Jobs map[string]struct {
			Steps []step `yaml:"steps"`
		} `yaml:"jobs"`
	}
	if err := yaml.Unmarshal(data, &workflow); err != nil {
		t.Fatal(err)
	}
	steps := workflow.Jobs["screening"].Steps
	if len(steps) == 0 {
		t.Fatal("screening job must not be empty")
	}
	foundSetupGo := false
	for _, s := range steps {
		if strings.Contains(s.Uses, "restore-go-cache") ||
			strings.Contains(s.Uses, "save-go-cache") ||
			strings.HasPrefix(s.Uses, "actions/cache/") {
			t.Fatalf("custom Go cache steps must not be used: %+v", s)
		}
		if strings.HasPrefix(s.Uses, "actions/setup-go@") {
			if foundSetupGo || s.With["go-version-file"] != "go.mod" ||
				s.With["cache-dependency-path"] != "go.sum" ||
				s.With["cache"] == "false" || s.If != "" || s.Continue {
				t.Fatalf("Go setup/cache must use go.mod and standard caching: %+v", s)
			}
			foundSetupGo = true
		}
	}
	if !foundSetupGo {
		t.Fatal("screening must install and cache Go with actions/setup-go")
	}
	for _, check := range []string{
		"./.github/actions/lint",
		"./.github/actions/unit-test",
		"./.github/actions/build",
	} {
		found := false
		for _, s := range steps {
			if s.Uses == check && s.If == "" && !s.Continue {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("required check must run regardless of cache result: %s", check)
		}
	}
}
