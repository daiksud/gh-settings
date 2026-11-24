package githubactions

import (
	"reflect"
	"strings"
	"testing"
)

func TestPRWorkflowScreeningCheck(t *testing.T) {
	w, source := readWorkflow(t, "../../.github/workflows/pr.yml")
	pullRequest := w.On["pull_request"]
	var filters map[string][]string
	if w.Name != "pr" || len(w.On) != 1 || pullRequest.Decode(&filters) != nil ||
		!reflect.DeepEqual(filters, map[string][]string{"types": {"opened", "synchronize", "reopened", "edited"}}) {
		t.Fatal("PR screening must run for candidate updates, including base changes")
	}
	check := w.Jobs["verify"]
	if len(w.Jobs) != 1 || len(w.Permissions) != 0 || check.Name != screeningCheckName || check.If != candidateEventCondition ||
		check.Needs != "" || check.Continue || check.Environment != "" || check.Runner != "ubuntu-slim" ||
		!reflect.DeepEqual(check.Permissions, map[string]string{"contents": "read"}) {
		t.Fatal("the read-only screening check must report its result for each candidate")
	}
	if w.Concurrency.Group != "" || check.Concurrency.Group != "pr-${{ github.event.pull_request.number }}" || !check.Concurrency.Cancel {
		t.Fatal("only a new candidate may cancel active screening")
	}
	if strings.Contains(source, "/statuses/") || strings.Contains(source, "gh pr ready") {
		t.Fatal("PR screening cannot publish a separate status or change the PR state")
	}
	for _, s := range check.Steps {
		if s.Env["GH_TOKEN"] != "" || s.Env["GITHUB_TOKEN"] != "" || s.If != "" || s.Continue {
			t.Fatalf("PR checks must preserve failures and isolate credentials: %+v", s)
		}
	}
	if len(check.Steps) == 0 || !strings.HasPrefix(check.Steps[0].Uses, "actions/checkout@") ||
		!reflect.DeepEqual(check.Steps[0].With, map[string]string{"persist-credentials": "false"}) {
		t.Fatal("screening must check out the merge candidate without persisting credentials")
	}
}
