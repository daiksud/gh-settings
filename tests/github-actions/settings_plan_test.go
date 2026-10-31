package githubactions

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"go.yaml.in/yaml/v4"
)

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
	const consumer = "daiksud/gh-settings@cbc112ae1330b4ad4debe013857bd67d2e1a1e1e"
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
