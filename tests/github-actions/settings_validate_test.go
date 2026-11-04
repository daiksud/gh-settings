package githubactions

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func TestSettingsPRValidationUsesPinnedConsumer(t *testing.T) {
	w, _ := readWorkflow(t, "../../.github/workflows/settings-pr.yml")
	var trigger map[string][]string
	target := w.On["pull_request"]
	if err := target.Decode(&trigger); err != nil { t.Fatal(err) }
	if len(w.On) != 1 || len(trigger) != 1 || !reflect.DeepEqual(trigger["branches"], []string{"main"}) {
		t.Fatal("the required validator must use default pull_request activities on main")
	}
	j, present := w.Jobs[settingsValidateCheckName]
	if !present || len(w.Jobs) != 1 || len(w.Permissions) != 0 || len(w.Env) != 0 ||
		j.If != "" || j.Needs != "" || j.Environment != "" || j.Continue || j.Runner != "ubuntu-slim" ||
		j.Timeout != 10 || len(j.Env) != 0 ||
		!reflect.DeepEqual(j.Permissions, map[string]string{"contents": "read"}) {
		t.Fatal("required validation must execute without privilege")
	}
	if w.Concurrency.Group != "" || w.Concurrency.Cancel ||
		j.Concurrency.Group != "" || j.Concurrency.Cancel || len(j.Outputs) != 0 {
		t.Fatal("required validation must not override concurrency or publish outputs")
	}
	if len(j.Steps) != 2 || j.Steps[0].Uses != "actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1" ||
		len(j.Steps[0].With) != 0 || len(j.Steps[0].Env) != 0 {
		t.Fatal("required validator must use default merge checkout")
	}
	validate := j.Steps[1]
	if validate.Uses != operationalAction || !reflect.DeepEqual(validate.With, map[string]string{"command": "validate"}) || len(validate.Env) != 0 {
		t.Fatal("required validation must use the fixed published Action")
	}
	for _, step := range j.Steps {
		if step.If != "" || step.Continue || strings.Contains(fmt.Sprint(step), "secrets.") ||
			strings.Contains(fmt.Sprint(step), "create-github-app-token") {
			t.Fatal("required validation cannot skip, suppress failure or mint credentials")
		}
	}
	for _, status := range []int{0, 37} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			f := newCompositeFixture(t)
			f.base["GITHUB_EVENT_NAME"] = "pull_request"
			f.base["GITHUB_WORKFLOW_REF"] = "owner/config/.github/workflows/settings-pr.yml@refs/pull/1/merge"
			f.base["MOCK_VALIDATE_STATUS"] = fmt.Sprint(status)
			r := f.call(t, validate, nil, nil)
			if r.status != status || r.calls != "validate\n" || r.api != "" || r.summary != "" {
				t.Fatalf("required validation failed: %+v", r)
			}
			if status == 0 && r.outputs["config-sha"] != f.sha {
				t.Fatalf("validated SHA missing: %+v", r)
			}
			if status != 0 && len(r.outputs) != 0 {
				t.Fatalf("failed validation published outputs: %+v", r)
			}
		})
	}
}
