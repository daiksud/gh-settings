package githubactions

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestPRLintEventsAndDependabotAuthor(t *testing.T) {
	w, _ := readWorkflow(t, "../../.github/workflows/prlint.yml")
	var event struct {
		Types []string `yaml:"types"`
	}
	node := w.On["pull_request"]
	if err := node.Decode(&event); err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{"opened", "synchronize", "reopened", "edited"} {
		if !slices.Contains(event.Types, required) {
			t.Fatalf("PRLint must run on %s", required)
		}
	}
	for i := 0; i < len(node.Content); i += 2 {
		if slices.Contains([]string{"paths", "paths-ignore", "branches", "branches-ignore"}, node.Content[i].Value) {
			t.Fatal("workflow filters must not omit the required prlint check")
		}
	}
	j, ok := w.Jobs["prlint"]
	if !ok || strings.TrimSpace(j.If) != "github.event.pull_request.user.login != 'dependabot[bot]'" {
		t.Fatal("the required prlint job must exclude Dependabot by PR author, not event actor")
	}
}

func TestPRLintMessageAndFailurePropagation(t *testing.T) {
	w, _ := readWorkflow(t, "../../.github/workflows/prlint.yml")
	j := w.Jobs["prlint"]
	var validate step
	var titleInput, bodyInput string
	for _, s := range j.Steps {
		var titleName, bodyName string
		for name, value := range s.Env {
			switch value {
			case "${{ github.event.pull_request.title }}":
				titleName = name
			case "${{ github.event.pull_request.body }}":
				bodyName = name
			}
		}
		if s.Run != "" && titleName != "" && bodyName != "" {
			validate, titleInput, bodyInput = s, titleName, bodyName
			break
		}
	}
	if titleInput == "" || bodyInput == "" || j.Continue || validate.Continue || validate.If != "" {
		t.Fatal("ordinary PR message validation must run and propagate failures")
	}

	// Run the workflow's shell script with a local validator stand-in. This
	// checks message transport and exit status without installing tools or
	// asserting any action version, step position or runner configuration.
	const runner = `#!/bin/sh
if [ "$1" = which ]; then
  printf 'commitlint\n'
  exit 0
fi
case " $* " in *" commitlint "*) ;; *) exit 99 ;; esac
case " $* " in *" --strict "*) ;; *) exit 99 ;; esac
cat > "$PRLINT_CAPTURE"
exit "$PRLINT_EXIT"
`
	title := `ci: preserve "$literal" $(printf injected) *`
	body := "First line with `backticks` and %s.\nSecond line."
	for _, exit := range []string{"0", "7"} {
		t.Run("validator-exit-"+exit, func(t *testing.T) {
			bin := t.TempDir()
			if err := os.WriteFile(filepath.Join(bin, "mise"), []byte(runner), 0700); err != nil {
				t.Fatal(err)
			}
			capture := filepath.Join(bin, "message")
			cmd := exec.Command("bash", "--noprofile", "--norc", "-euo", "pipefail", "-c", validate.Run)
			cmd.Env = append(os.Environ(), "PATH="+bin+string(os.PathListSeparator)+os.Getenv("PATH"), titleInput+"="+title, bodyInput+"="+body, "PRLINT_CAPTURE="+capture, "PRLINT_EXIT="+exit)
			output, err := cmd.CombinedOutput()
			if (err == nil) != (exit == "0") {
				t.Fatalf("validator exit %s: %v\n%s", exit, err, output)
			}
			message, err := os.ReadFile(capture)
			if err != nil {
				t.Fatal(err)
			}
			if string(message) != title+"\n\n"+body+"\n" {
				t.Fatalf("title/body changed in transit: %q", message)
			}
		})
	}
}
