package githubactions

import (
	"debug/buildinfo"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestUnitTestActionPreservesCleanBuildMetadata(t *testing.T) {
	// Exercise the Action's actual commands with importing test fixtures, then
	// inspect a real binary from the same clean Git checkout.
	action, _ := readReleaseAction(t, "unit-test")
	checkout := filepath.Join(t.TempDir(), "source")
	if err := os.MkdirAll(filepath.Join(checkout, ".github/scripts"), 0700); err != nil {
		t.Fatal(err)
	}
	pythonTest := `import unittest
import fixture_module
class CacheTest(unittest.TestCase):
    def test_import(self):
        self.assertEqual(fixture_module.VALUE, 42)
`
	for name, content := range map[string]string{
		"go.mod":                            "module fixture.invalid/cleanbuild\n\ngo 1.26.7\n",
		"main.go":                           "package main\nfunc main() {}\n",
		".github/scripts/fixture_module.py": "VALUE = 42\n",
		".github/scripts/test_release_version.py": pythonTest,
		".github/scripts/test_release_ci.py":      pythonTest,
	} {
		if err := os.WriteFile(filepath.Join(checkout, name), []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	run := func(name string, args ...string) []byte {
		t.Helper()
		command := exec.Command(name, args...)
		command.Dir = checkout
		command.Env = append(os.Environ(), "GOWORK=off", "GOTOOLCHAIN=local")
		output, err := command.CombinedOutput()
		if err != nil {
			t.Fatalf("%s %v: %v\n%s", name, args, err, output)
		}
		return output
	}
	run("git", "init", "--quiet")
	run("git", "config", "core.excludesFile", os.DevNull)
	run("git", "add", ".")
	run("git", "-c", "commit.gpgsign=false", "-c", "core.hooksPath="+os.DevNull,
		"-c", "user.name=Fixture", "-c", "user.email=fixture@example.invalid",
		"commit", "--quiet", "-m", "fixture")
	revision := strings.TrimSpace(string(run("git", "rev-parse", "HEAD")))
	pythonSteps := 0
	for _, s := range action.Runs.Steps {
		if strings.Contains(s.Run, "python3") && strings.Contains(s.Run, "unittest") {
			run("bash", "-c", s.Run)
			pythonSteps++
		}
	}
	if pythonSteps != 2 {
		t.Fatalf("expected both Python test suites, got %d", pythonSteps)
	}
	if status := run("git", "status", "--porcelain", "--untracked-files=all"); len(status) != 0 {
		t.Fatalf("Python test steps dirtied the build checkout:\n%s", status)
	}
	binary := filepath.Join(t.TempDir(), "product")
	run("go", "build", "-buildvcs=true", "-o", binary, ".")
	info, err := buildinfo.ReadFile(binary)
	if err != nil {
		t.Fatal(err)
	}
	settings := map[string]string{}
	for _, setting := range info.Settings {
		settings[setting.Key] = setting.Value
	}
	if settings["vcs.revision"] != revision || settings["vcs.modified"] != "false" {
		t.Fatalf("build must record the clean fixture revision: %v", settings)
	}
}
