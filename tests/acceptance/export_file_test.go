//go:build acceptance

package acceptance

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"testing"
	"time"

	"go.yaml.in/yaml/v4"
)

func TestExportSavesAtCheckoutRoot(t *testing.T) {
	fixture, err := startAPIFixture(`{"description":"Example repository","homepage":"","has_issues":true,"has_projects":false,"has_wiki":false}`, 200)
	if err != nil {
		t.Fatal(err)
	}
	defer fixture.close()
	env := exportFixtureEnv(fixture.cfg)
	if err := setupOrdinaryOrigin(fixture.work, env); err != nil {
		t.Fatal(err)
	}
	subdir := filepath.Join(fixture.work, "nested", "directory")
	if err := os.MkdirAll(subdir, 0700); err != nil {
		t.Fatal(err)
	}
	result, err := executeCLI(acceptanceBinary(t), subdir, fixture.root, env, []string{"export"}, false)
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(fixture.work, ".github", "settings.yml"))
	if err != nil {
		t.Fatalf("root configuration was not saved: %v", err)
	}
	var actual map[string]any
	if err := yaml.Unmarshal(data, &actual); err != nil {
		t.Fatal(err)
	}
	want := map[string]any{"schemaVersion": 1, "settings": map[string]any{"topics": map[string]any{"names": []any{}}, "repository": map[string]any{
		"description": "Example repository", "homepage": "", "has_issues": true, "has_projects": false, "has_wiki": false,
	}}}
	if !reflect.DeepEqual(actual, want) || result.runErr != nil || result.stdout.Len() != 0 || result.stderr.Len() != 0 {
		t.Fatalf("YAML=%#v execution=%v stdout=%q stderr=%q", actual, result.runErr, result.stdout.String(), result.stderr.String())
	}
	if _, err := os.Lstat(filepath.Join(subdir, ".github")); !os.IsNotExist(err) {
		t.Fatalf("subdirectory destination exists: %v", err)
	}
	if err := fixture.checkRequests(false); err != nil {
		t.Fatal(err)
	}
}

func TestExportRejectsUnsafeDestination(t *testing.T) {
	for _, name := range []string{"github-symlink", "github-dangling-symlink", "file-symlink", "file-dangling-symlink", "directory", "fifo"} {
		t.Run(name, func(t *testing.T) {
			fixture, err := startAPIFixture(planResponse, 200)
			if err != nil {
				t.Fatal(err)
			}
			defer fixture.close()
			env := exportFixtureEnv(fixture.cfg)
			if err := setupOrdinaryOrigin(fixture.work, env); err != nil {
				t.Fatal(err)
			}
			outside := filepath.Join(fixture.root, "outside")
			if err := os.Mkdir(outside, 0700); err != nil {
				t.Fatal(err)
			}
			sentinel := filepath.Join(outside, "settings.yml")
			if err := os.WriteFile(sentinel, []byte("external configuration\n"), 0600); err != nil {
				t.Fatal(err)
			}
			github := filepath.Join(fixture.work, ".github")
			path := filepath.Join(github, "settings.yml")
			link, target := "", ""
			if strings.HasPrefix(name, "github-") {
				link, target = github, outside
				if strings.Contains(name, "dangling") {
					target = filepath.Join(outside, "missing")
				}
			} else {
				if err := os.Mkdir(github, 0700); err != nil {
					t.Fatal(err)
				}
				switch name {
				case "file-symlink":
					link, target = path, sentinel
				case "file-dangling-symlink":
					link, target = path, filepath.Join(outside, "missing")
				case "directory":
					err = os.Mkdir(path, 0700)
				case "fifo":
					err = syscall.Mkfifo(path, 0600)
				}
				if err != nil {
					t.Fatal(err)
				}
			}
			if link != "" {
				if err := os.Symlink(target, link); err != nil {
					t.Fatal(err)
				}
			}
			observed := path
			if link != "" {
				observed = link
			}
			before, err := os.Lstat(observed)
			if err != nil {
				t.Fatal(err)
			}
			result, err := executeCLI(acceptanceBinary(t), fixture.work, fixture.root, env, []string{"export"}, false)
			if err != nil {
				t.Fatal(err)
			}
			exit, ok := result.runErr.(*exec.ExitError)
			if !ok || exit.ExitCode() != 1 || result.stdout.Len() != 0 || !strings.Contains(result.stderr.String(), ".github") {
				t.Fatalf("execution=%v stdout=%q stderr=%q", result.runErr, result.stdout.String(), result.stderr.String())
			}
			after, err := os.Lstat(observed)
			if err != nil || !os.SameFile(before, after) || before.Mode() != after.Mode() {
				t.Fatalf("destination changed: before=%v after=%v error=%v", before, after, err)
			}
			if link != "" {
				if got, err := os.Readlink(link); err != nil || got != target {
					t.Fatalf("symlink changed: target=%q error=%v", got, err)
				}
			}
			data, err := os.ReadFile(sentinel)
			if err != nil || string(data) != "external configuration\n" {
				t.Fatalf("external target changed: %q error=%v", data, err)
			}
			entries, err := os.ReadDir(outside)
			if err != nil || len(entries) != 1 || entries[0].Name() != "settings.yml" {
				t.Fatalf("external files changed: %v error=%v", entries, err)
			}
		})
	}
}

func TestExportReportsWriteFailure(t *testing.T) {
	body, err := json.Marshal(map[string]any{"description": strings.Repeat("x", 8192), "homepage": "", "has_issues": true, "has_projects": false, "has_wiki": false})
	if err != nil {
		t.Fatal(err)
	}
	fixture, err := startAPIFixture(string(body), 200)
	if err != nil {
		t.Fatal(err)
	}
	defer fixture.close()
	env := exportFixtureEnv(fixture.cfg)
	if err := setupOrdinaryOrigin(fixture.work, env); err != nil {
		t.Fatal(err)
	}
	if err := writeSettingsFile(fixture.work, "existing configuration\n"); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	// Limit only the child process. Ignore SIGXFSZ so the partial write reports an error.
	cmd := exec.CommandContext(ctx, "/bin/sh", "-c", `trap '' XFSZ; ulimit -f 1; exec "$@"`, "sh", acceptanceBinary(t), "export")
	cmd.Dir, cmd.Env = fixture.work, env
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	runErr := cmd.Run()
	exit, ok := runErr.(*exec.ExitError)
	if ctx.Err() != nil || !ok || exit.ExitCode() != 1 || stdout.Len() != 0 || !strings.Contains(stderr.String(), "write .github/settings.yml") {
		t.Fatalf("execution=%v timeout=%v stdout=%q stderr=%q", runErr, ctx.Err(), stdout.String(), stderr.String())
	}
	if err := fixture.checkRequests(false); err != nil {
		t.Fatal(err)
	}
}
