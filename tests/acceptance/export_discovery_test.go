//go:build acceptance

package acceptance

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestExportKeepsOriginAndDestinationTogether(t *testing.T) {
	fixture, err := startHTTPFixture(func(w http.ResponseWriter, r *http.Request) {
		description := ""
		if r.Method == http.MethodGet {
			switch r.URL.Path {
			case "/repos/owner/current":
				description = "original origin"
			case "/repos/owner/other":
				description = "replacement origin"
			}
		}
		if description == "" {
			http.Error(w, "unexpected fixture request", http.StatusBadRequest)
			return
		}
		fmt.Fprint(w, strings.Replace(planResponse, `"old"`, `"`+description+`"`, 1))
	}, false)
	if err != nil {
		t.Fatal(err)
	}
	defer fixture.close()
	env := exportFixtureEnv(fixture.cfg)
	if err := setupOrdinaryOrigin(fixture.work, env); err != nil {
		t.Fatal(err)
	}
	replacement := filepath.Join(fixture.root, "replacement")
	if err := os.Mkdir(replacement, 0700); err != nil {
		t.Fatal(err)
	}
	if err := setupOrdinaryOrigin(replacement, env); err != nil {
		t.Fatal(err)
	}
	if err := runGit(replacement, env, "remote", "set-url", "origin", "https://github.com/owner/other.git"); err != nil {
		t.Fatal(err)
	}
	for dir, contents := range map[string]string{fixture.work: "original sentinel\n", replacement: "replacement sentinel\n"} {
		if err := writeSettingsFile(dir, contents); err != nil {
			t.Fatal(err)
		}
	}
	realGit, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	wrappers := filepath.Join(fixture.root, "bin")
	if err := os.Mkdir(wrappers, 0700); err != nil {
		t.Fatal(err)
	}
	// The second root discovery is the confirmed old root-to-open boundary.
	// An anchored discovery still returns through this wrapper, after root opening.
	script := `#!/bin/sh
set -eu
case "$*" in
  *"rev-parse --show-toplevel")
    root=$("$FIXTURE_REAL_GIT" "$@")
    count=0
    if [ -f "$FIXTURE_COUNT" ]; then count=$(cat "$FIXTURE_COUNT"); fi
    count=$((count + 1))
    printf '%s\n' "$count" > "$FIXTURE_COUNT"
    if [ "$count" -eq 2 ]; then
      mv "$FIXTURE_WORK" "$FIXTURE_MOVED"
      mv "$FIXTURE_REPLACEMENT" "$FIXTURE_WORK"
      : > "$FIXTURE_MUTATED"
    fi
    printf '%s\n' "$root"
    ;;
  *) exec "$FIXTURE_REAL_GIT" "$@" ;;
esac
`
	if err := os.WriteFile(filepath.Join(wrappers, "git"), []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	moved := filepath.Join(fixture.root, "moved")
	mutated := filepath.Join(fixture.root, "mutated")
	env = append(env, "PATH="+wrappers+string(os.PathListSeparator)+os.Getenv("PATH"),
		"FIXTURE_REAL_GIT="+realGit, "FIXTURE_COUNT="+filepath.Join(fixture.root, "count"),
		"FIXTURE_WORK="+fixture.work, "FIXTURE_MOVED="+moved,
		"FIXTURE_REPLACEMENT="+replacement, "FIXTURE_MUTATED="+mutated)
	result, err := executeCLI(acceptanceBinary(t), fixture.work, fixture.root, env, []string{"export"}, false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(mutated); err != nil {
		t.Fatalf("Git wrapper did not replace the checkout: %v", err)
	}
	if result.runErr != nil || result.stdout.Len() != 0 || result.stderr.Len() != 0 {
		t.Errorf("execution=%v stdout=%q stderr=%q", result.runErr, result.stdout.String(), result.stderr.String())
	}
	if err := checkSettingsUnchanged(fixture.work, "replacement sentinel\n"); err != nil {
		t.Error(err)
	}
	data, err := os.ReadFile(filepath.Join(moved, ".github", "settings.yml"))
	if err != nil || !strings.Contains(string(data), "description: original origin\n") {
		t.Errorf("original checkout configuration=%q error=%v", data, err)
	}
	if err := fixture.checkRequests(false); err != nil {
		t.Fatal(err)
	}
}
