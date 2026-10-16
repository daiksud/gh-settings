package githubactions

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

const operationalAction = "daiksud/gh-settings@9d01d3258d9f42239b2daeb1ac26bc532cd56105"

func TestOperationalPRPlan(t *testing.T) {
	w, text := readWorkflow(t, "../../.github/workflows/settings-pr.yml")
	j := w.Jobs["plan"]
	if len(j.Steps) != 2 || j.Steps[1].Uses != operationalAction || !reflect.DeepEqual(j.Steps[1].With, map[string]string{"command": "plan", "token": "${{ github.token }}"}) {
		t.Fatal("operational PR must call the released public Action with an explicit read token")
	}
	if len(w.On) != 1 || w.On["pull_request"].Kind == 0 || len(w.Jobs) != 1 || j.Environment != "" || j.If != "" || len(j.Env) != 0 || len(j.Outputs) != 0 || j.Continue || j.Runner != "ubuntu-24.04" || j.Timeout != 5 {
		t.Fatal("operational PR trust boundary changed")
	}
	if !reflect.DeepEqual(w.Permissions, map[string]string{"contents": "read"}) || w.Defaults.Run.Shell != "bash" || !w.Concurrency.Cancel || w.Concurrency.Group != "settings-pr-${{ github.repository }}-${{ github.event.pull_request.number }}" || strings.Contains(text, "secrets.") {
		t.Fatal("operational PR permissions or concurrency changed")
	}
	if !reflect.DeepEqual(j.Steps[0].With, map[string]string{"repository": "${{ github.repository }}", "ref": "${{ github.sha }}", "persist-credentials": "false"}) || j.Steps[0].Uses != "actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1" {
		t.Fatal("operational PR must check out its exact merge configuration without persisted credentials")
	}
	for _, scenario := range []struct {
		name, output string
		status       int
	}{
		{"differences", "before → desired", 0},
		{"no differences", "No differences for managed repository settings.", 0},
		{"invalid configuration", "", 37},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			f := newCompositeFixture(t)
			f.base["GITHUB_EVENT_NAME"] = "pull_request"
			f.base["GITHUB_WORKFLOW_REF"] = "owner/config/.github/workflows/settings-pr.yml@refs/pull/1/merge"
			f.base["GITHUB_WORKFLOW_SHA"] = strings.Repeat("b", 40)
			f.base["MOCK_PLAN_OUTPUT"] = scenario.output
			f.base["MOCK_VALIDATE_STATUS"] = fmt.Sprint(scenario.status)
			r := f.call(t, j.Steps[1], map[string]string{"github.token": "read-fixture"}, nil)
			if r.status != scenario.status || r.api != "" || strings.Contains(r.log+r.summary+fmt.Sprint(r.outputs), "read-fixture") {
				t.Fatalf("operational PR result: %+v", r)
			}
			if scenario.status == 0 {
				if r.calls != "validate\nplan\n" || r.outputs["config-sha"] != f.sha || r.outputs["target"] != "owner/config" || r.outputs["cli-version"] != "v0.6.1" || r.outputs["cli-sha256"] != f.digest || !strings.Contains(r.summary, strings.TrimPrefix(operationalAction, "daiksud/gh-settings@")) {
					t.Fatalf("operational PR identity or execution: %+v", r)
				}
			} else if r.calls != "validate\n" || len(r.outputs) != 0 {
				t.Fatalf("invalid configuration reached plan or published outputs: %+v", r)
			}
		})
	}
}

func TestOperationalActionHandoff(t *testing.T) {
	cmd := exec.Command("bash", filepath.Join(root(), "tests/github-actions/action-handoff.sh"))
	cmd.Dir = root()
	cmd.Env = []string{"PATH=" + os.Getenv("PATH"), "HOME=" + os.Getenv("HOME"), "TMPDIR=" + os.Getenv("TMPDIR"),
		"VERIFY_COMMAND=bash \"$VERIFY_SCRIPT\"", "VERIFY_SCRIPT=" + filepath.Join(root(), ".github/scripts/verify-settings-handoff.sh"),
		"REVIEWED_ACTION_SHA=9d01d3258d9f42239b2daeb1ac26bc532cd56105", "FIXTURE_ACTION_SHA=9d01d3258d9f42239b2daeb1ac26bc532cd56105",
		"FIXTURE_CLI_VERSION=v0.6.1", "FIXTURE_CLI_SHA256=7b93802a64609ed5daf1b3bd02e33ad9384fda26b7776233cd094c8dd85d5874"}
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("operational pre-token handoff failed: %v\n%s", err, output)
	}
}
