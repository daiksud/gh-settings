package githubactions

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

const commitlintWorkflow = "../../.github/workflows/commitlint.yml"

func TestCommitlintRunsForEveryPR(t *testing.T) {
	w, _ := readWorkflow(t, commitlintWorkflow)
	var event struct {
		Types []string `yaml:"types"`
	}
	node := w.On["pull_request"]
	if len(w.On) != 1 || len(node.Content) != 2 || node.Content[0].Value != "types" {
		t.Fatal("commitlint requires pull_request events without branch or path filters")
	}
	if err := node.Decode(&event); err != nil {
		t.Fatal(err)
	}
	for _, eventType := range []string{"opened", "synchronize", "reopened", "edited"} {
		if !slices.Contains(event.Types, eventType) {
			t.Fatalf("commitlint must run on %s", eventType)
		}
	}
	j, ok := w.Jobs["commitlint"]
	if !ok || j.If != "" || j.Continue || (j.Name != "" && j.Name != "commitlint") {
		t.Fatal("every PR must report the commitlint check, including Dependabot and drafts")
	}
	if len(w.Permissions) != 1 || w.Permissions["contents"] != "read" || len(j.Permissions) != 0 {
		t.Fatal("checking PR-supplied commitlint configuration requires a read-only token")
	}
	checkout := false
	for _, s := range j.Steps {
		if s.If != "" || s.Continue {
			t.Fatal("validation must always run and propagate failures")
		}
		if strings.HasPrefix(s.Uses, "actions/checkout@") {
			checkout = s.With["ref"] == "${{ github.event.pull_request.head.sha }}" &&
				s.With["fetch-depth"] == "0" && s.With["persist-credentials"] == "false"
		}
	}
	if !checkout {
		t.Fatal("validate complete PR head history without persisting checkout credentials")
	}
}

// Locate the actual workflow script by the event fields it consumes and bind
// their fixture values using the workflow's environment variable names.
func commitlintScript(t *testing.T, fields map[string]string) (string, []string) {
	t.Helper()
	w, _ := readWorkflow(t, commitlintWorkflow)
	for _, s := range w.Jobs["commitlint"].Steps {
		if s.Run == "" || len(s.Env) != len(fields) {
			continue
		}
		var env []string
		for field, value := range fields {
			for name, expression := range s.Env {
				if expression == "${{ github.event.pull_request."+field+" }}" {
					env = append(env, name+"="+value)
				}
			}
		}
		if len(env) == len(fields) {
			return s.Run, env
		}
	}
	t.Fatalf("missing validation step for fields %v", fields)
	return "", nil
}

func TestCommitlintMessageAndRange(t *testing.T) {
	// Simulate the validator to exercise message and argument transport and
	// failure propagation without a network install in the Go test suite.
	const runner = `#!/bin/sh
case " $* " in *" --strict "*) exit 99 ;; esac
case " $* " in
  *" --from "*) printf '%s\n' "$@" ;;
  *) cat ;;
esac
exit "$LINT_EXIT"
`
	bin := t.TempDir()
	if err := os.WriteFile(filepath.Join(bin, "bunx"), []byte(runner), 0700); err != nil {
		t.Fatal(err)
	}
	title := `ci: preserve "$literal" $(printf injected) *`
	body := "First line with `backticks` and %s.\nSecond line."
	base, head := strings.Repeat("1", 40), strings.Repeat("2", 40)
	for _, tc := range []struct {
		name     string
		fields   map[string]string
		expected []string
	}{
		{"PR commits", map[string]string{"base.sha": base, "head.sha": head}, []string{"--from\n" + base + "\n", "--to\n" + head + "\n"}},
		{"squash message", map[string]string{"title": title, "body": body}, []string{title + "\n\n" + body + "\n"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			script, env := commitlintScript(t, tc.fields)
			for _, exit := range []string{"0", "7"} {
				cmd := exec.Command("bash", "-euo", "pipefail", "-c", script)
				cmd.Env = append(os.Environ(), env...)
				cmd.Env = append(cmd.Env, "PATH="+bin+string(os.PathListSeparator)+os.Getenv("PATH"), "LINT_EXIT="+exit)
				output, err := cmd.CombinedOutput()
				if (err == nil) != (exit == "0") {
					t.Fatalf("commitlint exit %s: %v\n%s", exit, err, output)
				}
				for _, expected := range tc.expected {
					if !strings.Contains(string(output), expected) {
						t.Fatalf("validator did not receive %q: %q", expected, output)
					}
				}
			}
		})
	}
}

func TestCommitlintRejectsHTMLCommentsAnywhereInPRBody(t *testing.T) {
	for _, body := range []string{"", "Plain\nbody", "<!-- unfinished", "body <!-- help -->", "```html\n<!-- -->\n```"} {
		script, env := commitlintScript(t, map[string]string{"body": body})
		cmd := exec.Command("bash", "-euo", "pipefail", "-c", script)
		cmd.Env = append(os.Environ(), env...)
		if err := cmd.Run(); (err != nil) != strings.Contains(body, "<!--") {
			t.Fatalf("body %q returned %v", body, err)
		}
	}
}
