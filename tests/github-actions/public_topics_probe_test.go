package githubactions

import (
	"os"
	"os/exec"
	"path/filepath"
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
		if strings.Contains(s.Run, `release-ci.py status "$CI_RUN_ID"`) {
			recheckIndex = i
			if s.Env["SOURCE_SHA"] != "${{ github.sha }}" || s.Env["CI_RUN_ID"] != "${{ steps.ci.outputs.ci-run-id }}" || s.Env["GH_TOKEN"] != "${{ github.token }}" {
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
		"name": "release-linux-amd64", "path": "${{ runner.temp }}/topics-candidate-linux",
		"github-token": "${{ github.token }}", "repository": "daiksud/gh-settings",
		"run-id": "${{ steps.ci.outputs.ci-run-id }}", "digest-mismatch": "error",
	}) {
		t.Fatal("candidate download must bind the successful CI run and reject archive digest mismatch")
	}
	candidate := findStep(t, j, "candidate")
	if candidate.Env["SOURCE_SHA"] != "${{ github.sha }}" {
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
	selectCI := findStep(t, w.Jobs["topics"], "ci")
	for _, tc := range []struct {
		name, id, status string
		valid            bool
	}{
		{"successful selected run", "123", "0", true},
		{"missing exact-SHA run", "", "0", false},
		{"failed or unfinished CI", "123", "1", false},
		{"invalid producer ID", "not-a-run", "0", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			source := filepath.Join(dir, ".verification-source")
			if err := os.Mkdir(source, 0o700); err != nil {
				t.Fatal(err)
			}
			git := func(args ...string) string {
				t.Helper()
				cmd := exec.Command("git", args...)
				cmd.Dir = source
				out, err := cmd.CombinedOutput()
				if err != nil {
					t.Fatalf("git: %v %s", err, out)
				}
				return strings.TrimSpace(string(out))
			}
			git("init", "--quiet")
			git("-c", "commit.gpgsign=false", "-c", "user.name=Fixture", "-c", "user.email=fixture@example.invalid", "commit", "--allow-empty", "--quiet", "-m", "fixture")
			bin := filepath.Join(dir, "bin")
			if err := os.Mkdir(bin, 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(bin, "python3"), []byte("#!/bin/sh\nprintf 'ci-run-id=%s\\n' \"$FIXTURE_RUN_ID\" >> \"$GITHUB_OUTPUT\"\nexit \"$FIXTURE_STATUS\"\n"), 0o700); err != nil {
				t.Fatal(err)
			}
			output := filepath.Join(dir, "output")
			cmd := exec.Command("bash", "-euo", "pipefail", "-c", selectCI.Run)
			cmd.Dir = dir
			cmd.Env = append(os.Environ(), "PATH="+bin+string(os.PathListSeparator)+os.Getenv("PATH"), "SOURCE_SHA="+git("rev-parse", "HEAD"), "RUNNER_TEMP="+dir, "GITHUB_OUTPUT="+output, "FIXTURE_RUN_ID="+tc.id, "FIXTURE_STATUS="+tc.status)
			out, err := cmd.CombinedOutput()
			if (err == nil) != tc.valid {
				t.Fatalf("valid=%t err=%v log=%s", tc.valid, err, out)
			}
			data, _ := os.ReadFile(output)
			if tc.valid && string(data) != "ci-run-id=123\n" || !tc.valid && len(data) != 0 {
				t.Fatalf("unexpected producer handoff %q", data)
			}
		})
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
