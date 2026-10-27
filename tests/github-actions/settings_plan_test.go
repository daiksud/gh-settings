package githubactions

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"go.yaml.in/yaml/v4"
)

const planAction = "daiksud/gh-settings@e4a90d281aff8eab3f057eadcdbf609330c014a0"

func TestSettingsPlanConcurrencyIsolation(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(root(), settingsPlanWorkflowPath))
	if err != nil {
		t.Fatal(err)
	}
	var w workflow
	if err := yaml.Unmarshal(data, &w); err != nil {
		t.Fatal(err)
	}
	const group = "settings-plan-${{ github.repository }}-${{ github.event_name }}-${{ github.event.pull_request.number || 'manual' }}"
	if w.Concurrency.Group != group || w.Concurrency.Cancel {
		t.Fatal("PR validation needs per-PR groups separate from non-cancelling manual plans")
	}
	// Render the closed interpolation contract for collision cases, not the
	// GitHub scheduler itself. Actionlint separately checks expression validity.
	groups := make(map[string]bool)
	for _, tc := range []struct{ event, number string }{
		{"pull_request", "455"}, {"pull_request", "459"}, {"workflow_dispatch", "manual"},
	} {
		rendered := strings.NewReplacer(
			"${{ github.repository }}", "owner/repo",
			"${{ github.event_name }}", tc.event,
			"${{ github.event.pull_request.number || 'manual' }}", tc.number,
		).Replace(w.Concurrency.Group)
		if groups[rendered] {
			t.Fatalf("independent PR/manual runs collide in %q", rendered)
		}
		groups[rendered] = true
	}
}

func TestProtectedPlanSourceAndCredentialBoundary(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(root(), settingsPlanWorkflowPath))
	if err != nil {
		t.Fatal(err)
	}
	var w workflow
	if err := yaml.Unmarshal(data, &w); err != nil {
		t.Fatal(err)
	}
	if len(w.On) != 2 || w.On["workflow_dispatch"].Kind != yaml.MappingNode || w.On["pull_request"].Kind != yaml.MappingNode || len(w.Jobs) != 3 || !reflect.DeepEqual(w.Permissions, map[string]string{}) {
		t.Fatal("protected manual plan and required PR validation must have no default token grants")
	}
	var inputs struct {
		Inputs map[string]struct {
			Type     string `yaml:"type"`
			Required bool   `yaml:"required"`
		} `yaml:"inputs"`
	}
	dispatch := w.On["workflow_dispatch"]
	if err := dispatch.Decode(&inputs); err != nil {
		t.Fatal(err)
	}
	if len(inputs.Inputs) != 2 {
		t.Fatal("protected plan cannot accept target, ref, command, or token overrides")
	}
	for _, key := range []string{"pr_number", "head_sha"} {
		if inputs.Inputs[key].Type != "string" || !inputs.Inputs[key].Required {
			t.Fatalf("missing bounded mandatory %s", key)
		}
	}
	required, present := w.Jobs["settings-plan"]
	if !present || required.Runner != "ubuntu-slim" || required.If != "github.event_name == 'pull_request'" ||
		required.Environment != "" || required.Timeout != 10 || required.Needs != "" ||
		!reflect.DeepEqual(required.Permissions, map[string]string{"contents": "read"}) ||
		len(required.Steps) != 3 || !strings.Contains(required.Steps[2].Run, "go run . validate") {
		t.Fatal("required PR plan check must validate locally without privileged credentials")
	}
	gate, present := w.Jobs["activation-gate"]
	if !present || gate.If != "github.event_name == 'workflow_dispatch'" || gate.Runner != "ubuntu-slim" || len(gate.Steps) != 1 || len(gate.Permissions) != 0 ||
		gate.Steps[0].Env["ENABLED"] != "${{ vars.GH_SETTINGS_PLAN_ENABLED }}" ||
		gate.Steps[0].Env["REPOSITORY"] != "${{ github.repository }}" ||
		gate.Steps[0].Env["REF"] != "${{ github.ref }}" ||
		!strings.Contains(gate.Steps[0].Run, "exit 1") {
		t.Fatal("a skipped observer must fail a read-only activation gate rather than report success")
	}
	j, ok := w.Jobs["plan"]
	if !ok || j.Needs != "activation-gate" || j.Environment != settingsPlanEnvironment || j.Runner != "ubuntu-24.04" || j.Timeout <= 0 || j.Timeout > 15 ||
		j.If != "github.event_name == 'workflow_dispatch' && github.ref == 'refs/heads/main' && github.repository == 'daiksud/gh-settings' && vars.GH_SETTINGS_PLAN_ENABLED == 'true'" ||
		!reflect.DeepEqual(j.Permissions, map[string]string{"contents": "read", "pull-requests": "read"}) || j.Continue || len(j.Env) != 0 {
		t.Fatal("plan job must be protected, main-only, inactive by default and read-only")
	}
	if len(j.Steps) != 11 {
		t.Fatalf("expected eleven controlled steps, got %d", len(j.Steps))
	}
	for i, s := range j.Steps {
		if s.Continue || strings.Contains(s.Run, "gh settings apply") || strings.Contains(s.Run, "workflow_dispatch") ||
			strings.Contains(s.Uses, "upload-artifact") || strings.Contains(s.Uses, "download-artifact") {
			t.Fatalf("step %d could bypass validation or execute candidate content", i)
		}
		if strings.Contains(s.Run, "secrets.") || strings.Contains(s.Run, "steps.app.outputs.token") ||
			strings.Contains(s.Run, "git checkout") {
			t.Fatalf("step %d has an unsafe shell credential or candidate code path", i)
		}
	}
	checkout, published, build, prepare, validate, token, plan, verify, summary, cleanup := j.Steps[0], j.Steps[1], j.Steps[3], j.Steps[4], j.Steps[5], j.Steps[6], j.Steps[7], j.Steps[8], j.Steps[9], j.Steps[10]
	if checkout.Uses != "actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1" ||
		checkout.With["ref"] != "${{ github.sha }}" || checkout.With["path"] != ".trusted" ||
		checkout.With["persist-credentials"] != "false" || !strings.Contains(build.Run, "./scripts/pr-settings") {
		t.Fatal("preparation must build and use reviewed main code, never candidate code")
	}
	if published.Uses != "actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1" ||
		published.With["repository"] != "daiksud/gh-settings" ||
		published.With["ref"] != "e4a90d281aff8eab3f057eadcdbf609330c014a0" ||
		published.With["path"] != ".published" ||
		published.With["persist-credentials"] != "false" {
		t.Fatal("private validation installer must come from the exact published release revision")
	}
	if prepare.ID != "candidate" || prepare.Env["GH_TOKEN"] != "${{ github.token }}" ||
		prepare.Env["PR_NUMBER"] != "${{ inputs.pr_number }}" || prepare.Env["EXPECTED_HEAD"] != "${{ inputs.head_sha }}" ||
		!strings.Contains(prepare.Run, "prepare \"$PR_NUMBER\" \"$EXPECTED_HEAD\"") {
		t.Fatal("candidate preparation must validate exact identifiers before credential issuance")
	}
	if validate.Uses != "" || len(validate.With) != 0 || validate.Shell != "bash" ||
		!strings.Contains(validate.Run, ".published/action/install-cli.sh") ||
		!strings.Contains(validate.Run, "gh-settings v0.18.1") ||
		!strings.Contains(validate.Run, `validate >/dev/null 2>&1`) ||
		!strings.Contains(validate.Run, `action metadata >> "$GITHUB_OUTPUT" 2>/dev/null`) ||
		!strings.Contains(validate.Run, "unset GH_TOKEN GITHUB_TOKEN GH_DEBUG") ||
		validate.ID != "validation" ||
		!strings.Contains(validate.Run, "printf 'binary-path=%s") {
		t.Fatal("checksum-verified published CLI must validate privately before App token issuance")
	}
	if token.Uses != "actions/create-github-app-token@64ab729a6294000f5aa4a4bc0b73d1f32bc45510" || token.ID != "app" ||
		token.With["owner"] != "${{ github.repository_owner }}" || token.With["repositories"] != "${{ github.event.repository.name }}" ||
		token.With["permission-administration"] != "write" || token.With["permission-issues"] != "" ||
		token.With["client-id"] != "${{ vars.ANSWER26_CLIENT_ID }}" || token.With["private-key"] != "${{ secrets.ANSWER26_PRIVATE_KEY }}" {
		t.Fatal("plan token must be separately protected, repo-scoped and without unrelated grants")
	}
	if plan.Uses != "" || plan.ID != "observation" || len(plan.With) != 0 || plan.Shell != "bash" ||
		plan.Env["SETTINGS_CLI"] != "${{ steps.validation.outputs.binary-path }}" ||
		plan.Env["GH_TOKEN"] != "${{ steps.app.outputs.token }}" ||
		!strings.Contains(plan.Run, `action plan "${SETTINGS_CLI%/*}/plan.json" >/dev/null 2>&1`) ||
		!strings.Contains(plan.Run, "unset GITHUB_TOKEN GH_DEBUG") ||
		!strings.Contains(plan.Run, "settings observation failed") ||
		!strings.Contains(plan.Run, `printf 'exit-status=%s\n' "$status" >> "$GITHUB_OUTPUT"`) ||
		strings.Contains(plan.Run, "GITHUB_STEP_SUMMARY") ||
		strings.Contains(plan.Run, "action summary") {
		t.Fatal("plan must fail honestly without logging raw observation diagnostics")
	}
	if verify.Env["GH_TOKEN"] != "${{ github.token }}" || !strings.Contains(verify.Run, "verify \"$PR_NUMBER\" \"$EXPECTED_HEAD\"") ||
		summary.Env["GH_TOKEN"] != "" || summary.Env["SETTINGS_CLI"] != "${{ steps.validation.outputs.binary-path }}" ||
		summary.Env["SETTINGS_ACTION_REF"] != "e4a90d281aff8eab3f057eadcdbf609330c014a0" ||
		summary.Env["PR_NUMBER"] != "${{ steps.candidate.outputs.pr-number }}" ||
		summary.Env["HEAD_SHA"] != "${{ steps.candidate.outputs.head-sha }}" ||
		summary.Env["BASE_SHA"] != "${{ steps.candidate.outputs.base-sha }}" ||
		summary.Env["CONFIG_SHA256"] != "${{ steps.candidate.outputs.config-sha256 }}" ||
		summary.Env["PLAN_STATUS"] != "${{ steps.observation.outputs.exit-status }}" ||
		!strings.Contains(summary.Run, "Protected settings observation") ||
		!strings.Contains(summary.Run, `action summary "${SETTINGS_CLI%/*}/plan.json" "$GITHUB_STEP_SUMMARY" >/dev/null 2>&1`) ||
		cleanup.If != "always()" || cleanup.Env["VALIDATION_CLI"] != "${{ steps.validation.outputs.binary-path }}" ||
		!strings.Contains(cleanup.Run, "GITHUB_WORKSPACE/.github") || !strings.Contains(cleanup.Run, "gh-settings-cli.") {
		t.Fatal("freshness verification, declared-only Summary and owned sensitive cleanup must remain mandatory")
	}
}

func TestProtectedPlanGuardRejectsUnexpectedPlacement(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(root(), settingsPlanWorkflowPath))
	if err != nil {
		t.Fatal(err)
	}
	base := map[string][]byte{
		settingsWorkflowPath:      []byte("on: {push: {branches: [main]}}\njobs:\n  apply:\n    environment: gh-settings\n"),
		settingsGuardWorkflowPath: []byte("jobs:\n  settings-env:\n    name: settings-env\n"),
		settingsPlanWorkflowPath:  data,
	}
	for _, tc := range []struct {
		name, path, content string
	}{
		{"untrusted PR target event", settingsPlanWorkflowPath, strings.Replace(string(data), "  workflow_dispatch:", "  pull_request_target: {}\n  workflow_dispatch:", 1)},
		{"push event", settingsPlanWorkflowPath, strings.Replace(string(data), "  workflow_dispatch:", "  push:\n  workflow_dispatch:", 1)},
		{"wrong main gate", settingsPlanWorkflowPath, strings.Replace(string(data), "github.ref == 'refs/heads/main'", "github.ref == 'refs/heads/dev'", 1)},
		{"missing activation", settingsPlanWorkflowPath, strings.Replace(string(data), " && vars.GH_SETTINGS_PLAN_ENABLED == 'true'", "", 1)},
		{"wrong environment", settingsPlanWorkflowPath, strings.Replace(string(data), "environment: gh-settings-plan", "environment: gh-settings", 1)},
		{"duplicate environment", settingsPlanWorkflowPath, strings.Replace(string(data), "environment: gh-settings-plan", "environment: gh-settings-plan\n    environment: preview", 1)},
		{"no environment", settingsPlanWorkflowPath, strings.Replace(string(data), "    environment: gh-settings-plan\n", "", 1)},
		{"untrusted job name", ".github/workflows/other.yml", "jobs:\n  settings-plan:\n    runs-on: ubuntu-24.04\n"},
		{"untrusted environment", ".github/workflows/other.yml", "jobs:\n  unrelated:\n    environment: gh-settings-plan\n"},
		{"reusable workflow", settingsPlanWorkflowPath, strings.Replace(string(data), "  plan:\n", "  delegate:\n    uses: owner/repo/.github/workflows/call.yml@main\n  plan:\n", 1)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			files := make(map[string][]byte, len(base)+1)
			for path, content := range base {
				files[path] = content
			}
			files[tc.path] = []byte(tc.content)
			if err := checkSettingsEnvironmentScope(files); err == nil {
				t.Fatal("expected trusted guard to reject unsafe environment or trigger")
			}
		})
	}
}

func TestProtectedPlanValidationDoesNotLogUntrustedDiagnostics(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(root(), settingsPlanWorkflowPath))
	if err != nil {
		t.Fatal(err)
	}
	var w workflow
	if err := yaml.Unmarshal(data, &w); err != nil {
		t.Fatal(err)
	}
	step := w.Jobs["plan"].Steps[5]
	for _, tc := range []struct {
		name, status string
		wantSuccess  bool
	}{
		{"valid candidate", "0", true},
		{"invalid candidate", "3", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			installDir := filepath.Join(root, ".published", "action")
			binDir := filepath.Join(root, "gh-settings-cli.fake")
			if err := os.MkdirAll(installDir, 0700); err != nil {
				t.Fatal(err)
			}
			if err := os.MkdirAll(binDir, 0700); err != nil {
				t.Fatal(err)
			}
			bin := filepath.Join(binDir, "gh-settings")
			installed := `#!/bin/bash
if [[ "$1" == --version ]]; then
  echo 'gh-settings v0.18.1'
elif [[ "$1" == validate ]]; then
  echo 'SENSITIVE_CANDIDATE_VALUE' >&2
  exit "$VALIDATE_STATUS"
elif [[ "$1 $2" == 'action metadata' ]]; then
  printf 'config-sha=%040d\ntarget=daiksud/gh-settings\ncli-version=v0.18.1\ncli-sha256=%064d\n' 0 0
else
  exit 44
fi
`
			installer := `#!/bin/bash
echo 'INSTALLER_SENSITIVE_ERROR' >&2
printf '%s\n' "$RUNNER_TEMP/gh-settings-cli.fake/gh-settings"
`
			if err := os.WriteFile(bin, []byte(installed), 0700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(installDir, "install-cli.sh"), []byte(installer), 0700); err != nil {
				t.Fatal(err)
			}
			cmd := exec.Command("bash", "-e", "-o", "pipefail", "-c", step.Run)
			cmd.Env = append(os.Environ(), "GITHUB_WORKSPACE="+root, "RUNNER_TEMP="+root, "VALIDATE_STATUS="+tc.status, "GITHUB_OUTPUT="+filepath.Join(root, "validation-outputs"))
			var stdout, stderr bytes.Buffer
			cmd.Stdout, cmd.Stderr = &stdout, &stderr
			err := cmd.Run()
			if (err == nil) != tc.wantSuccess {
				t.Fatalf("unexpected validation status: %v", err)
			}
			text := stdout.String() + stderr.String()
			if strings.Contains(text, "SENSITIVE_CANDIDATE_VALUE") || strings.Contains(text, "INSTALLER_SENSITIVE_ERROR") {
				t.Fatalf("candidate data or installer diagnostics leaked: %q", text)
			}
			if tc.wantSuccess && text != "" {
				t.Fatalf("private success must not log candidate content: %q", text)
			}
			if !tc.wantSuccess && !strings.Contains(text, "candidate settings validation failed") {
				t.Fatalf("expected only generic validation failure: %q", text)
			}
			output, err := os.ReadFile(filepath.Join(root, "validation-outputs"))
			wantOutput := "binary-path=" + bin + "\n"
			if tc.wantSuccess {
				wantOutput += "config-sha=" + strings.Repeat("0", 40) + "\ntarget=daiksud/gh-settings\ncli-version=v0.18.1\ncli-sha256=" + strings.Repeat("0", 64) + "\n"
			}
			if err != nil || string(output) != wantOutput {
				t.Fatalf("published CLI path was not passed privately to the plan: %q, %v", output, err)
			}
		})
	}
}

func TestProtectedPlanReportsHonestDeclaredOnlyOutcomes(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(root(), settingsPlanWorkflowPath))
	if err != nil {
		t.Fatal(err)
	}
	var w workflow
	if err := yaml.Unmarshal(data, &w); err != nil {
		t.Fatal(err)
	}
	plan, verify, summary := w.Jobs["plan"].Steps[7], w.Jobs["plan"].Steps[8], w.Jobs["plan"].Steps[9]
	if plan.Env["GH_TOKEN"] != "${{ steps.app.outputs.token }}" || plan.Env["SETTINGS_CLI"] != "${{ steps.validation.outputs.binary-path }}" {
		t.Fatal("protected plan must use the published binary with narrowly scoped credentials")
	}
	run := func(script string, env map[string]string) (string, error) {
		cmd := exec.Command("bash", "-e", "-o", "pipefail", "-c", script)
		cmd.Env = os.Environ()
		for name, value := range env {
			cmd.Env = append(cmd.Env, name+"="+value)
		}
		var stdout, stderr bytes.Buffer
		cmd.Stdout, cmd.Stderr = &stdout, &stderr
		err := cmd.Run()
		return stdout.String() + stderr.String(), err
	}
	for _, tc := range []struct {
		name, outcome, summaryText              string
		planStatus, verifyStatus, summaryStatus int
		reportOnFailure                         bool
		wantSummary, wantSuccess                bool
	}{
		{name: "declared difference", outcome: "changed", summaryText: "settings.repository.has_issues: false → true", wantSummary: true, wantSuccess: true},
		{name: "no difference", outcome: "no-difference", summaryText: "No differences for managed settings.", wantSummary: true, wantSuccess: true},
		{name: "API command failure", planStatus: 17},
		{name: "read failure report", outcome: "read-failure", summaryText: "Error — plan failed.", planStatus: 1, summaryStatus: 1, reportOnFailure: true, wantSummary: true},
		{name: "partial observation", outcome: "read-failure", summaryText: "Error — plan failed: incomplete observation.", planStatus: 1, summaryStatus: 1, reportOnFailure: true, wantSummary: true},
		{name: "stale candidate", outcome: "changed", verifyStatus: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			bin := filepath.Join(dir, "gh-settings")
			script := `#!/bin/bash
case "$1 $2" in
  'action plan')
    printf 'CANARY-unmanaged-remote-value\n' >&2
    printf 'CANARY-token=%s\n' "${GH_TOKEN-}"
    if [[ "$PLAN_STATUS" != 0 && "$REPORT_ON_FAILURE" != true ]]; then exit "$PLAN_STATUS"; fi
    printf '{"outcome":"%s"}\n' "$PLAN_OUTCOME" > "$3"
    exit "$PLAN_STATUS"
    ;;
  'action summary')
    printf 'CANARY-summary-internal\n' >&2
    printf '%s\n' "$SUMMARY_TEXT" >> "$4"
    exit "$SUMMARY_STATUS"
    ;;
  *) exit 79 ;;
esac
`
			if err := os.WriteFile(bin, []byte(script), 0700); err != nil {
				t.Fatal(err)
			}
			planOutput := filepath.Join(dir, "plan-output")
			planPublic, planErr := run(plan.Run, map[string]string{
				"SETTINGS_CLI": bin, "GH_TOKEN": "CANARY-credential", "PLAN_STATUS": strconv.Itoa(tc.planStatus), "PLAN_OUTCOME": tc.outcome,
				"REPORT_ON_FAILURE": strconv.FormatBool(tc.reportOnFailure), "GITHUB_OUTPUT": planOutput,
			})
			if tc.planStatus != 0 && !tc.reportOnFailure {
				if planErr == nil || !strings.Contains(planPublic, "settings observation failed") || strings.Contains(planPublic, "CANARY") {
					t.Fatalf("API failure was not a generic failed outcome: %v, %q", planErr, planPublic)
				}
				return
			}
			if planErr != nil || planPublic != "" {
				t.Fatalf("successful observation leaked diagnostics: %v, %q", planErr, planPublic)
			}
			output, err := os.ReadFile(planOutput)
			if err != nil || string(output) != "exit-status="+strconv.Itoa(tc.planStatus)+"\n" {
				t.Fatalf("plan status was not preserved for final reporting: %q, %v", output, err)
			}
			reportPath := filepath.Join(dir, "plan.json")
			info, err := os.Stat(reportPath)
			if err != nil || info.Mode().Perm() != 0600 {
				t.Fatalf("plan report must remain owner-readable: %v, %v", info, err)
			}

			helper := filepath.Join(dir, "pr-settings")
			if err := os.WriteFile(helper, []byte("#!/bin/bash\nif [[ \"$VERIFY_STATUS\" != 0 ]]; then echo 'pr-settings: preparation or freshness verification failed' >&2; fi\nexit \"$VERIFY_STATUS\"\n"), 0700); err != nil {
				t.Fatal(err)
			}
			verifyPublic, verifyErr := run(verify.Run, map[string]string{
				"RUNNER_TEMP": dir, "GH_TOKEN": "ordinary-token", "PR_NUMBER": "12", "EXPECTED_HEAD": strings.Repeat("a", 40), "VERIFY_STATUS": strconv.Itoa(tc.verifyStatus),
			})
			if tc.verifyStatus != 0 {
				if verifyErr == nil || !strings.Contains(verifyPublic, "freshness verification failed") {
					t.Fatalf("stale candidate did not fail freshness verification: %v, %q", verifyErr, verifyPublic)
				}
				return
			}
			if verifyErr != nil || verifyPublic != "" {
				t.Fatalf("fresh candidate verification failed: %v, %q", verifyErr, verifyPublic)
			}

			summaryPath := filepath.Join(dir, "summary")
			summaryPublic, summaryErr := run(summary.Run, map[string]string{
				"SETTINGS_CLI": bin, "GITHUB_STEP_SUMMARY": summaryPath, "SUMMARY_STATUS": strconv.Itoa(tc.summaryStatus), "SUMMARY_TEXT": tc.summaryText,
				"PLAN_STATUS": strconv.Itoa(tc.planStatus),
				"PR_NUMBER":   "12", "HEAD_SHA": strings.Repeat("a", 40), "BASE_SHA": strings.Repeat("b", 40),
				"SETTINGS_CONFIG_SHA": strings.Repeat("c", 40), "CONFIG_SHA256": strings.Repeat("d", 64),
			})
			if (summaryErr == nil) != tc.wantSuccess || strings.Contains(summaryPublic, "CANARY") {
				t.Fatalf("unexpected Summary outcome or leaked diagnostic: %v, %q", summaryErr, summaryPublic)
			}
			if tc.summaryStatus != 0 && !strings.Contains(summaryPublic, "settings observation report failed") {
				t.Fatalf("failed observation did not return a generic failure: %q", summaryPublic)
			}
			content, err := os.ReadFile(summaryPath)
			if !tc.wantSummary || err != nil || !strings.Contains(string(content), "Pull request: #12") ||
				!strings.Contains(string(content), strings.Repeat("a", 40)) || !strings.Contains(string(content), tc.summaryText) ||
				strings.Contains(string(content), "CANARY-unmanaged") || strings.Contains(string(content), "CANARY-credential") {
				t.Fatalf("Summary did not identify the fresh declared-only result: %q, %v", content, err)
			}
		})
	}
}
