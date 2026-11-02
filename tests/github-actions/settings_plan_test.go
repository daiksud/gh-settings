package githubactions

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
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
		t.Fatal("validation must have its own non-cancelling group")
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
