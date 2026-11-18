package githubactions

import (
	"os"
	"os/exec"
	"reflect"
	"strings"
	"testing"
)

func TestPublicTopicsProbeRecovery(t *testing.T) {
	command := exec.Command("python3", "test_public_topics_probe.py", "-q")
	command.Env = append(os.Environ(), "PYTHONDONTWRITEBYTECODE=1")
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("bounded Topics probe: %v\n%s", err, output)
	}
}

func TestPublicTopicsProbeWorkflow(t *testing.T) {
	w, source := readWorkflow(t, "../../.github/workflows/verify-public-action.yml")
	j := w.Jobs["topics"]
	if len(w.On) != 1 || w.On["workflow_dispatch"].Tag != "!!null" || len(w.Jobs) != 1 ||
		j.Environment != "gh-settings" || j.Runner != "ubuntu-24.04" || j.Timeout != 15 ||
		j.If != "github.event_name == 'workflow_dispatch' && github.ref == 'refs/heads/main' && github.repository == 'daiksud/gh-settings'" ||
		!reflect.DeepEqual(w.Permissions, map[string]string{"contents": "read"}) || w.Concurrency.Cancel {
		t.Fatal("probe must remain a manually approved main-only job with read-only workflow permissions")
	}
	app := findStep(t, j, "app")
	if app.Uses != "actions/create-github-app-token@64ab729a6294000f5aa4a4bc0b73d1f32bc45510" || !reflect.DeepEqual(app.With, map[string]string{
		"client-id": "${{ vars.ANSWER26_CLIENT_ID }}", "private-key": "${{ secrets.ANSWER26_PRIVATE_KEY }}",
		"owner": "daiksud", "repositories": "test-gh-settings-public", "permission-administration": "write",
	}) {
		t.Fatal("probe token must retain existing credential bindings, narrow target/grant and default revocation")
	}
	commands := []string{}
	for _, s := range j.Steps {
		if s.Continue {
			t.Fatal("probe failures must not be ignored")
		}
		if strings.HasPrefix(s.Uses, "daiksud/gh-settings@") {
			if s.Uses != operationalAction || s.With["prune"] != "" {
				t.Fatal("probe must install the actual reviewed public release without pruning")
			}
			commands = append(commands, s.With["command"])
			if s.With["command"] != "validate" && s.With["token"] != "${{ steps.app.outputs.token }}" {
				t.Fatal("remote operations must use the dedicated App token")
			}
		}
	}
	if !reflect.DeepEqual(commands, []string{"validate", "plan", "apply", "plan", "apply", "apply", "plan"}) ||
		findStep(t, j, "restore-config").If != "always() && steps.prepare.outcome == 'success'" ||
		!strings.Contains(source, "always() && steps.restore-config.outputs.ready == 'true'") {
		t.Fatal("public lifecycle or failure restoration boundary changed")
	}
	for _, id := range []string{"repeat-plan", "final-plan"} {
		if findStep(t, j, id).With["plan-reporting"] != "status-only" {
			t.Fatal("matching plans must expose an assertable no-difference result")
		}
	}
}

func TestPublicTopicsProbeEnvironmentScope(t *testing.T) {
	_, source := readWorkflow(t, "../../.github/workflows/verify-public-action.yml")
	for _, tc := range []struct {
		name, source, path string
		valid              bool
	}{
		{"approved probe", source, publicProbeWorkflowPath, true},
		{"different workflow", source, ".github/workflows/other.yml", false},
		{"different job", strings.Replace(source, "  topics:", "  other:", 1), publicProbeWorkflowPath, false},
		{"extra credential job", source + "  other:\n    environment: gh-settings\n", publicProbeWorkflowPath, false},
		{"automatic trigger", strings.Replace(source, "  workflow_dispatch:", "  push:", 1), publicProbeWorkflowPath, false},
		{"dispatch input", strings.Replace(source, "  workflow_dispatch:", "  workflow_dispatch: {inputs: {target: {type: string}}}", 1), publicProbeWorkflowPath, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			files := settingsValidationFixtureWorkflows()
			files[tc.path] = []byte(tc.source)
			if err := checkSettingsEnvironmentScope(files); (err == nil) != tc.valid {
				t.Fatalf("valid=%t error=%v", tc.valid, err)
			}
		})
	}
}
