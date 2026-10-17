//go:build acceptance

package acceptance

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestExportExistingFile(t *testing.T) {
	for _, test := range []struct {
		name, body, wantError string
		status                int
	}{
		{name: "replace", body: planResponse, status: 200},
		{name: "api-failure", body: `{"message":"Fixture access denied"}`, status: 403, wantError: "403"},
		{name: "malformed-response", body: `{`, status: 200, wantError: "read repository settings"},
		{name: "incomplete-response", body: `{}`, status: 200, wantError: "missing description"},
	} {
		t.Run(test.name, func(t *testing.T) {
			fixture, err := startAPIFixture(test.body, test.status)
			if err != nil {
				t.Fatal(err)
			}
			defer fixture.close()
			env := exportFixtureEnv(fixture.cfg)
			if err := setupOrdinaryOrigin(fixture.work, env); err != nil {
				t.Fatal(err)
			}
			original := "old configuration with trailing data that must be replaced\n"
			if err := writeSettingsFile(fixture.work, original); err != nil {
				t.Fatal(err)
			}
			// Export has no stdout output, including when stdout cannot be written.
			result, err := executeCLI(acceptanceBinary(t), fixture.work, fixture.root, env, []string{"export"}, true)
			if err != nil {
				t.Fatal(err)
			}
			want := original
			if test.wantError == "" {
				if result.runErr != nil || result.stderr.Len() != 0 {
					t.Fatalf("execution=%v stderr=%q", result.runErr, result.stderr.String())
				}
				want = "schemaVersion: 1\nsettings:\n    repository:\n        description: old\n        homepage: old-url\n        has_issues: true\n        has_projects: true\n        has_wiki: true\n    topics:\n        names: []\n"
			} else {
				exit, ok := result.runErr.(*exec.ExitError)
				if !ok || exit.ExitCode() != 1 || !strings.Contains(result.stderr.String(), test.wantError) {
					t.Fatalf("execution=%v stderr=%q", result.runErr, result.stderr.String())
				}
			}
			if err := checkSettingsUnchanged(fixture.work, want); err != nil {
				t.Fatal(err)
			}
			if err := fixture.checkRequests(false); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestExportLinkedWorktree(t *testing.T) {
	fixture, err := startAPIFixture(planResponse, 200)
	if err != nil {
		t.Fatal(err)
	}
	defer fixture.close()
	env := exportFixtureEnv(fixture.cfg)
	if err := setupOrdinaryOrigin(fixture.work, env); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"-c", "user.name=Fixture Maintainer", "-c", "user.email=fixture@example.com", "-c", "commit.gpgsign=false", "commit", "--allow-empty", "--quiet", "-m", "fixture"},
		{"config", "extensions.worktreeConfig", "true"},
		{"config", "--local", "--unset-all", "remote.origin.url"},
		{"config", "--worktree", "remote.origin.url", "https://github.com/owner/other.git"},
	} {
		if err := runGit(fixture.work, env, args...); err != nil {
			t.Fatal(err)
		}
	}
	base := "base checkout configuration\n"
	if err := writeSettingsFile(fixture.work, base); err != nil {
		t.Fatal(err)
	}
	worktree := filepath.Join(fixture.root, "linked")
	if err := runGit(fixture.work, env, "worktree", "add", "--quiet", "--detach", worktree, "HEAD"); err != nil {
		t.Fatal(err)
	}
	if err := runGit(worktree, env, "config", "--worktree", "remote.origin.url", "https://github.com/owner/current.git"); err != nil {
		t.Fatal(err)
	}
	subdir := filepath.Join(worktree, "nested")
	if err := os.Mkdir(subdir, 0700); err != nil {
		t.Fatal(err)
	}
	result, err := executeCLI(acceptanceBinary(t), subdir, fixture.root, env, []string{"export"}, false)
	if err != nil || result.runErr != nil || result.stdout.Len() != 0 || result.stderr.Len() != 0 {
		t.Fatalf("execution=%v result=%+v", err, result)
	}
	data, err := os.ReadFile(filepath.Join(worktree, ".github", "settings.yml"))
	if err != nil || !strings.Contains(string(data), "has_issues: true") {
		t.Fatalf("worktree configuration=%q error=%v", data, err)
	}
	if err := checkSettingsUnchanged(fixture.work, base); err != nil {
		t.Fatal(err)
	}
	if err := fixture.checkRequests(false); err != nil {
		t.Fatal(err)
	}
}
