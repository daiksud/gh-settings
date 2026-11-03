package githubactions

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestSettingsPRMergeIdentity(t *testing.T) {
	w, _ := readWorkflow(t, "../../.github/workflows/settings-pr.yml")
	prepare := findStep(t, w.Jobs["settings-validate"], "candidate")
	realGit, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name, change string
		fail         bool
	}{
		{"unrelated file", "readme", false},
		{"settings changed", "settings", false},
		{"PR workflow changed", "pr-workflow", false},
		{"main workflow changed", "main-workflow", false},
		{"nested example added", "example", false},
		{"settings deleted", "delete", false},
		{"settings renamed away", "rename-away", false},
		{"example renamed in", "rename-in", false},
		{"similar directory excluded", "lookalike", false},
		{"event head moved", "head-moved", true},
		{"event base moved", "base-moved", true},
		{"invalid event SHA", "invalid-sha", true},
		{"checkout is not a merge", "not-merge", true},
		{"missing parent history", "missing-history", true},
		{"shallow checkout with parents", "shallow", false},
		{"replacement object ignored", "replace", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			directory := t.TempDir()
			source := filepath.Join(directory, "source")
			if err := os.Mkdir(source, 0700); err != nil {
				t.Fatal(err)
			}
			var environment []string
			for _, value := range os.Environ() {
				name, _, _ := strings.Cut(value, "=")
				if !strings.HasPrefix(name, "GIT_") && name != "GH_TOKEN" && name != "GITHUB_TOKEN" &&
					name != "HOME" && name != "XDG_CONFIG_HOME" && name != "PYTHONPATH" {
					environment = append(environment, value)
				}
			}
			environment = append(environment, "HOME="+directory, "XDG_CONFIG_HOME="+directory,
				"GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_NOSYSTEM=1")
			git := func(args ...string) string {
				t.Helper()
				command := exec.Command(realGit, args...)
				command.Dir = source
				command.Env = environment
				output, err := command.CombinedOutput()
				if err != nil {
					t.Fatalf("git %v: %v\n%s", args, err, output)
				}
				return strings.TrimSpace(string(output))
			}
			write := func(name, content string) {
				t.Helper()
				path := filepath.Join(source, name)
				if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte(content), 0700); err != nil {
					t.Fatal(err)
				}
			}
			git("init", "--quiet", "--template=", "--initial-branch=main")
			git("config", "user.name", "Fixture")
			git("config", "user.email", "fixture@example.test")
			git("config", "commit.gpgsign", "false")
			write(".github/settings.yml", "version: daiksud/gh-settings@v0\n")
			write(".github/workflows/settings-pr.yml", "name: fixture\n")
			write(".github/workflows/apply-settings.yml", "name: fixture\n")
			write("README.md", "base\n")
			// Neither Python startup imports nor Git conversion programs from
			// candidate content may execute during identity/path inspection.
			write("sitecustomize.py", "from pathlib import Path\nPath('PWNED').write_text('python startup')\n")
			write("candidate-hook.sh", "#!/bin/sh\ntouch PWNED\nexit 83\n")
			write(".gitattributes", "*.yml diff=fixture\n")
			git("config", "diff.external", filepath.Join(source, "candidate-hook.sh"))
			git("config", "diff.fixture.textconv", filepath.Join(source, "candidate-hook.sh"))
			git("add", ".")
			git("commit", "--quiet", "-m", "base")
			base := git("rev-parse", "HEAD")
			git("checkout", "--quiet", "-b", "candidate")
			switch test.change {
			case "settings":
				write(".github/settings.yml", "version: daiksud/gh-settings@v0\nrepository: {}\n")
				write("README.md", "mixed change\n")
			case "pr-workflow":
				write(".github/workflows/settings-pr.yml", "name: changed\n")
			case "main-workflow":
				write(".github/workflows/apply-settings.yml", "name: changed\n")
			case "example":
				write("examples/github-actions/nested/workflow.yml", "name: added\n")
			case "delete":
				if err := os.Remove(filepath.Join(source, ".github/settings.yml")); err != nil {
					t.Fatal(err)
				}
			case "rename-away":
				if err := os.Rename(filepath.Join(source, ".github/settings.yml"), filepath.Join(source, "saved-settings.yml")); err != nil {
					t.Fatal(err)
				}
			case "rename-in":
				if err := os.MkdirAll(filepath.Join(source, "examples/github-actions"), 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.Rename(filepath.Join(source, "README.md"), filepath.Join(source, "examples/github-actions/README.md")); err != nil {
					t.Fatal(err)
				}
			case "lookalike":
				write("examples/github-actions-extra/workflow.yml", "name: unrelated\n")
			default:
				write("README.md", "candidate\n")
			}
			git("add", "--all")
			git("commit", "--quiet", "-m", "candidate")
			head := git("rev-parse", "HEAD")
			git("checkout", "--quiet", "main")
			git("merge", "--quiet", "--no-ff", "-m", "merge candidate", "candidate")
			merge := git("rev-parse", "HEAD")
			expectedHead, expectedBase, consumer := head, base, source
			switch test.change {
			case "head-moved":
				expectedHead = base
			case "base-moved":
				expectedBase = head
			case "invalid-sha":
				expectedHead = "main\nuntrusted-value"
			case "not-merge":
				git("checkout", "--quiet", "--detach", head)
			case "shallow", "missing-history":
				depth := "2"
				if test.change == "missing-history" {
					depth = "1"
				}
				consumer = filepath.Join(directory, "consumer")
				git("clone", "--quiet", "--depth="+depth, "file://"+source, consumer)
			case "replace":
				tree := git("rev-parse", merge+"^{tree}")
				replacement := git("commit-tree", tree, "-p", head, "-p", base, "-m", "wrong parents")
				git("replace", merge, replacement)
			}
			outputPath := filepath.Join(directory, "github-output")
			if err := os.WriteFile(outputPath, nil, 0600); err != nil {
				t.Fatal(err)
			}
			command := exec.Command("bash", "--noprofile", "--norc", "-euo", "pipefail", "-c", prepare.Run)
			command.Dir = consumer
			command.Env = append(environment, "EXPECTED_HEAD="+expectedHead, "EXPECTED_BASE="+expectedBase,
				"GITHUB_OUTPUT="+outputPath, "GIT_DIR=/invalid/inherited/git", "GIT_WORK_TREE=/invalid/inherited/worktree",
				"GH_TOKEN=must-not-leak", "GITHUB_TOKEN=must-not-leak", "PYTHONPATH="+consumer)
			log, runErr := command.CombinedOutput()
			output, err := os.ReadFile(outputPath)
			if err != nil {
				t.Fatal(err)
			}
			if (runErr != nil) != test.fail {
				t.Fatalf("unexpected validation result: %v\n%s", runErr, log)
			}
			if test.fail {
				if len(output) != 0 || !strings.Contains(string(log), "merge candidate validation failed") {
					t.Fatalf("failed identity check published output: %q\n%s", output, log)
				}
			} else if len(output) != 0 ||
				!strings.Contains(string(log), merge) || !strings.Contains(string(log), base) || !strings.Contains(string(log), head) {
				t.Fatalf("unexpected output or verified identity: %q\n%s", output, log)
			}
			if strings.Contains(string(log), "must-not-leak") || strings.Contains(string(log), "untrusted-value") {
				t.Fatal("candidate identity or credentials leaked in failure output")
			}
			for _, path := range []string{source, consumer} {
				if _, err := os.Stat(filepath.Join(path, "PWNED")); !os.IsNotExist(err) {
					t.Fatalf("candidate code executed during validation: %v", err)
				}
			}
		})
	}
}
