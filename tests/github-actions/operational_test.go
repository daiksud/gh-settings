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

func TestOperationalMainConsumer(t *testing.T) {
	w, text := readWorkflow(t, "../../.github/workflows/settings-main.yml")
	prepare, apply := w.Jobs["prepare"], w.Jobs["apply"]
	if len(prepare.Steps) != 3 || len(apply.Steps) != 5 || findStep(t, prepare, "prepare").Uses != operationalAction || findStep(t, apply, "apply").Uses != operationalAction {
		t.Fatal("operational prepare/apply must use the same reviewed public Action")
	}
	guard := "github.event_name == 'push' && github.ref == 'refs/heads/main'"
	if len(w.On) != 1 || w.On["push"].Kind == 0 || len(w.Jobs) != 2 || prepare.If != guard || apply.If != guard || apply.Needs != "prepare" || apply.Environment != "gh-settings" || prepare.Environment != "" || len(w.Permissions) != 0 || w.Concurrency.Cancel || w.Concurrency.Group != "settings-main-${{ github.repository }}" {
		t.Fatal("operational approval, event or concurrency boundary changed")
	}
	for _, j := range []job{prepare, apply} {
		if !reflect.DeepEqual(j.Permissions, map[string]string{"contents": "read"}) || len(j.Env) != 0 || j.Continue || j.Runner != "ubuntu-24.04" || j.Timeout != 5 {
			t.Fatal("operational job permissions, runner or failure boundary changed")
		}
	}
	if prepare.Steps[0].With["ref"] != "${{ github.sha }}" || apply.Steps[0].With["ref"] != "${{ needs.prepare.outputs.config-sha }}" || prepare.Steps[0].With["persist-credentials"] != "false" || apply.Steps[0].With["persist-credentials"] != "false" {
		t.Fatal("prepare/apply configuration checkout changed")
	}
	if !reflect.DeepEqual(findStep(t, prepare, "prepare").With, map[string]string{"command": "plan", "token": "${{ github.token }}"}) || !reflect.DeepEqual(findStep(t, apply, "apply").With, map[string]string{"command": "apply", "token": "${{ steps.app.outputs.token }}"}) {
		t.Fatal("read/write token destination changed")
	}
	app := findStep(t, apply, "app")
	if app.Uses != "actions/create-github-app-token@64ab729a6294000f5aa4a4bc0b73d1f32bc45510" || !reflect.DeepEqual(app.With, map[string]string{"client-id": "${{ vars.ANSWER26_CLIENT_ID }}", "private-key": "${{ secrets.ANSWER26_PRIVATE_KEY }}", "owner": "${{ github.repository_owner }}", "repositories": "${{ github.event.repository.name }}", "permission-administration": "write"}) || apply.Steps[1].ID != "handoff" || apply.Steps[2].ID != "app" || apply.Steps[3].ID != "apply" {
		t.Fatal("pre-mint verification, existing App scope or default revocation changed")
	}
	if strings.Contains(text, "examples/github-actions/prepare.sh") || strings.Contains(text, "examples/github-actions/apply.sh") || len(apply.Outputs) != 0 {
		t.Fatal("legacy execution or credential output retained")
	}
	for _, scenario := range []string{"approved", "failed plan", "waiting", "rejected", "config mismatch", "target mismatch", "Action mismatch", "mint failure", "apply failure", "apply installation failure"} {
		t.Run(scenario, func(t *testing.T) {
			r := runMainConsumerWorkflow(t, scenario, "../../.github/workflows/settings-main.yml")
			r.assertNoCredentials(t)
			if scenario == "approved" || scenario == "apply failure" {
				if !r.minted || !r.revoked || r.applied.api != "PATCH\n" || r.prepared.sha != r.applied.sha || r.prepared.outputs["cli-version"] != "v0.6.1" || (r.status == 0) != (scenario == "approved") {
					t.Fatalf("approved operational result: %+v", r)
				}
			} else if scenario == "apply installation failure" {
				if r.status == 0 || !r.minted || !r.revoked || r.applied.api != "" || r.applied.calls != "" || len(r.applied.outputs) != 0 || !strings.Contains(r.summary, "CLI step outcome: failure") || !reflect.DeepEqual(r.trace, []string{"guard", "validate", "plan", "approved", "verify", "mint", "summary", "revoke"}) {
					t.Fatalf("post-mint installation failure boundary: %+v", r)
				}
			} else if r.minted || r.revoked || r.applied.sha != "" || r.prepared.api+r.applied.api != "" || (r.status == 0) != (scenario == "waiting" || scenario == "rejected") {
				t.Fatalf("blocked operational result: %+v", r)
			}
		})
	}
}
