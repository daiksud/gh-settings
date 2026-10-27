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
	if !present || j.Runner != "ubuntu-slim" || j.If != "github.event_name == 'pull_request'" || j.Timeout != 10 || j.Environment != "" || j.Needs != "" || j.Continue || len(j.Env) != 0 || !reflect.DeepEqual(j.Permissions, map[string]string{"contents": "read"}) {
		t.Fatal("required settings-plan check cannot acquire privileged credentials or an Environment")
	}
	if len(j.Steps) != 3 || j.Steps[0].Uses != "actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1" || !reflect.DeepEqual(j.Steps[0].With, map[string]string{"ref": "${{ github.sha }}", "persist-credentials": "false"}) || j.Steps[1].Uses != "jdx/mise-action@82a612c5ab8bcda7cc5db6b7e81a8e7584c3dc6e" || !reflect.DeepEqual(j.Steps[1].With, map[string]string{"version": "2026.10.8", "install_args": "go", "cache": "false"}) || j.Steps[2].Run != "go run . validate" {
		t.Fatal("preserve credential-free checkout, pinned Go and local validation")
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
