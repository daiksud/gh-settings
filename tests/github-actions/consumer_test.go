package githubactions

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestPRConsumerPlan(t *testing.T) {
	for _, test := range []struct {
		name, outcome, review string
		status                int
	}{
		{"diff", "changed", "Observed before → desired after.", 0},
		{"no diff", "no-difference", "No differences for managed settings.", 0},
		{"invalid config", "", "Error — plan failed.", 37},
	} {
		t.Run(test.name, func(t *testing.T) {
			w, _ := readWorkflow(t, "settings-pr.yml")
			job := w.Jobs["plan"]
			f := newCompositeFixture(t)
			if err := os.RemoveAll(filepath.Join(f.caller, "action")); err != nil {
				t.Fatal(err)
			}
			if test.status != 0 {
				f.write(filepath.Join(f.caller, ".github/settings.yml"), "schemaVersion: [\n")
				f.git("add", ".github/settings.yml")
				f.git("-c", "user.name=Fixture", "-c", "user.email=fixture@example.test", "-c", "commit.gpgsign=false", "commit", "--quiet", "-m", "invalid configuration")
				f.sha = f.git("rev-parse", "HEAD")
				f.base["MOCK_VALIDATE_STATUS"] = fmt.Sprint(test.status)
			}
			context := map[string]string{"github.repository": "owner/config", "github.sha": f.sha, "github.token": "read-fixture"}
			checkout := job.Steps[0]
			if fixtureValue(t, checkout.With["repository"], context) != "owner/config" || fixtureValue(t, checkout.With["ref"], context) != f.git("rev-parse", "HEAD") {
				t.Fatal("PR checkout did not select the triggering configuration")
			}
			f.base["GITHUB_EVENT_NAME"] = "pull_request"
			f.base["GITHUB_WORKFLOW_REF"] = "owner/config/.github/workflows/settings-pr.yml@refs/pull/1/merge"
			f.base["GITHUB_WORKFLOW_SHA"] = strings.Repeat("b", 40)
			f.base["MOCK_PLAN_OUTCOME"] = test.outcome
			result := f.call(t, job.Steps[1], context, nil)
			want := map[string]string{"config-sha": f.sha, "target": "owner/config", "cli-version": committedCLIVersion(t), "cli-sha256": f.digest}
			calls := "plan\n"
			if test.status != 0 {
				want = map[string]string{}
			}
			if result.status != test.status || result.calls != calls || result.api != "" || !reflect.DeepEqual(result.outputs, want) {
				t.Fatalf("PR plan boundary failed: %+v", result)
			}
			for _, value := range []string{test.review, f.sha, "pull_request", ".github/workflows/settings-pr.yml", "daiksud/gh-settings", strings.TrimPrefix(exampleAction, "daiksud/gh-settings@")} {
				if !strings.Contains(result.summary, value) {
					t.Fatalf("PR Summary missing %q: %s", value, result.summary)
				}
			}
			if strings.Contains(result.log+result.summary+fmt.Sprint(result.outputs), "read-fixture") {
				t.Fatal("read token escaped the PR plan boundary")
			}
		})
	}
}

type consumerResult struct {
	prepared, applied compositeResult
	status            int
	minted, revoked   bool
	trace             []string
	log, summary      string
}

func (result consumerResult) assertNoCredentials(t *testing.T) {
	t.Helper()
	data := result.log + result.summary + result.prepared.summary + result.applied.summary + fmt.Sprint(result.prepared.outputs, result.applied.outputs)
	for _, secret := range []string{"read-fixture", "app-fixture", "fixture-key"} {
		if strings.Contains(data, secret) {
			t.Fatalf("dummy credential escaped its boundary: %s", secret)
		}
	}
}

func TestMainConsumerApply(t *testing.T) {
	for _, test := range []struct {
		scenario, calls, api string
		status, outputs      int
	}{
		{"approved", "apply\n", "PATCH\n", 0, 4},
		{"apply failure", "apply\n", "PATCH\n", 37, 0},
		{"invalid configuration", "apply\n", "", 37, 0},
		{"apply installation failure", "", "", 1, 0},
	} {
		t.Run(test.scenario, func(t *testing.T) {
			result := runMainConsumer(t, test.scenario)
			result.assertNoCredentials(t)
			if result.status != test.status || !result.minted || !result.revoked || result.applied.calls != test.calls || result.applied.api != test.api || len(result.applied.outputs) != test.outputs {
				t.Fatalf("single-job apply result: %+v", result)
			}
			wantTrace := append([]string{"checkout", "mint"}, strings.Fields(test.calls)...)
			wantTrace = append(wantTrace, "revoke")
			if !reflect.DeepEqual(result.trace, wantTrace) || result.prepared.sha != "" || result.summary != "" {
				t.Fatalf("single-job execution order or unexpected prepare/Summary: %+v", result)
			}
			if test.status == 0 {
				want := map[string]string{"config-sha": result.applied.sha, "target": "owner/config", "cli-version": committedCLIVersion(t), "cli-sha256": result.applied.digest}
				if !reflect.DeepEqual(result.applied.outputs, want) {
					t.Fatalf("apply checkout/release identity=%v, want %v", result.applied.outputs, want)
				}
			}
		})
	}
}

func TestMainConsumerBlocked(t *testing.T) {
	for _, scenario := range []string{"waiting", "rejected", "pull request", "other branch", "tag push", "mint failure"} {
		t.Run(scenario, func(t *testing.T) {
			result := runMainConsumer(t, scenario)
			result.assertNoCredentials(t)
			wantStatus := 0
			wantTrace := []string{scenario}
			if scenario == "mint failure" {
				wantStatus, wantTrace = 1, []string{"checkout"}
				if !strings.Contains(result.log, "dummy token provider rejected credentials") {
					t.Fatalf("missing mint failure diagnostic: %+v", result)
				}
			}
			if result.status != wantStatus || !reflect.DeepEqual(result.trace, wantTrace) || result.minted || result.revoked || result.applied.sha != "" || result.applied.api != "" {
				t.Fatalf("blocked single-job consumer reached privileged work: %+v", result)
			}
		})
	}
}

func runMainConsumer(t *testing.T, scenario string, workflowFile ...string) (result consumerResult) {
	t.Helper()
	name, expectedEnvironment := "settings-main.yml", "settings-apply"
	clientID, privateKey := "vars.SETTINGS_APP_CLIENT_ID", "secrets.SETTINGS_APP_PRIVATE_KEY"
	if len(workflowFile) != 0 {
		name, expectedEnvironment = workflowFile[0], "gh-settings"
		clientID, privateKey = "vars.ANSWER26_CLIENT_ID", "secrets.ANSWER26_PRIVATE_KEY"
	}
	w, _ := readWorkflow(t, name)
	applyJob := w.Jobs["apply"]
	event, ref := "push", "refs/heads/main"
	switch scenario {
	case "pull request":
		event = "pull_request"
	case "other branch":
		ref = "refs/heads/other"
	case "tag push":
		ref = "refs/tags/main"
	}
	// Model the exact job condition and Environment gate before any caller step.
	if applyJob.If != "github.event_name == 'push' && github.ref == 'refs/heads/main'" || applyJob.Environment != expectedEnvironment {
		t.Fatal("unmodeled caller approval boundary")
	}
	if event != "push" || ref != "refs/heads/main" || scenario == "waiting" || scenario == "rejected" {
		result.trace = append(result.trace, scenario)
		return
	}
	apply := newCompositeFixture(t)
	if err := os.RemoveAll(filepath.Join(apply.caller, "action")); err != nil {
		t.Fatal(err)
	}
	apply.base["GITHUB_EVENT_NAME"], apply.base["GITHUB_REF"] = event, ref
	apply.base["GITHUB_SHA"], apply.base["GITHUB_REPOSITORY"] = apply.sha, "owner/config"
	apply.base["GITHUB_WORKFLOW_REF"] = "owner/config/.github/workflows/settings-main.yml@refs/heads/main"
	apply.base["GITHUB_WORKFLOW_SHA"] = strings.Repeat("b", 40)
	context := map[string]string{
		"github.repository": "owner/config", "github.repository_owner": "owner", "github.event.repository.name": "config", "github.sha": apply.sha,
	}
	context[clientID], context[privateKey] = "fixture-client", "fixture-key"
	checkout := applyJob.Steps[0]
	repository := fixtureValue(t, checkout.With["repository"], context)
	if repository == "" {
		repository = context["github.repository"] // actions/checkout's containing-repository default.
	}
	if repository != "owner/config" || fixtureValue(t, checkout.With["ref"], context) != apply.git("rev-parse", "HEAD") || checkout.With["persist-credentials"] != "false" {
		t.Fatal("main checkout did not select the triggering configuration without persisted credentials")
	}
	result.trace = append(result.trace, "checkout")
	// Token minting and post-job revocation are local stubs, not GitHub operations.
	app := applyJob.Steps[1]
	if app.With["skip-token-revoke"] != "" && app.With["skip-token-revoke"] != "false" {
		t.Fatal("default post-job token revocation disabled")
	}
	if scenario == "mint failure" {
		context[privateKey] = ""
	}
	for key, want := range map[string]string{"owner": "owner", "repositories": "config", "permission-administration": "write", "client-id": "fixture-client", "private-key": "fixture-key"} {
		if fixtureValue(t, app.With[key], context) != want {
			result.log, result.status = "dummy token provider rejected credentials\n", 1
			return
		}
	}
	result.minted = true
	result.trace = append(result.trace, "mint")
	defer func() {
		result.revoked = true
		result.trace = append(result.trace, "revoke")
	}()
	context["steps.app.outputs.token"] = "app-fixture"
	extra := map[string]string{}
	switch scenario {
	case "apply failure":
		extra["MOCK_REMOTE_STATUS"] = "37"
	case "invalid configuration":
		extra["MOCK_VALIDATE_STATUS"] = "37"
	case "apply installation failure":
		extra["RUNNER_OS"] = "Windows"
	}
	result.applied = apply.call(t, applyJob.Steps[2], context, extra)
	result.status, result.summary = result.applied.status, result.applied.summary
	result.log += result.applied.log
	result.trace = append(result.trace, strings.Fields(result.applied.calls)...)
	return
}

func (f *compositeFixture) call(t *testing.T, call step, context, extra map[string]string) compositeResult {
	t.Helper()
	repository, ref, ok := strings.Cut(call.Uses, "@")
	if !ok || repository != "daiksud/gh-settings" {
		t.Fatalf("unexpected public Action call %s", call.Uses)
	}
	// Model pinned caller wiring with candidate package bytes, not released tag bytes.
	if ref != strings.TrimPrefix(exampleAction, "daiksud/gh-settings@") {
		t.Fatalf("unmodeled public Action revision %s", ref)
	}
	overrides := map[string]string{
		"github.action_repository": repository, "github.action_ref": ref,
		"github.event_name": f.base["GITHUB_EVENT_NAME"], "github.workflow_ref": f.base["GITHUB_WORKFLOW_REF"], "github.workflow_sha": f.base["GITHUB_WORKFLOW_SHA"],
	}
	for key, value := range extra {
		overrides[key] = value
	}
	return f.run(t, fixtureValue(t, call.With["command"], context), fixtureValue(t, call.With["token"], context), overrides)
}
