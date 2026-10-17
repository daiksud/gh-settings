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

func TestOperationalMainConsumerApplyDetails(t *testing.T) {
	for _, test := range []struct {
		scenario, outcome string
		status            int
	}{
		{"approved", "success", 0},
		{"apply failure", "failure", 37},
	} {
		t.Run(test.scenario, func(t *testing.T) {
			result := runMainConsumerWorkflow(t, test.scenario, "../../.github/workflows/settings-main.yml")
			if result.status != test.status || !result.minted || !result.revoked || result.applied.api != "PATCH\n" {
				t.Fatalf("status=%d minted=%t revoked=%t API=%q\n%s", result.status, result.minted, result.revoked, result.applied.api, result.log)
			}
			want := map[string]string{"config-sha": result.prepared.sha, "target": "owner/config", "cli-version": committedCLIVersion(t), "cli-sha256": result.prepared.digest}
			wantApplied := want
			if test.status != 0 {
				wantApplied = map[string]string{}
			}
			if !reflect.DeepEqual(result.prepared.outputs, want) || !reflect.DeepEqual(result.applied.outputs, wantApplied) || result.prepared.sha != result.applied.sha {
				t.Fatalf("prepare=%v apply=%v want=%v", result.prepared.outputs, result.applied.outputs, want)
			}
			if !reflect.DeepEqual(result.trace, []string{"guard", "plan", "approved", "verify", "mint", "apply", "summary", "revoke"}) {
				t.Fatalf("execution order=%v", result.trace)
			}
			if !strings.Contains(result.summary, "CLI step outcome: "+test.outcome) {
				t.Fatalf("missing apply outcome: %s", result.summary)
			}
			for _, identity := range []string{"daiksud/gh-settings", "9d01d3258d9f42239b2daeb1ac26bc532cd56105", result.prepared.sha, ".github/workflows/settings-main.yml"} {
				if !strings.Contains(result.prepared.summary, identity) {
					t.Fatalf("prepare Summary missing identity %q: %s", identity, result.prepared.summary)
				}
			}
			result.assertNoCredentials(t)
		})
	}
}

func TestOperationalMainConsumerPrepareFailure(t *testing.T) {
	for _, test := range []struct {
		scenario, calls, diagnostic string
		status                      int
	}{
		{"failed validate", "plan\n", "configuration validation failed", 37},
		{"failed plan", "plan\n", "repository read failed", 37},
		{"missing Summary", "plan\n", "missing SETTINGS_RUN_ID", 1},
		{"Summary write failure", "plan\n", "open Summary:", 1},
	} {
		t.Run(test.scenario, func(t *testing.T) {
			result := runMainConsumerWorkflow(t, test.scenario, "../../.github/workflows/settings-main.yml")
			result.assertNoCredentials(t)
			wantTrace := append([]string{"guard"}, strings.Fields(test.calls)...)
			if result.status != test.status || result.prepared.calls != test.calls || len(result.prepared.outputs) != 0 || !reflect.DeepEqual(result.trace, wantTrace) {
				t.Fatalf("prepare failure boundary failed: %+v", result)
			}
			if result.minted || result.revoked || result.applied.sha != "" || result.prepared.api+result.applied.api != "" || !strings.Contains(result.summary+result.log, test.diagnostic) {
				t.Fatalf("prepare failure reached privileged work: %+v", result)
			}
		})
	}
}

func TestOperationalMainConsumerBlockedDetails(t *testing.T) {
	for _, test := range []struct {
		scenario, diagnostic string
		status               int
	}{
		{"waiting", "", 0},
		{"rejected", "", 0},
		{"config mismatch", "configuration commit mismatch", 1},
		{"target mismatch", "prepare target mismatch", 1},
		{"Action mismatch", "prepare Action revision mismatch", 1},
		{"mint failure", "dummy token provider rejected credentials", 1},
	} {
		t.Run(test.scenario, func(t *testing.T) {
			result := runMainConsumerWorkflow(t, test.scenario, "../../.github/workflows/settings-main.yml")
			result.assertNoCredentials(t)
			wantTrace := []string{"guard", "plan", test.scenario}
			if test.status != 0 {
				wantTrace = []string{"guard", "plan", "approved"}
				if test.scenario == "mint failure" {
					wantTrace = append(wantTrace, "verify")
				}
				wantTrace = append(wantTrace, "summary")
				if !strings.Contains(result.log, test.diagnostic) || !strings.Contains(result.summary, "CLI step outcome: skipped") {
					t.Fatalf("missing handoff failure evidence: %+v", result)
				}
			}
			if result.status != test.status || len(result.prepared.outputs) != 4 || !reflect.DeepEqual(result.trace, wantTrace) {
				t.Fatalf("blocked consumer trace failed: %+v", result)
			}
			if result.minted || result.revoked || result.applied.sha != "" || result.prepared.api+result.applied.api != "" {
				t.Fatalf("blocked path reached privileged work: %+v", result)
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

func runMainConsumer(t *testing.T, scenario string) (result consumerResult) {
	t.Helper()
	w, _ := readWorkflow(t, "settings-main.yml")
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
	if applyJob.If != "github.event_name == 'push' && github.ref == 'refs/heads/main'" || applyJob.Environment != "settings-apply" {
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
		"vars.SETTINGS_APP_CLIENT_ID": "fixture-client", "secrets.SETTINGS_APP_PRIVATE_KEY": "fixture-key",
	}
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
		context["secrets.SETTINGS_APP_PRIVATE_KEY"] = ""
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

func runMainConsumerWorkflow(t *testing.T, scenario, workflowFile string) (result consumerResult) {
	t.Helper()
	w, text := readWorkflow(t, workflowFile)
	prepareJob, applyJob := w.Jobs["prepare"], w.Jobs["apply"]
	prepare := newCompositeFixture(t)
	if err := os.RemoveAll(filepath.Join(prepare.caller, "action")); err != nil {
		t.Fatal(err)
	}
	verifierPath := strings.TrimPrefix(findStep(t, applyJob, "handoff").Run, "bash ")
	for _, directory := range []string{".github/workflows", filepath.Dir(verifierPath)} {
		if err := os.MkdirAll(filepath.Join(prepare.caller, directory), 0700); err != nil {
			t.Fatal(err)
		}
	}
	prepare.write(filepath.Join(prepare.caller, ".github/workflows/settings-main.yml"), text)
	verifier, err := os.ReadFile(filepath.Join(root(), verifierPath))
	if err != nil {
		t.Fatal(err)
	}
	// Only candidate version and dummy checksum data change in the copied verifier.
	// Each production verifier also has an unchanged-identity handoff test.
	profile := strings.NewReplacer("== v0.6.1", "== "+committedCLIVersion(t), "7b93802a64609ed5daf1b3bd02e33ad9384fda26b7776233cd094c8dd85d5874", prepare.digest)
	prepare.write(filepath.Join(prepare.caller, verifierPath), profile.Replace(string(verifier)))
	prepare.git("add", ".github/workflows/settings-main.yml", verifierPath)
	prepare.git("-c", "user.name=Fixture", "-c", "user.email=fixture@example.test", "-c", "commit.gpgsign=false", "commit", "--quiet", "-m", "consumer")
	prepare.sha = prepare.git("rev-parse", "HEAD")
	prepare.base["GITHUB_EVENT_NAME"], prepare.base["GITHUB_REF"] = "push", "refs/heads/main"
	prepare.base["GITHUB_SHA"], prepare.base["GITHUB_REPOSITORY"] = prepare.sha, "owner/config"
	prepare.base["GITHUB_WORKFLOW_REF"] = "owner/config/.github/workflows/settings-main.yml@refs/heads/main"
	prepare.base["GITHUB_WORKFLOW_SHA"] = strings.Repeat("b", 40)
	context := map[string]string{"github.repository": "owner/config", "github.repository_owner": "owner", "github.event.repository.name": "config", "github.sha": prepare.sha, "github.token": "read-fixture"}
	result.status, result.log = prepare.runCaller(t, findStep(t, prepareJob, "guard"), context)
	if result.status != 0 {
		return
	}
	result.trace = append(result.trace, "guard")
	prepareExtra := map[string]string{}
	switch scenario {
	case "failed validate":
		prepareExtra["MOCK_VALIDATE_STATUS"] = "37"
	case "failed plan":
		prepareExtra["MOCK_REMOTE_STATUS"] = "37"
	case "missing Summary":
		prepareExtra["github.run_id"] = ""
	case "Summary write failure":
		prepareExtra["SUMMARY_DIRECTORY"] = "1"
	}
	result.prepared = prepare.call(t, findStep(t, prepareJob, "prepare"), context, prepareExtra)
	result.status = result.prepared.status
	result.log += result.prepared.log
	result.trace = append(result.trace, strings.Fields(result.prepared.calls)...)
	result.summary = result.prepared.summary
	if result.status != 0 {
		return
	}
	for name, value := range result.prepared.outputs {
		context["steps.prepare.outputs."+name] = value
	}
	for name, binding := range prepareJob.Outputs {
		context["needs.prepare.outputs."+name] = fixtureValue(t, binding, context)
	}

	// Approval and token mint/revocation are local stubs, not GitHub operations.
	if scenario == "waiting" || scenario == "rejected" {
		result.trace = append(result.trace, scenario)
		return
	}
	result.trace = append(result.trace, "approved")
	apply := newCompositeFixture(t)
	if err := os.RemoveAll(apply.caller); err != nil {
		t.Fatal(err)
	}
	clone := exec.Command("git", "clone", "--quiet", "--no-local", prepare.caller, apply.caller)
	clone.Env = fixtureEnvironment(apply.base)
	if output, err := clone.CombinedOutput(); err != nil {
		t.Fatalf("fresh apply checkout: %v\n%s", err, output)
	}
	checkout := applyJob.Steps[0]
	apply.git("checkout", "--quiet", fixtureValue(t, checkout.With["ref"], context))
	apply.git("remote", "set-url", "origin", "https://github.com/"+fixtureValue(t, checkout.With["repository"], context)+".git")
	if apply.git("rev-parse", "--show-toplevel") == prepare.git("rev-parse", "--show-toplevel") {
		t.Fatal("prepare and apply did not use separate checkouts")
	}
	apply.sha = apply.git("rev-parse", "HEAD")
	for _, name := range []string{"GITHUB_EVENT_NAME", "GITHUB_REF", "GITHUB_SHA", "GITHUB_REPOSITORY", "GITHUB_WORKFLOW_REF", "GITHUB_WORKFLOW_SHA"} {
		apply.base[name] = prepare.base[name]
	}
	context["steps.apply.outcome"] = "skipped"
	defer func() {
		status, log := apply.runCaller(t, applyJob.Steps[4], context)
		result.log += log
		if result.status == 0 {
			result.status = status
		}
		outcome, _ := os.ReadFile(apply.base["GITHUB_STEP_SUMMARY"])
		result.summary += string(outcome)
		result.trace = append(result.trace, "summary")
		if result.minted {
			result.revoked = true
			result.trace = append(result.trace, "revoke")
		}
	}()
	// Fault copied handoff data only after the checkout used the real prepared ref.
	switch scenario {
	case "config mismatch":
		context["needs.prepare.outputs.config-sha"] = strings.Repeat("a", 40)
	case "target mismatch":
		context["needs.prepare.outputs.target"] = "owner/other"
	case "Action mismatch":
		context["needs.prepare.outputs.action-sha"] = strings.Repeat("a", 40)
	}
	status, log := apply.runCaller(t, findStep(t, applyJob, "handoff"), context)
	result.log += log
	result.status = status
	if result.status != 0 {
		return
	}
	result.trace = append(result.trace, "verify")
	clientID, privateKey := "vars.SETTINGS_APP_CLIENT_ID", "secrets.SETTINGS_APP_PRIVATE_KEY"
	if applyJob.Environment == "gh-settings" {
		clientID, privateKey = "vars.ANSWER26_CLIENT_ID", "secrets.ANSWER26_PRIVATE_KEY"
	}
	context[clientID], context[privateKey] = "fixture-client", "fixture-key"
	if scenario == "mint failure" {
		context[privateKey] = ""
	}
	app := findStep(t, applyJob, "app")
	for key, want := range map[string]string{"owner": "owner", "repositories": "config", "permission-administration": "write", "client-id": "fixture-client", "private-key": "fixture-key"} {
		if fixtureValue(t, app.With[key], context) != want {
			result.log += "dummy token provider rejected credentials\n"
			result.status = 1
			return
		}
	}
	result.minted = true
	result.trace = append(result.trace, "mint")
	context["steps.app.outputs.token"] = "app-fixture"
	applyExtra := map[string]string{}
	if scenario == "apply failure" {
		applyExtra["MOCK_REMOTE_STATUS"] = "37"
	}
	if scenario == "apply installation failure" {
		applyExtra["RUNNER_OS"] = "Windows"
	}
	result.applied = apply.call(t, findStep(t, applyJob, "apply"), context, applyExtra)
	result.status = result.applied.status
	result.log += result.applied.log
	result.trace = append(result.trace, strings.Fields(result.applied.calls)...)
	context["steps.apply.outcome"] = "success"
	if result.status != 0 {
		context["steps.apply.outcome"] = "failure"
	}
	return
}

func (f *compositeFixture) call(t *testing.T, call step, context, extra map[string]string) compositeResult {
	t.Helper()
	repository, ref, ok := strings.Cut(call.Uses, "@")
	if !ok || repository != "daiksud/gh-settings" {
		t.Fatalf("unexpected public Action call %s", call.Uses)
	}
	// Model pinned caller wiring with candidate package bytes, not released tag bytes.
	if ref != strings.TrimPrefix(operationalAction, "daiksud/gh-settings@") && ref != strings.TrimPrefix(exampleAction, "daiksud/gh-settings@") {
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

func (f *compositeFixture) runCaller(t *testing.T, caller step, context map[string]string) (int, string) {
	t.Helper()
	env := map[string]string{}
	for key, value := range f.base {
		env[key] = value
	}
	for key, value := range caller.Env {
		env[key] = fixtureValue(t, value, context)
	}
	cmd := exec.Command("bash", "--noprofile", "--norc", "-e", "-o", "pipefail", "-c", caller.Run)
	cmd.Dir, cmd.Env = f.caller, fixtureEnvironment(env)
	output, err := cmd.CombinedOutput()
	if err == nil {
		return 0, string(output)
	}
	if exit, ok := err.(*exec.ExitError); ok {
		return exit.ExitCode(), string(output)
	}
	t.Fatal(err)
	return 1, string(output)
}
