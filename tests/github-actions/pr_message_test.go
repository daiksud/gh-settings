package githubactions

import (
	"reflect"
	"slices"
	"strings"
	"testing"
)

func TestIndependentPRMessageValidation(t *testing.T) {
	w, source := readWorkflow(t, "../../.github/workflows/prlint.yml")
	if w.Name != "PRLint" || len(w.On) != 1 || w.On["pull_request"].Kind == 0 || len(w.Jobs) != 1 || len(w.Permissions) != 0 {
		t.Fatal("message validation must remain a separate read-only PR workflow")
	}
	var event struct {
		Types []string `yaml:"types"`
	}
	node := w.On["pull_request"]
	if err := node.Decode(&event); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(event.Types, []string{"opened", "synchronize", "reopened", "edited"}) || w.Concurrency.Group != "pr-message-${{ github.event.pull_request.number }}" || !w.Concurrency.Cancel {
		t.Fatal("message edits need their own concurrency group and validation event")
	}
	j := w.Jobs["pr-message"]
	if j.If != "" || j.Needs != "" || j.Environment != "" || j.Continue || j.Timeout != 5 || !reflect.DeepEqual(j.Permissions, map[string]string{"contents": "read"}) || len(j.Steps) != 3 {
		t.Fatal("stable required message job must always run without write permissions")
	}
	if j.Steps[0].Uses != "actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1" || j.Steps[0].With["persist-credentials"] != "false" {
		t.Fatal("message policy checkout must not retain credentials")
	}
	if j.Steps[1].Uses != "jdx/mise-action@2d8d4cafcbd33be2ea37d2b6f5ad595363d1f1ca" || j.Steps[1].With["install_args"] != "bun npm:@commitlint/cli" {
		t.Fatal("install the pinned commitlint tools")
	}
	validate := j.Steps[2]
	if !reflect.DeepEqual(validate.Env, map[string]string{"PR_TITLE": "${{ github.event.pull_request.title }}", "PR_BODY": "${{ github.event.pull_request.body }}"}) || !strings.Contains(validate.Run, `printf '%s\n\n%s\n' "$PR_TITLE" "$PR_BODY"`) || !strings.Contains(validate.Run, "--strict") {
		t.Fatal("validate the title and body as one message through environment values")
	}
	for _, s := range j.Steps {
		if s.If != "" || s.Continue {
			t.Fatal("message validation must not skip or ignore a failed step")
		}
	}
	for _, forbidden := range []string{"secrets.", "gh pr ready", "go build", "acceptance-stage", "candidate-smoke"} {
		if strings.Contains(source, forbidden) {
			t.Fatalf("metadata-only workflow contains %q", forbidden)
		}
	}
}
