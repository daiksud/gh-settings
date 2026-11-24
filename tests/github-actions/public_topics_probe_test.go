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
		j.Environment != "gh-settings" || j.Runner != "ubuntu-24.04" || j.Timeout != 15 || len(j.Env) != 0 ||
		j.If != "github.event_name == 'workflow_dispatch' && github.ref == 'refs/heads/main' && github.repository == 'daiksud/gh-settings'" ||
		!reflect.DeepEqual(w.Permissions, map[string]string{"contents": "read", "actions": "read"}) || w.Concurrency.Cancel {
		t.Fatal("probe must remain a manually approved main-only job with read-only workflow permissions")
	}
	app := findStep(t, j, "app")
	if !strings.HasPrefix(app.Uses, "actions/create-github-app-token@") || !reflect.DeepEqual(app.With, map[string]string{
		"client-id": "${{ vars.ANSWER26_CLIENT_ID }}", "private-key": "${{ secrets.ANSWER26_PRIVATE_KEY }}",
		"owner": "daiksud", "repositories": "test-gh-settings-public", "permission-administration": "write",
	}) {
		t.Fatal("probe token must retain existing credential bindings, narrow target/grant and default revocation")
	}
	commands := []string{}
	ciIndex, candidateIndex, recheckIndex, appIndex := -1, -1, -1, -1
	for i, s := range j.Steps {
		switch s.ID {
		case "ci":
			ciIndex = i
		case "candidate":
			candidateIndex = i
		case "app":
			appIndex = i
		}
		if strings.Contains(s.Run, `release-ci.py status "$CI_RUN_ID" "$CI_RUN_ATTEMPT"`) {
			recheckIndex = i
			if s.Env["SOURCE_SHA"] != "${{ github.sha }}" || s.Env["CI_RUN_ID"] != "${{ steps.ci.outputs.ci-run-id }}" || s.Env["CI_RUN_ATTEMPT"] != "${{ steps.ci.outputs.ci-run-attempt }}" || s.Env["GH_TOKEN"] != "${{ github.token }}" {
				t.Fatal("producer recheck must bind the selected CI before the live token is minted")
			}
		}
		if s.Continue {
			t.Fatal("probe failures must not be ignored")
		}
		if s.Uses == "./.verification-source" {
			if s.With["prune"] != "" {
				t.Fatal("candidate probe must not select pruning")
			}
			commands = append(commands, s.With["command"])
			if s.With["command"] != "validate" && s.With["token"] != "${{ steps.app.outputs.token }}" {
				t.Fatal("remote operations must use the dedicated App token")
			}
		}
	}
	if ciIndex < 0 || candidateIndex <= ciIndex || recheckIndex <= candidateIndex || appIndex <= recheckIndex {
		t.Fatal("CI, product and producer recheck gates must precede live credential issuance")
	}
	if j.Steps[0].With["ref"] != "${{ github.sha }}" || j.Steps[0].With["path"] != ".verification-source" || j.Steps[0].With["persist-credentials"] != "false" {
		t.Fatal("candidate source must be the exact dispatched SHA without persisted credentials")
	}
	for _, forbidden := range []string{"uses: daiksud/gh-settings@", "candidate-plan.sh", "http_unix_socket", "plan-fixture"} {
		if strings.Contains(source, forbidden) {
			t.Fatalf("candidate probe must not substitute public release or API fixtures: %s", forbidden)
		}
	}
	var download step
	for _, s := range j.Steps {
		if strings.HasPrefix(s.Uses, "actions/download-artifact@") {
			download = s
		}
	}
	if download.Uses != "actions/download-artifact@9000827ccba6bdab643e8b6fd33ac0654aef8333" || !reflect.DeepEqual(download.With, map[string]string{
		"artifact-ids": "${{ steps.ci.outputs.linux-id }}", "path": "${{ runner.temp }}/topics-candidate-linux",
		"github-token": "${{ github.token }}", "repository": "daiksud/gh-settings",
		"run-id": "${{ steps.ci.outputs.ci-run-id }}", "digest-mismatch": "error",
	}) {
		t.Fatal("candidate download must bind the successful CI run and reject archive digest mismatch")
	}
	candidate := findStep(t, j, "candidate")
	if candidate.Env["SOURCE_SHA"] != "${{ github.sha }}" || candidate.Env["VERSION"] != "${{ steps.ci.outputs.version }}" {
		t.Fatal("manifest must bind the candidate product to the exact Action and workflow source")
	}
	for _, required := range []string{"release-artifact.sh\" verify linux-amd64", "prepare-candidate-download.sh", "git -C \"$source\" rev-parse HEAD", "git -C \"$source\" status --porcelain"} {
		if !strings.Contains(candidate.Run, required) {
			t.Fatalf("candidate identity/installer gate is missing %q", required)
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

func TestCandidateProbeRequiresCIWithoutFallback(t *testing.T) {
	w, _ := readWorkflow(t, "../../.github/workflows/verify-public-action.yml")
	selection := findStep(t, w.Jobs["topics"], "ci")
	if !strings.HasSuffix(strings.TrimSpace(selection.Run), "python3 .github/scripts/release-ci.py probe") || selection.Continue {
		t.Fatal("the live probe must stop when an inspected exact-SHA deployment is unavailable")
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
