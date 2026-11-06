package githubactions

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.yaml.in/yaml/v4"
)

// All build jobs use mise to install tools, then reuse that Go installation
// with setup-go's standard module and compilation cache.
func TestBuildJobsUseMiseWithSetupGoCache(t *testing.T) {
	for _, tc := range []struct {
		workflow string
		job      string
	}{
		{workflow: "pr.yml", job: "screening"},
		{workflow: "ci-push.yml", job: "commit-stage"},
		{workflow: "release.yml", job: "commit-stage"},
	} {
		t.Run(tc.workflow, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join(root(), ".github/workflows", tc.workflow))
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

			steps := workflow.Jobs[tc.job].Steps
			miseIndex, cacheIndex, buildIndex := -1, -1, -1
			for i, s := range steps {
				switch {
				case strings.HasPrefix(s.Uses, "jdx/mise-action@"):
					if miseIndex != -1 || s.With["install_args"] != "" || s.With["cache"] == "false" {
						t.Fatalf("toolchain must be installed once by mise with cache enabled: %+v", s)
					}
					miseIndex = i
				case strings.HasPrefix(s.Uses, "actions/setup-go@"):
					if cacheIndex != -1 || s.With["go-version"] != "" ||
						s.With["go-version-file"] != "" ||
						s.With["cache-dependency-path"] != "go.sum" ||
						s.With["cache"] == "false" {
						t.Fatalf("setup-go must cache the Go already installed by mise: %+v", s)
					}
					cacheIndex = i
				case strings.Contains(s.Uses, "go-cache") || strings.HasPrefix(s.Uses, "actions/cache/"):
					t.Fatalf("custom Go cache management must not return: %+v", s)
				case s.Uses == "./.github/actions/build":
					buildIndex = i
					if s.If != "" || s.Continue {
						t.Fatalf("build must not be skipped on a cache hit: %+v", s)
					}
				}
			}
			if miseIndex < 0 || cacheIndex != miseIndex+1 || buildIndex <= cacheIndex {
				t.Fatalf("install tools, restore Go caches, then build: mise=%d cache=%d build=%d",
					miseIndex, cacheIndex, buildIndex)
			}
		})
	}
}
