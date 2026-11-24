package githubactions

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.yaml.in/yaml/v4"
)

// Every native build uses the module toolchain; only Linux runs the Bun-based source checks.
func TestBuildJobsUseManifestToolchains(t *testing.T) {
	for _, tc := range []struct {
		workflow string
		job      string
		usesBun  bool
	}{
		{workflow: "pr.yml", job: "verify", usesBun: true},
		{workflow: "candidate.yml", job: "screening-linux", usesBun: true},
		{workflow: "candidate.yml", job: "screening-macos"},
	} {
		t.Run(tc.workflow+"/"+tc.job, func(t *testing.T) {
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
			bunIndex, cacheIndex, buildIndex := -1, -1, -1
			for i, s := range steps {
				switch {
				case strings.HasPrefix(s.Uses, "jdx/mise-action@"):
					t.Fatal("CI must not install tools through mise")
				case strings.HasPrefix(s.Uses, "oven-sh/setup-bun@"):
					if bunIndex != -1 || s.With["bun-version-file"] != "package.json" {
						t.Fatal("Bun must use package.json")
					}
					bunIndex = i
				case strings.HasPrefix(s.Uses, "actions/setup-go@"):
					if cacheIndex != -1 || s.With["go-version"] != "" ||
						s.With["go-version-file"] != "go.mod" ||
						s.With["cache-dependency-path"] != "go.sum" ||
						s.With["cache"] == "false" {
						t.Fatalf("setup-go must install Go from go.mod and enable caches: %+v", s)
					}
					cacheIndex = i
				case strings.Contains(s.Uses, "go-cache") || strings.HasPrefix(s.Uses, "actions/cache/"):
					t.Fatalf("custom Go cache management must not return: %+v", s)
				case s.Uses == "./.github/actions/build" || s.Uses == "./.github/actions/screening":
					buildIndex = i
					if s.If != "" || s.Continue {
						t.Fatalf("build must not be skipped on a cache hit: %+v", s)
					}
				}
			}
			if (bunIndex >= 0) != tc.usesBun || cacheIndex <= bunIndex || buildIndex <= cacheIndex {
				t.Fatalf("install Bun, resolve Go, then build: bun=%d cache=%d build=%d",
					bunIndex, cacheIndex, buildIndex)
			}
		})
	}
}
