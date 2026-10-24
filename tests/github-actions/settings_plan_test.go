package githubactions

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"go.yaml.in/yaml/v4"
)

const planAction = "daiksud/gh-settings@6d3146b30860692f84dc54e32d746e6fbbb55c66"

func TestProtectedPlanSourceAndCredentialBoundary(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(root(), settingsPlanWorkflowPath))
	if err != nil {
		t.Fatal(err)
	}
	var w workflow
	if err := yaml.Unmarshal(data, &w); err != nil {
		t.Fatal(err)
	}
	if len(w.On) != 1 || w.On["workflow_dispatch"].Kind != yaml.MappingNode || len(w.Jobs) != 2 || !reflect.DeepEqual(w.Permissions, map[string]string{}) {
		t.Fatal("protected plan must use only manual dispatch and no default token grants")
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
	gate, present := w.Jobs["activation-gate"]
	if !present || gate.Runner != "ubuntu-slim" || len(gate.Steps) != 1 || len(gate.Permissions) != 0 ||
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
	if len(j.Steps) != 10 {
		t.Fatalf("expected ten controlled steps, got %d", len(j.Steps))
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
	checkout, published, build, prepare, validate, token, plan, verify, cleanup := j.Steps[0], j.Steps[1], j.Steps[3], j.Steps[4], j.Steps[5], j.Steps[6], j.Steps[7], j.Steps[8], j.Steps[9]
	if checkout.Uses != "actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1" ||
		checkout.With["ref"] != "${{ github.sha }}" || checkout.With["path"] != ".trusted" ||
		checkout.With["persist-credentials"] != "false" || !strings.Contains(build.Run, "./scripts/pr-settings") {
		t.Fatal("preparation must build and use reviewed main code, never candidate code")
	}
	if published.Uses != "actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1" ||
		published.With["repository"] != "daiksud/gh-settings" ||
		published.With["ref"] != "6d3146b30860692f84dc54e32d746e6fbbb55c66" ||
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
		!strings.Contains(validate.Run, "gh-settings v0.18.0") ||
		!strings.Contains(validate.Run, `validate >/dev/null 2>&1`) ||
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
	if plan.Uses != "" || plan.ID != "" || len(plan.With) != 0 || plan.Shell != "bash" ||
		plan.Env["SETTINGS_CLI"] != "${{ steps.validation.outputs.binary-path }}" ||
		plan.Env["GH_TOKEN"] != "${{ steps.app.outputs.token }}" ||
		!strings.Contains(plan.Run, `action plan "${SETTINGS_CLI%/*}/plan.json" status-only >/dev/null 2>&1`) ||
		!strings.Contains(plan.Run, "unset GITHUB_TOKEN GH_DEBUG") ||
		strings.Contains(plan.Run, "GITHUB_STEP_SUMMARY") || strings.Contains(plan.Run, "GITHUB_OUTPUT") ||
		strings.Contains(plan.Run, "action summary") {
		t.Fatal("private plan must not emit an outcome/count or an Action Summary")
	}
	if verify.Env["GH_TOKEN"] != "${{ github.token }}" || !strings.Contains(verify.Run, "verify \"$PR_NUMBER\" \"$EXPECTED_HEAD\"") ||
		cleanup.If != "always()" || cleanup.Env["VALIDATION_CLI"] != "${{ steps.validation.outputs.binary-path }}" ||
		!strings.Contains(cleanup.Run, "GITHUB_WORKSPACE/.github") || !strings.Contains(cleanup.Run, "gh-settings-cli.") {
		t.Fatal("freshness verification and owned sensitive cleanup must remain mandatory")
	}
}

func TestProtectedPlanGuardRejectsUnexpectedPlacement(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(root(), settingsPlanWorkflowPath))
	if err != nil {
		t.Fatal(err)
	}
	base := map[string][]byte{
		settingsWorkflowPath:      []byte("on: {push: {branches: [main]}}\njobs:\n  apply:\n    environment: gh-settings\n"),
		settingsGuardWorkflowPath: []byte("jobs:\n  gh-settings:\n    name: gh-settings\n"),
		settingsPlanWorkflowPath:  data,
	}
	for _, tc := range []struct {
		name, path, content string
	}{
		{"PR event", settingsPlanWorkflowPath, strings.Replace(string(data), "  workflow_dispatch:", "  pull_request:\n  workflow_dispatch:", 1)},
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
  echo 'gh-settings v0.18.0'
elif [[ "$1" == validate ]]; then
  echo 'SENSITIVE_CANDIDATE_VALUE' >&2
  exit "$VALIDATE_STATUS"
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
			if err != nil || string(output) != "binary-path="+bin+"\n" {
				t.Fatalf("published CLI path was not passed privately to the plan: %q, %v", output, err)
			}
		})
	}
}

func TestProtectedPlanHidesPolicyEqualityFromPublicSurfaces(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(root(), settingsPlanWorkflowPath))
	if err != nil {
		t.Fatal(err)
	}
	var w workflow
	if err := yaml.Unmarshal(data, &w); err != nil {
		t.Fatal(err)
	}
	step := w.Jobs["plan"].Steps[7]
	if step.Env["GH_TOKEN"] != "${{ steps.app.outputs.token }}" || step.Env["SETTINGS_CLI"] != "${{ steps.validation.outputs.binary-path }}" {
		t.Fatal("protected plan must use the published binary with narrowly scoped credentials")
	}
	var public []string
	for _, remote := range []string{"changed", "no-difference"} {
		t.Run(remote, func(t *testing.T) {
			dir := t.TempDir()
			bin := filepath.Join(dir, "gh-settings")
			script := `#!/bin/bash
if [[ "$1" != action || "$2" != plan || "$4" != status-only ]]; then
  exit 79
fi
printf '%s\n' "$REMOTE_OUTCOME" > "$3"
printf 'sensitive policy equals %s\n' "$REMOTE_OUTCOME" >&2
printf 'sensitive policy is %s\n' "$REMOTE_OUTCOME"
`
			if err := os.WriteFile(bin, []byte(script), 0700); err != nil {
				t.Fatal(err)
			}
			summary := filepath.Join(dir, "summary")
			output := filepath.Join(dir, "output")
			cmd := exec.Command("bash", "-e", "-o", "pipefail", "-c", step.Run)
			cmd.Env = append(os.Environ(),
				"SETTINGS_CLI="+bin, "GH_TOKEN=local-fixture", "REMOTE_OUTCOME="+remote,
				"GITHUB_STEP_SUMMARY="+summary, "GITHUB_OUTPUT="+output,
			)
			var stdout, stderr bytes.Buffer
			cmd.Stdout, cmd.Stderr = &stdout, &stderr
			if err := cmd.Run(); err != nil {
				t.Fatalf("both managed outcomes must succeed: %v", err)
			}
			public = append(public, stdout.String()+stderr.String())
			report, err := os.ReadFile(filepath.Join(dir, "plan.json"))
			if err != nil || string(report) != remote+"\n" {
				t.Fatalf("comparison did not execute privately: report=%q, err=%v", report, err)
			}
			info, err := os.Stat(filepath.Join(dir, "plan.json"))
			if err != nil || info.Mode().Perm() != 0600 {
				t.Fatalf("private comparison must remain owner-readable: %v, %v", info, err)
			}
			if _, err := os.Stat(summary); !os.IsNotExist(err) {
				t.Fatal("protected plan must not publish an Action Summary")
			}
			if _, err := os.Stat(output); !os.IsNotExist(err) {
				t.Fatal("protected plan must not publish a result output")
			}
		})
	}
	if len(public) != 2 || public[0] != "" || public[1] != "" {
		t.Fatalf("private policy equality leaked into public diagnostics: %q", public)
	}
}
