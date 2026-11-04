package githubactions

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestWorkflowLintOnlyAcceptsLiteralInertBridgeDiagnostics(t *testing.T) {
	python, err := exec.LookPath("python3")
	if err != nil {
		t.Fatal(err)
	}
	script, err := os.ReadFile(filepath.Join(root(), ".github/scripts/lint-workflows.py"))
	if err != nil {
		t.Fatal(err)
	}
	const bridge = "$/.github/actions/lock-workspace-dependencies"
	const screening = "$/.github/actions/lock-workspace-dependencies-screening"
	actionMessage := `specifying action "` + bridge + `" in invalid format because ref is missing. available formats are "{owner}/{repo}@{ref}" or "{owner}/{repo}/{path}@{ref}"`
	screeningMessage := strings.ReplaceAll(actionMessage, bridge, screening)
	ifMessage := `constant expression "false" in condition. remove the if: section`
	for _, test := range []struct {
		name, file, source, message, stderr string
		kind                                string
		line, status                        int
		valid                               bool
	}{
		{name: "inert same-repo reference", file: "ci-pr.yml", kind: "action", line: 4, message: screeningMessage, status: 1, valid: true},
		{name: "intentional false guard", file: "release.yml", kind: "if-cond", line: 3, message: ifMessage, status: 1, valid: true},
		{name: "other same-repo reference", file: "ci-pr.yml", kind: "action", line: 4, message: strings.ReplaceAll(screeningMessage, screening, "$/.github/actions/other"), status: 1},
		{name: "unrelated false guard", file: "ci-pr.yml", kind: "if-cond", line: 2, message: ifMessage, status: 1},
		{name: "other workflow", file: "other.yml", kind: "action", line: 4, message: actionMessage, status: 1},
		{name: "enabled bridge", file: "ci-pr.yml", source: "steps:\n  - name: Inventory\n    if: ${{ true }}\n    uses: " + screening + "\n", kind: "action", line: 4, message: screeningMessage, status: 1},
		{name: "tool failure", file: "ci-pr.yml", kind: "action", line: 4, message: screeningMessage, status: 2},
		{name: "stderr failure", file: "ci-pr.yml", kind: "action", line: 4, message: screeningMessage, status: 1, stderr: "tool error"},
		{name: "no diagnostics failure", file: "ci-pr.yml", message: "EMPTY", status: 1},
		{name: "malformed output", file: "ci-pr.yml", message: "MALFORMED", status: 1},
		{name: "non-array output", file: "ci-pr.yml", message: "OBJECT", status: 1},
		{name: "clean native support", file: "ci-pr.yml", message: "EMPTY", status: 0, valid: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			repo := t.TempDir()
			scripts := filepath.Join(repo, ".github/scripts")
			workflows := filepath.Join(repo, ".github/workflows")
			bin := filepath.Join(repo, "bin")
			for _, dir := range []string{scripts, workflows, bin} {
				if err := os.MkdirAll(dir, 0755); err != nil {
					t.Fatal(err)
				}
			}
			path := filepath.Join(scripts, "lint-workflows.py")
			if err := os.WriteFile(path, script, 0644); err != nil {
				t.Fatal(err)
			}
			workflow := filepath.Join(workflows, test.file)
			source := test.source
			if source == "" {
				source = "steps:\n  - name: Inventory\n    if: ${{ false }}\n    uses: " + bridge + "\n"
				if test.file == "ci-pr.yml" {
					source = "steps:\n  - name: Inventory\n    if: ${{ false }}\n    uses: " + screening + "\n"
				}
			}
			if err := os.WriteFile(workflow, []byte(source), 0644); err != nil {
				t.Fatal(err)
			}
			diagnostics, _ := json.Marshal([]any{map[string]any{"filepath": workflow, "kind": test.kind, "line": test.line, "column": 1, "message": test.message}})
			switch test.message {
			case "EMPTY":
				diagnostics = []byte("[]")
			case "MALFORMED":
				diagnostics = []byte("not json")
			case "OBJECT":
				diagnostics = []byte("{}")
			}
			output := filepath.Join(repo, "diagnostics.json")
			if err := os.WriteFile(output, diagnostics, 0644); err != nil {
				t.Fatal(err)
			}
			fake := "#!/bin/sh\ncat \"$LINT_JSON\"\nif [ -n \"$LINT_STDERR\" ]; then printf '%s' \"$LINT_STDERR\" >&2; fi\nexit \"$LINT_EXIT\"\n"
			if err := os.WriteFile(filepath.Join(bin, "actionlint"), []byte(fake), 0755); err != nil {
				t.Fatal(err)
			}
			status := "1"
			if test.status == 0 {
				status = "0"
			} else if test.status == 2 {
				status = "2"
			}
			cmd := exec.Command(python, path, workflow)
			cmd.Dir = repo
			cmd.Env = append(os.Environ(), "PATH="+bin+string(os.PathListSeparator)+os.Getenv("PATH"), "LINT_JSON="+output, "LINT_EXIT="+status, "LINT_STDERR="+test.stderr)
			out, err := cmd.CombinedOutput()
			if (err == nil) != test.valid {
				t.Fatalf("valid=%v err=%v output=%s", test.valid, err, out)
			}
		})
	}
}
