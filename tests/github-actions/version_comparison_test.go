package githubactions

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"go.yaml.in/yaml/v4"
)

func versionJob(t *testing.T, file, id string) job {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root(), ".github/workflows", file))
	if err != nil {
		t.Fatal(err)
	}
	var w struct {
		Jobs map[string]struct{ Steps []step }
	}
	if err := yaml.Unmarshal(data, &w); err != nil {
		t.Fatal(err)
	}
	return job{Steps: w.Jobs[id].Steps}
}

func TestNativeVersionComparison(t *testing.T) {
	compare := versionJob(t, "version-compare.yml", "compare")
	action, _ := readReleaseAction(t, "build")
	build := findStep(t, job{Steps: action.Runs.Steps}, "build").Run
	workspace, temp := t.TempDir(), t.TempDir()
	output := filepath.Join(temp, "output")
	binary := filepath.Join(temp, "gh-settings")
	env := append(os.Environ(), "GOWORK=off", "GOTOOLCHAIN=local", "RUNNER_TEMP="+temp,
		"GITHUB_OUTPUT="+output, "BINARY="+binary, "VERSION=v0.3.0", "BUILD_VERSION=v0.3.0",
		"TARGET="+runtime.GOOS+"-"+runtime.GOARCH)
	write := func(name, content string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(workspace, name), []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir, cmd.Env = workspace, env
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%s: %v\n%s", args[0], err, out)
		}
	}
	commit := func(message string) {
		t.Helper()
		run("git", "add", ".")
		run("git", "-c", "commit.gpgsign=false", "-c", "user.name=Fixture", "-c", "user.email=fixture@example.org", "commit", "-qm", message)
	}
	runBuild := func(id string) {
		t.Helper()
		s := findStep(t, compare, id)
		if s.Uses != "./.github/actions/build" || s.With["version"] != "${{ inputs.previous-tag }}" {
			t.Fatal("comparison builds must use the checked-out recipe and a common version")
		}
		run("env", "GOFLAGS="+s.Env["GOFLAGS"], "bash", "-eo", "pipefail", "-c", build)
	}
	const source = `package main
import ("fmt"; "os")
var version = "dev"
func main() {
	if len(os.Args) == 2 && os.Args[1] == "--version" { fmt.Println("gh-settings", version); return }
	fmt.Println("old behavior")
}
`
	run("git", "init", "-q")
	write("go.mod", "module example.org/fixture\n\ngo 1.25\n")
	write("main.go", source)
	commit("feat: initial behavior")
	runBuild("previous")
	if err := os.Rename(binary, filepath.Join(temp, "previous-gh-settings")); err != nil {
		t.Fatal(err)
	}
	check := func(want string) {
		t.Helper()
		runBuild("candidate")
		if err := os.Remove(output); err != nil {
			t.Fatal(err)
		}
		run("bash", "-eo", "pipefail", "-c", findStep(t, compare, "compare").Run)
		result, err := os.ReadFile(output)
		if err != nil || string(result) != "changed="+want+"\n" {
			t.Fatalf("comparison = %q, %v; want %s", result, err, want)
		}
	}
	write("README.md", "Document the existing behavior.\n")
	commit("docs: explain the command")
	check("false")
	write("main.go", strings.Replace(source, "old behavior", "new behavior", 1))
	commit("refactor: change the implementation")
	check("true")
}

func TestComparisonChecksOutOnlyThePublishedAncestor(t *testing.T) {
	compare := versionJob(t, "version-compare.yml", "compare")
	for _, s := range compare.Steps {
		if strings.HasPrefix(s.Uses, "actions/checkout@") &&
			(s.With["ref"] != "${{ github.sha }}" || s.With["persist-credentials"] != "false") {
			t.Fatal("diagnostic builds must begin at the frozen main commit without retained credentials")
		}
		if strings.HasPrefix(s.Uses, "actions/setup-go@") && s.With["cache"] != "false" {
			t.Fatal("diagnostic builds must not write shared Go caches")
		}
	}
	dir := t.TempDir()
	git := func(args ...string) string {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %s: %v, %s", args[0], err, out)
		}
		return strings.TrimSpace(string(out))
	}
	git("init", "-q")
	git("config", "user.name", "Fixture")
	git("config", "user.email", "fixture@example.org")
	git("config", "commit.gpgsign", "false")
	git("commit", "-qm", "chore: released", "--allow-empty")
	published := git("rev-parse", "HEAD")
	git("tag", "v0.3.0")
	git("commit", "-qm", "fix: later change", "--allow-empty")
	source := git("rev-parse", "HEAD")
	unrelated := git("commit-tree", git("rev-parse", "HEAD^{tree}"), "-m", "chore: unrelated")
	git("tag", "v9.0.0", unrelated)
	selectAncestor := findStep(t, compare, "predecessor").Run
	for _, tc := range []struct{ tag, sha, want string }{
		{"v0.3.0", published, published},
		{"v0.3.0", source, ""},
		{"v0.4.0", published, ""},
		{"v9.0.0", unrelated, ""},
	} {
		git("checkout", "--quiet", "--detach", source)
		cmd := exec.Command("bash", "-eo", "pipefail", "-c", selectAncestor)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "GITHUB_SHA="+source, "PREVIOUS_TAG="+tc.tag, "PREVIOUS_SHA="+tc.sha)
		out, err := cmd.CombinedOutput()
		if tc.want == "" {
			if err == nil || git("rev-parse", "HEAD") != source {
				t.Fatalf("selected unverified tag %s at %s", tc.tag, tc.sha)
			}
		} else if err != nil || git("rev-parse", "HEAD") != tc.want {
			t.Fatalf("expected published ancestor %s: %s", tc.want, out)
		}
	}
}

func TestVersionSelectionRequiresBothNativeResults(t *testing.T) {
	script := findStep(t, versionJob(t, "version.yml", "resolve"), "select").Run
	for _, tc := range []struct {
		name, compare, linux, macos, linuxChanged, macosChanged, want string
	}{
		{"both identical", "true", "success", "success", "false", "false", "v0.3.0"},
		{"Linux changed", "true", "success", "success", "true", "false", "v0.3.1"},
		{"macOS changed", "true", "success", "success", "false", "true", "v0.3.1"},
		{"both changed", "true", "success", "success", "true", "true", "v0.3.1"},
		{"failed comparison", "true", "failure", "success", "", "true", ""},
		{"missing result", "true", "success", "success", "true", "", ""},
		{"skipped comparison", "true", "success", "skipped", "false", "", ""},
		{"semantic release", "false", "skipped", "skipped", "", "", "v0.3.1"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			output := filepath.Join(t.TempDir(), "outputs")
			sha := strings.Repeat("1", 40)
			cmd := exec.Command("bash", "-eo", "pipefail", "-c", script)
			cmd.Env = append(os.Environ(), "GITHUB_OUTPUT="+output, "VERSION=v0.3.1", "RELEASE_NEEDED=true",
				"PREVIOUS_TAG=v0.3.0", "PREVIOUS_SHA="+sha, "COMPARE_BINARIES="+tc.compare,
				"LINUX_RESULT="+tc.linux, "MACOS_RESULT="+tc.macos,
				"LINUX_CHANGED="+tc.linuxChanged, "MACOS_CHANGED="+tc.macosChanged)
			out, err := cmd.CombinedOutput()
			data, readErr := os.ReadFile(output)
			if tc.want == "" {
				if err == nil || !os.IsNotExist(readErr) {
					t.Fatalf("invalid comparison published a version: %q, %s", data, out)
				}
				return
			}
			want := fmt.Sprintf("version=%s\nprevious-tag=v0.3.0\nprevious-sha=%s\nrelease-needed=%t\n", tc.want, sha, tc.want != "v0.3.0")
			if err != nil || readErr != nil || string(data) != want {
				t.Fatalf("got %q, %v, %v (%s); want %q", data, err, readErr, out, want)
			}
		})
	}
}
