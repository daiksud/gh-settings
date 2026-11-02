package githubactions

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"go.yaml.in/yaml/v4"
)

func TestGoCachePreservesCommitChecksAndTrustBoundary(t *testing.T) {
	read := func(name string) ([]step, string, map[string]struct{ Default string }) {
		t.Helper()
		data, err := os.ReadFile(filepath.Join(root(), ".github/actions", name, "action.yml"))
		if err != nil {
			t.Fatal(err)
		}
		var action struct {
			Inputs map[string]struct{ Default string } `yaml:"inputs"`
			Runs   struct{ Steps []step }              `yaml:"runs"`
		}
		if err := yaml.Unmarshal(data, &action); err != nil {
			t.Fatal(err)
		}
		return action.Runs.Steps, string(data), action.Inputs
	}
	steps, _, inputs := read("commit-stage")
	if inputs["cache-go"].Default != "false" {
		t.Fatal("cache must be opt-in and never decide whether checks execute")
	}
	guard := "inputs.cache-go == 'true' && (github.event_name == 'pull_request' || (github.event_name == 'push' && github.ref == 'refs/heads/main'))"
	cacheCount, saveCount := 0, 0
	checks := map[string]bool{
		"go mod download": false, "go mod verify": false, "go mod tidy -diff": false,
		"go vet ./...": false, "go test -race -count=1 -vet=off ./...": false,
		"go build -trimpath": false,
	}
	for index, s := range steps {
		if strings.HasPrefix(s.Uses, "actions/cache/save@") {
			saveCount++
			if index != len(steps)-1 || !s.Continue || s.If != guard+" && steps.go-cache-restore.outputs.cache-hit != 'true' && steps.go-cache-restore.outputs.cache-primary-key != ''" || s.With["key"] != "${{ steps.go-cache-restore.outputs.cache-primary-key }}" {
				t.Fatal("save must be explicit after all checks, reuse the resolved key and tolerate unavailability")
			}
			if s.With["path"] != "${{ steps.go-cache.outputs.build-path }}\n${{ steps.go-cache.outputs.module-path }}\n" {
				t.Fatal("save must use the same in-scope Go cache paths as restore")
			}
			continue
		}
		if strings.HasPrefix(s.Uses, "actions/cache/restore@") {
			cacheCount++
			if s.If != guard || !s.Continue || s.With["fail-on-cache-miss"] == "true" || s.With["lookup-only"] == "true" || s.With["enableCrossOsArchive"] == "true" {
				t.Fatal("cache must tolerate unavailability without widening event/environment scope")
			}
			if s.With["path"] != "${{ steps.go-cache.outputs.build-path }}\n${{ steps.go-cache.outputs.module-path }}\n" {
				t.Fatal("only resolved Go build/module cache paths may be cached")
			}
			prefix := strings.TrimSpace(s.With["restore-keys"])
			if prefix == "" || strings.Contains(prefix, "\n") || s.With["key"] != prefix+"${{ github.sha }}" {
				t.Fatal("restore must stay within one compatible family and save per revision")
			}
			for _, required := range []string{"go-ci-v1-", "runner.os", "runner.arch", "ubuntu-slim", "steps.go-cache.outputs.environment", "'go.mod'", "'go.sum'", "'mise.toml'", "'.github/actions/commit-stage/action.yml'"} {
				if !strings.Contains(prefix, required) {
					t.Fatalf("cache family missing %s", required)
				}
			}
			continue
		}
		if s.ID == "go-cache" {
			if s.If != guard {
				t.Fatal("cache identity must use the same event guard")
			}
			continue
		}
		if s.If != "" || s.Continue {
			t.Fatalf("commit check became optional: %s", s.Name)
		}
		for command := range checks {
			if strings.Contains(s.Run, command) {
				checks[command] = true
			}
		}
	}
	if cacheCount != 1 || saveCount != 1 {
		t.Fatalf("want explicit restore/save, got %d/%d", cacheCount, saveCount)
	}
	for command, found := range checks {
		if !found {
			t.Errorf("missing unconditional check: %s", command)
		}
	}
	for name, want := range map[string]string{"ci-commit": "true", "release-build": ""} {
		caller, _, _ := read(name)
		for _, s := range caller {
			if s.Uses == "./.github/actions/commit-stage" && s.With["cache-go"] != want {
				t.Fatalf("%s cache opt-in = %q, want %q", name, s.With["cache-go"], want)
			}
		}
	}
}

func TestGoCacheEnvironmentFingerprint(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(root(), ".github/actions/commit-stage/action.yml"))
	if err != nil {
		t.Fatal(err)
	}
	var action struct{ Runs struct{ Steps []step } }
	if err := yaml.Unmarshal(data, &action); err != nil {
		t.Fatal(err)
	}
	var script string
	for _, s := range action.Runs.Steps {
		if s.ID == "go-cache" {
			script = s.Run
		}
	}
	if script == "" {
		t.Fatal("missing cache identity step")
	}
	dir := t.TempDir()
	fixtures := map[string]string{
		"cat": `printf '%s\n' "$DISTRO"`,
		"gcc": `printf '%s\n' "$COMPILER"`,
		"ldd": `printf '%s\n' "$LIBC"`,
		"go": `case "$*" in
"env GOCACHE") echo /fixture/build ;;
"env GOMODCACHE") echo /fixture/modules ;;
"version") printf '%s\n' "$GO_VERSION" ;;
"env "*) printf '%s\n' "$GO_ENV" ;;
*) exit 1 ;;
esac`,
	}
	for name, body := range fixtures {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("#!/bin/bash\n"+body+"\n"), 0755); err != nil {
			t.Fatal(err)
		}
	}
	run := func(changed string) string {
		t.Helper()
		out := filepath.Join(t.TempDir(), "output")
		env := map[string]string{"DISTRO": "ubuntu-24.04", "COMPILER": "gcc-13", "LIBC": "glibc-2.39", "GO_VERSION": "go1.26.7", "GO_ENV": "linux amd64 v1 cgo=1"}
		if changed != "" {
			env[changed] += "-changed"
		}
		cmd := exec.Command("bash", "-euo", "pipefail", "-c", script)
		cmd.Env = append(os.Environ(), "PATH="+dir+":"+os.Getenv("PATH"), "GITHUB_OUTPUT="+out)
		for k, v := range env {
			cmd.Env = append(cmd.Env, k+"="+v)
		}
		if result, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("identity: %v: %s", err, result)
		}
		result, err := os.ReadFile(out)
		if err != nil {
			t.Fatal(err)
		}
		text := string(result)
		if !strings.HasPrefix(text, "build-path=/fixture/build\nmodule-path=/fixture/modules\nenvironment=") {
			t.Fatalf("unexpected outputs: %s", text)
		}
		return text
	}
	original := run("")
	if run("") != original {
		t.Fatal("same environment must have stable identity")
	}
	for _, changed := range []string{"DISTRO", "COMPILER", "LIBC", "GO_VERSION", "GO_ENV"} {
		if run(changed) == original {
			t.Errorf("%s change reused environment identity", changed)
		}
	}
}
