package githubactions

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"go.yaml.in/yaml/v4"
)

func TestGoCachePreservesChecksAndTrustBoundary(t *testing.T) {
	restore, _ := readReleaseAction(t, "restore-go-cache")
	save, _ := readReleaseAction(t, "save-go-cache")
	guard := "github.event_name == 'pull_request' || (github.event_name == 'push' && github.ref == 'refs/heads/main')"
	if len(restore.Runs.Steps) != 2 || len(save.Runs.Steps) != 1 {
		t.Fatal("cache actions must only resolve/restore and explicitly save Go cache data")
	}
	identity, restored, saved := restore.Runs.Steps[0], restore.Runs.Steps[1], save.Runs.Steps[0]
	if identity.ID != "go-cache" || identity.If != guard || restored.ID != "restore" || restored.If != guard || !restored.Continue ||
		!strings.HasPrefix(restored.Uses, "actions/cache/restore@") || restored.With["fail-on-cache-miss"] == "true" ||
		restored.With["lookup-only"] == "true" || restored.With["enableCrossOsArchive"] == "true" {
		t.Fatal("restore must tolerate unavailability without widening event/environment scope")
	}
	if restored.With["path"] != "${{ steps.go-cache.outputs.build-path }}\n${{ steps.go-cache.outputs.module-path }}\n" {
		t.Fatal("only resolved Go build/module cache paths may be cached")
	}
	prefix := strings.TrimSpace(restored.With["restore-keys"])
	if prefix == "" || strings.Contains(prefix, "\n") || restored.With["key"] != prefix+"${{ github.sha }}" {
		t.Fatal("restore must stay within one compatible family and save per revision")
	}
	for _, required := range []string{"go-ci-v1-", "runner.os", "runner.arch", "ubuntu-slim", "steps.go-cache.outputs.environment", "'go.mod'", "'go.sum'", "'mise.toml'", "'.github/actions/restore-go-cache/action.yml'", "'.github/actions/save-go-cache/action.yml'", "'.github/actions/lint/action.yml'", "'.github/actions/unit-test/action.yml'", "'.github/actions/build/action.yml'"} {
		if !strings.Contains(prefix, required) {
			t.Fatalf("cache family missing %s", required)
		}
	}
	for name, value := range map[string]string{
		"build-path":        "${{ steps.go-cache.outputs.build-path }}",
		"module-path":       "${{ steps.go-cache.outputs.module-path }}",
		"cache-hit":         "${{ steps.restore.outputs.cache-hit }}",
		"cache-primary-key": "${{ steps.restore.outputs.cache-primary-key }}",
	} {
		output := restore.Outputs[name]
		var definition struct{ Value string }
		if err := output.Decode(&definition); err != nil || definition.Value != value {
			t.Fatalf("restore output %s must retain the resolved value: %q %v", name, definition.Value, err)
		}
	}
	if !strings.HasPrefix(saved.Uses, "actions/cache/save@") || !saved.Continue ||
		saved.If != "("+guard+") && inputs.cache-hit != 'true' && inputs.cache-primary-key != ''" ||
		saved.With["key"] != "${{ inputs.cache-primary-key }}" ||
		saved.With["path"] != "${{ inputs.build-path }}\n${{ inputs.module-path }}\n" {
		t.Fatal("save must use explicit resolved paths/key, skip exact hits and tolerate unavailability")
	}

	// Cache hits cannot make source checks optional or reuse cached test results.
	checks := map[string][]string{
		"lint":      {"go mod download", "go mod verify", "go mod tidy -diff", "go vet ./..."},
		"unit-test": {"go test -race -count=1 -vet=off ./..."},
		"build":     {"go build -trimpath"},
	}
	for name, commands := range checks {
		action, _ := readReleaseAction(t, name)
		for _, command := range commands {
			found := false
			for _, s := range action.Runs.Steps {
				if strings.Contains(s.Run, command) {
					found = true
					if s.If != "" || s.Continue {
						t.Fatalf("source check became optional: %s", command)
					}
				}
			}
			if !found {
				t.Errorf("missing unconditional source check: %s", command)
			}
		}
	}
	ci, _ := readReleaseAction(t, "ci-commit")
	order := []string{"setup-tools", "restore-go-cache", "lint", "unit-test", "build", "save-go-cache"}
	position := 0
	for _, s := range ci.Runs.Steps {
		if !strings.HasPrefix(s.Uses, "./.github/actions/") {
			continue
		}
		if position >= len(order) || s.Uses != "./.github/actions/"+order[position] || s.If != "" || s.Continue {
			t.Fatalf("CI must restore, check, build and save without a cache-hit condition: %+v", s)
		}
		if order[position] == "save-go-cache" {
			for _, name := range []string{"build-path", "module-path", "cache-hit", "cache-primary-key"} {
				if s.With[name] != "${{ steps.go-cache.outputs."+name+" }}" {
					t.Fatalf("cache save must receive the restore output %s", name)
				}
			}
		}
		position++
	}
	if position != len(order) {
		t.Fatal("CI omitted a source check or explicit cache step")
	}
	release, _ := readReleaseAction(t, "release-build")
	for _, s := range release.Runs.Steps {
		if strings.Contains(s.Uses, "go-cache") || strings.HasPrefix(s.Uses, "actions/cache/") {
			t.Fatal("release builds must not opt in to the unprivileged CI cache")
		}
	}
}

func TestGoCacheEnvironmentFingerprint(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(root(), ".github/actions/restore-go-cache/action.yml"))
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
		"sha256sum": `exec shasum -a 256 "$@"`,
		"cat":       `printf '%s\n' "$DISTRO"`,
		"gcc":       `printf '%s\n' "$COMPILER"`,
		"ldd":       `printf '%s\n' "$LIBC"`,
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
