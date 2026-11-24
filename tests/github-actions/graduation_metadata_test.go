package githubactions

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"go.yaml.in/yaml/v4"
)

// Execute the source-controlled composite resolver against an isolated release
// catalog and Git history. The gh stand-in accepts only the expected read call.
func TestReleaseChoiceUsesOneFrozenMetadataResolver(t *testing.T) {
	for _, name := range []string{"push.yml", "release.yml"} {
		data, err := os.ReadFile(filepath.Join(root(), ".github/workflows", name))
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
		var checkout, metadata step
		for _, s := range workflow.Jobs["resolve"].Steps {
			if strings.HasPrefix(s.Uses, "actions/checkout@") {
				checkout = s
			}
			if s.ID == "metadata" {
				metadata = s
			}
		}
		if checkout.With["ref"] != "${{ github.sha }}" || checkout.With["fetch-depth"] != "0" ||
			metadata.Uses != "./.github/actions/release-metadata" || metadata.With["source-sha"] != "${{ github.sha }}" || metadata.If != "" || metadata.Continue {
			t.Fatalf("%s does not resolve metadata from frozen source: %+v %+v", name, checkout, metadata)
		}
		wantChoice := ""
		if name == "release.yml" {
			wantChoice = "${{ inputs.promote_to_v1 }}"
		}
		if metadata.With["promote-to-v1"] != wantChoice {
			t.Fatalf("%s has unexpected promotion selection: %+v", name, metadata.With)
		}
	}
	fixture := t.TempDir()
	git := func(args ...string) string {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = fixture
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	git("init", "--quiet")
	git("config", "user.name", "Fixture")
	git("config", "user.email", "fixture@example.invalid")
	git("config", "commit.gpgsign", "false")
	scriptDir := filepath.Join(fixture, ".github", "scripts")
	if err := os.MkdirAll(scriptDir, 0700); err != nil {
		t.Fatal(err)
	}
	calculator, err := os.ReadFile(filepath.Join(root(), ".github/scripts/release-version.py"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(scriptDir, "release-version.py"), calculator, 0600); err != nil {
		t.Fatal(err)
	}
	git("add", ".")
	git("commit", "--quiet", "-m", "chore: baseline")
	baseline := git("rev-parse", "HEAD")
	git("tag", "v0.3.0")
	if err := os.WriteFile(filepath.Join(fixture, "change"), []byte("change"), 0600); err != nil {
		t.Fatal(err)
	}
	git("add", "change")
	git("commit", "--quiet", "-m", "fix: correction")
	source := git("rev-parse", "HEAD")
	git("tag", "v0.4.0")
	bin := t.TempDir()
	stub := "#!/bin/sh\n[ \"$*\" = 'api repos/owner/repo/releases?per_page=100 --paginate --slurp' ] || exit 1\ncat \"$CATALOG_FIXTURE\"\n"
	if err := os.WriteFile(filepath.Join(bin, "gh"), []byte(stub), 0700); err != nil {
		t.Fatal(err)
	}
	action, _ := readReleaseAction(t, "release-metadata")
	if len(action.Runs.Steps) != 1 || action.Runs.Steps[0].ID != "resolve" || action.Runs.Steps[0].Env["SOURCE_SHA"] != "${{ inputs.source-sha }}" {
		t.Fatal("unexpected metadata action structure")
	}
	baselineRelease := map[string]any{"id": 1, "tag_name": "v0.3.0", "draft": false, "prerelease": false, "published_at": "2026-10-01T00:00:00Z"}
	marker, err := json.Marshal(map[string]string{"version": "v0.4.0", "commit": source, "macos_sha256": strings.Repeat("a", 64), "linux_sha256": strings.Repeat("b", 64), "checksums_sha256": strings.Repeat("c", 64)})
	if err != nil {
		t.Fatal(err)
	}
	retryRelease := map[string]any{"id": 2, "tag_name": "v0.4.0", "draft": false, "prerelease": false, "published_at": "2026-10-02T00:00:00Z", "body": "<!-- gh-settings-release: " + string(marker) + " -->\nNotes"}
	for _, tc := range []struct {
		caller, choice, version string
		retry, valid            bool
	}{
		{"push", "false", "v0.3.1", false, true},
		{"release-default", "false", "v0.3.1", false, true},
		{"release-promotion", "true", "v1.0.1", false, true},
		{"push-retry", "false", "v0.4.0", true, true},
		{"release-default-retry", "false", "v0.4.0", true, true},
		{"release-promotion-v0-retry", "true", "", true, false},
		{"invalid-choice", "yes", "", false, false},
	} {
		t.Run(tc.caller, func(t *testing.T) {
			runnerTemp := t.TempDir()
			catalogFile := filepath.Join(runnerTemp, "catalog.json")
			releases := []map[string]any{baselineRelease}
			if tc.retry {
				releases = append(releases, retryRelease)
			}
			catalog, err := json.Marshal([][]map[string]any{releases})
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(catalogFile, catalog, 0600); err != nil {
				t.Fatal(err)
			}
			outputFile := filepath.Join(runnerTemp, "outputs")
			cmd := exec.Command("bash", "--noprofile", "--norc", "-e", "-o", "pipefail", "-c", action.Runs.Steps[0].Run)
			cmd.Dir = fixture
			cmd.Env = append(os.Environ(), "PATH="+bin+string(os.PathListSeparator)+os.Getenv("PATH"), "GITHUB_REPOSITORY=owner/repo", "SOURCE_SHA="+source, "RUNNER_TEMP="+runnerTemp, "GITHUB_OUTPUT="+outputFile, "CATALOG_FIXTURE="+catalogFile, "PROMOTE_TO_V1="+tc.choice)
			out, err := cmd.CombinedOutput()
			if (err == nil) != tc.valid {
				t.Fatalf("metadata outcome: %v\n%s", err, out)
			}
			if !tc.valid {
				if _, err := os.Stat(outputFile); !os.IsNotExist(err) {
					t.Fatal("failed resolution must not publish metadata")
				}
				return
			}
			content, err := os.ReadFile(outputFile)
			if err != nil {
				t.Fatal(err)
			}
			want := fmt.Sprintf("version=%s\nprevious-tag=v0.3.0\nprevious-sha=%s\n", tc.version, baseline)
			if tc.retry {
				want = "version=v0.4.0\nprevious-tag=\nprevious-sha=\n"
			}
			if string(content) != want+"release-needed=true\n" {
				t.Fatalf("wrong frozen metadata: %s", content)
			}
		})
	}
}
