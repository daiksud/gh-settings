package githubactions

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"go.yaml.in/yaml/v4"
)

func TestSettingsPRValidationUsesTrustedConsumer(t *testing.T) {
	w, _ := readWorkflow(t, "../../.github/workflows/settings-pr.yml")
	var trigger map[string][]string
	target := w.On["pull_request_target"]
	if err := target.Decode(&trigger); err != nil {
		t.Fatal(err)
	}
	if len(w.On) != 1 || len(trigger) != 2 || !reflect.DeepEqual(trigger["branches"], []string{"main"}) ||
		!reflect.DeepEqual(trigger["types"], []string{"opened", "synchronize", "reopened", "edited"}) {
		t.Fatal("the required validator must run for every main PR update, without path filtering")
	}
	j, present := w.Jobs[settingsPlanCheckName]
	if !present || len(w.Jobs) != 2 || len(w.Permissions) != 0 || len(w.Env) != 0 || w.Defaults.Run.Shell != "bash" ||
		j.If != "" || j.Needs != "" || j.Environment != "" || j.Continue || j.Runner != "ubuntu-24.04" || j.Timeout != 10 || len(j.Env) != 0 ||
		!reflect.DeepEqual(j.Permissions, map[string]string{"contents": "read"}) {
		t.Fatal("required validation must execute independently without an Environment or write credential")
	}
	if w.Concurrency.Group != "" || w.Concurrency.Cancel || j.Concurrency.Cancel ||
		j.Concurrency.Group != "settings-pr-validation-${{ github.repository }}-${{ github.event.pull_request.number }}" {
		t.Fatal("new validation must have its own non-cancelling group while the old validator remains")
	}
	if !reflect.DeepEqual(j.Outputs, map[string]string{"plan-required": "${{ steps.candidate.outputs.plan-required }}"}) {
		t.Fatal("only path eligibility may cross from validation to privileged plan")
	}
	if len(j.Steps) != 3 || j.Steps[0].Uses != "actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1" ||
		!reflect.DeepEqual(j.Steps[0].With, map[string]string{
			"ref": "${{ format('refs/pull/{0}/merge', github.event.pull_request.number) }}", "fetch-depth": "2",
			"persist-credentials": "false", "allow-unsafe-pr-checkout": "true",
		}) || len(j.Steps[0].Env) != 0 {
		t.Fatal("validate the explicit PR merge as data with its parents available and no persisted credentials")
	}
	prepare, validate := j.Steps[1], j.Steps[2]
	if prepare.ID != "candidate" || prepare.Uses != "" || prepare.Run == "" ||
		!reflect.DeepEqual(prepare.Env, map[string]string{
			"EXPECTED_HEAD": "${{ github.event.pull_request.head.sha }}", "EXPECTED_BASE": "${{ github.event.pull_request.base.sha }}",
		}) || validate.Uses != operationalAction || !reflect.DeepEqual(validate.With, map[string]string{"command": "validate"}) || len(validate.Env) != 0 {
		t.Fatal("verify event identity before the pinned credential-free validator")
	}
	for _, step := range j.Steps {
		if step.If != "" || step.Continue || strings.Contains(fmt.Sprint(step), "secrets.") || strings.Contains(fmt.Sprint(step), "create-github-app-token") {
			t.Fatal("required validation cannot skip, ignore failures or mint privileged credentials")
		}
	}
	for _, status := range []int{0, 37} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			f := newCompositeFixture(t)
			f.base["GITHUB_EVENT_NAME"] = "pull_request_target"
			f.base["GITHUB_WORKFLOW_REF"] = "owner/config/.github/workflows/settings-pr.yml@refs/heads/main"
			f.base["MOCK_VALIDATE_STATUS"] = fmt.Sprint(status)
			r := f.call(t, validate, nil, nil)
			if r.status != status || r.calls != "validate\n" || r.api != "" || r.summary != "" {
				t.Fatalf("required validation failed to preserve its result without API work: %+v", r)
			}
			if status == 0 && r.outputs["config-sha"] != f.sha {
				t.Fatalf("validated configuration identity missing: %+v", r)
			}
			if status != 0 && len(r.outputs) != 0 {
				t.Fatalf("failed validation published success outputs: %+v", r)
			}
		})
	}
}

func TestSettingsPlanRemainsUnprivilegedPRValidation(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(root(), settingsPlanWorkflowPath))
	if err != nil {
		t.Fatal(err)
	}
	var w workflow
	if err := yaml.Unmarshal(data, &w); err != nil {
		t.Fatal(err)
	}
	if len(w.On) != 1 || w.On["pull_request"].Kind != yaml.MappingNode || len(w.Jobs) != 1 || len(w.Permissions) != 0 {
		t.Fatal("settings-plan must contain only unprivileged PR validation")
	}
	j, present := w.Jobs[settingsPlanCheckName]
	if !present || j.Runner != "ubuntu-24.04" || j.If != "github.event_name == 'pull_request'" || j.Timeout != 10 || j.Environment != "" || j.Needs != "" || j.Continue || len(j.Env) != 0 || !reflect.DeepEqual(j.Permissions, map[string]string{"contents": "read"}) {
		t.Fatal("settings-plan must preserve its read-only PR boundary and use the supported validator runner")
	}
	const consumer = "daiksud/gh-settings@6dbdbb3ab11cbc0ef257fef089d6915d31d003d6"
	if len(j.Steps) != 2 || j.Steps[0].Uses != "actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1" || !reflect.DeepEqual(j.Steps[0].With, map[string]string{"ref": "${{ github.sha }}", "persist-credentials": "false"}) || j.Steps[1].Uses != consumer || !reflect.DeepEqual(j.Steps[1].With, map[string]string{"command": "validate"}) || j.Steps[1].If != "" {
		t.Fatal("validate the merge declaration with its pinned operational consumer, without a token or conditional skip")
	}
	for _, caller := range []struct{ file, job, command, action string }{
		{"settings-pr.yml", "plan", "plan", operationalPlanAction},
		{"settings-main.yml", "apply", "apply", operationalAction},
	} {
		data, err := os.ReadFile(filepath.Join(root(), ".github/workflows", caller.file))
		if err != nil {
			t.Fatal(err)
		}
		var operational workflow
		if err := yaml.Unmarshal(data, &operational); err != nil {
			t.Fatal(err)
		}
		matches := 0
		for _, step := range operational.Jobs[caller.job].Steps {
			if strings.HasPrefix(step.Uses, "daiksud/gh-settings@") {
				if step.Uses != caller.action || step.With["command"] != caller.command {
					t.Fatal("operational consumers must use their declared immutable package")
				}
				matches++
			}
		}
		if matches != 1 {
			t.Fatalf("expected one pinned consumer in %s", caller.file)
		}
	}
	for _, step := range j.Steps {
		if step.Continue || len(step.Env) != 0 {
			t.Fatal("validation steps cannot inject credentials or ignore failures")
		}
	}
	for _, retired := range []string{"workflow_dispatch", "GH_SETTINGS_PLAN_ENABLED", "gh-settings-plan", "create-github-app-token", "secrets.", "scripts/pr-settings"} {
		if strings.Contains(string(data), retired) {
			t.Fatalf("retired manual path remains: %s", retired)
		}
	}
	const group = "settings-plan-${{ github.repository }}-${{ github.event.pull_request.number }}"
	if w.Concurrency.Group != group || w.Concurrency.Cancel {
		t.Fatal("preserve independent non-cancelling PR validation groups")
	}
	first := strings.Replace(group, "${{ github.event.pull_request.number }}", "471", 1)
	second := strings.Replace(group, "${{ github.event.pull_request.number }}", "472", 1)
	if first == second {
		t.Fatal("unrelated PR validation cannot share a concurrency group")
	}
}
