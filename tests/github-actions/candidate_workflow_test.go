package githubactions

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"go.yaml.in/yaml/v4"
)

func TestPRCandidateCompositeWiring(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(root(), ".github/workflows/ci-pr.yml"))
	if err != nil {
		t.Fatal(err)
	}
	var workflow struct{ Jobs map[string]yaml.Node }
	if err := yaml.Unmarshal(data, &workflow); err != nil {
		t.Fatal(err)
	}
	var smoke struct {
		Needs       []string          `yaml:"needs"`
		Runner      string            `yaml:"runs-on"`
		Timeout     int               `yaml:"timeout-minutes"`
		Permissions map[string]string `yaml:"permissions"`
		If          string            `yaml:"if"`
		Continue    bool              `yaml:"continue-on-error"`
		Steps       []step            `yaml:"steps"`
	}
	node := workflow.Jobs["composite-smoke"]
	if err := node.Decode(&smoke); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(smoke.Needs, []string{"commit-stage", "ready"}) || smoke.Runner != "ubuntu-24.04" || smoke.Timeout != 5 || !reflect.DeepEqual(smoke.Permissions, map[string]string{"contents": "read"}) || smoke.If != "" || smoke.Continue || len(smoke.Steps) != 4 {
		t.Fatalf("PR candidate caller boundaries changed: %+v", smoke)
	}
	checkout, tools, download, candidate := smoke.Steps[0], smoke.Steps[1], smoke.Steps[2], smoke.Steps[3]
	if checkout.Uses != "actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1" || !reflect.DeepEqual(checkout.With, map[string]string{
		"ref": "${{ needs.commit-stage.outputs.source-sha }}", "path": ".action", "persist-credentials": "false",
	}) {
		t.Fatalf("candidate must check out the exact producer source separately: %+v", checkout)
	}
	if !strings.HasPrefix(tools.Uses, "jdx/mise-action@") || tools.With["working_directory"] != ".action" || tools.With["install_args"] != "go" || tools.With["cache"] != "false" {
		t.Fatalf("caller must install Go only for the local API fixture: %+v", tools)
	}
	if !strings.HasPrefix(download.Uses, "actions/download-artifact@") || !reflect.DeepEqual(download.With, map[string]string{
		"name": "gh-settings", "path": "${{ runner.temp }}/product", "digest-mismatch": "error",
	}) {
		t.Fatalf("candidate must consume the existing producer artifact: %+v", download)
	}
	if candidate.Uses != "./.action/.github/actions/candidate-smoke" || !reflect.DeepEqual(candidate.With, map[string]string{
		"source-sha": "${{ needs.commit-stage.outputs.source-sha }}", "cli-version": "${{ needs.commit-stage.outputs.cli-version }}",
		"binary-sha256": "${{ needs.commit-stage.outputs.binary-sha256 }}", "binary-path": "${{ runner.temp }}/product/gh-settings",
	}) {
		t.Fatalf("candidate source/version/artifact handoff changed: %+v", candidate)
	}
	for _, s := range smoke.Steps {
		if s.If != "" || s.Continue || s.Run != "" || len(s.Env) != 0 {
			t.Fatalf("candidate caller bypasses its success gate or adds execution/token bindings: %+v", s)
		}
	}
	var pass struct {
		Needs []string `yaml:"needs"`
		If    string   `yaml:"if"`
		Steps []step   `yaml:"steps"`
	}
	node = workflow.Jobs["pass"]
	if err := node.Decode(&pass); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(pass.Needs, []string{"commitlint", "acceptance-stage", "composite-smoke"}) || pass.If != "always()" || len(pass.Steps) != 1 || pass.Steps[0].If != "cancelled() || needs.commitlint.result != 'success' || needs['acceptance-stage'].result != 'success' || needs['composite-smoke'].result != 'success'" || pass.Steps[0].Run != "exit 1" || pass.Steps[0].Continue {
		t.Fatalf("PR pass must reject failed, skipped or cancelled candidate validation: %+v", pass)
	}
}

func TestCandidateCompositeScenarios(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(root(), ".github/actions/candidate-smoke/action.yml"))
	if err != nil {
		t.Fatal(err)
	}
	var action struct {
		Inputs map[string]actionField `yaml:"inputs"`
		Runs   struct {
			Using string `yaml:"using"`
			Steps []step `yaml:"steps"`
		} `yaml:"runs"`
	}
	if err := yaml.Unmarshal(data, &action); err != nil {
		t.Fatal(err)
	}
	if action.Runs.Using != "composite" || len(action.Inputs) != 4 {
		t.Fatal("candidate scenarios must use a local composite with explicit producer identity")
	}
	for _, name := range []string{"source-sha", "cli-version", "binary-sha256", "binary-path"} {
		input := action.Inputs[name]
		if !input.Required || input.Description == "" || input.Default != nil {
			t.Fatalf("candidate input %s must be explicitly supplied: %+v", name, input)
		}
	}
	steps := action.Runs.Steps
	for _, s := range steps {
		if (s.Run != "" && s.Shell != "bash") || (s.Uses != "" && s.Uses != "./.action") || (s.Continue && s.Uses != "./.action") || s.Env["GH_TOKEN"] != "" || s.Env["GITHUB_TOKEN"] != "" {
			t.Fatalf("candidate step changes shell, public Action expansion or ambient credentials: %+v", s)
		}
	}
	if len(steps) == 0 || !strings.Contains(steps[0].Run, "bash .action/tests/github-actions/prepare-candidate-download.sh") || !reflect.DeepEqual(steps[0].Env, map[string]string{
		"BINARY_PATH": "${{ inputs.binary-path }}", "EXPECTED_SHA256": "${{ inputs.binary-sha256 }}",
		"SOURCE_SHA": "${{ inputs.source-sha }}", "CLI_VERSION": "${{ inputs.cli-version }}",
	}) {
		t.Fatal("candidate verification must consume all four producer inputs before scenarios")
	}
	covered := make(map[string]bool)
	for _, test := range []struct {
		id, command, token, assertion string
		failure                       bool
	}{
		{id: "validate", command: "validate"},
		{id: "plan", command: "plan", token: "plan-fixture-read", assertion: "assert"},
		{id: "read-failure", command: "plan", token: "plan-fixture-read", assertion: "assert read-failure", failure: true},
		{id: "invalid-config", command: "plan", token: "plan-fixture-read", assertion: "assert invalid-config", failure: true},
		{id: "apply", command: "apply", token: "apply-fixture-write", assertion: "assert apply"},
		{id: "write-failure", command: "apply", token: "apply-fixture-write", assertion: "assert apply-write-failure", failure: true},
		{id: "apply-invalid", command: "apply", token: "apply-fixture-write", assertion: "assert apply-invalid", failure: true},
		{id: "apply-empty", command: "apply", token: "apply-fixture-write", assertion: "assert apply-empty"},
	} {
		covered[test.id] = true
		t.Run(test.id, func(t *testing.T) {
			index := -1
			for i, s := range steps {
				if s.ID == test.id {
					if index != -1 {
						t.Fatalf("duplicate scenario ID %s", test.id)
					}
					index = i
				}
			}
			if index < 0 || index+1 >= len(steps) {
				t.Fatalf("missing scenario/assertion for %s", test.id)
			}
			scenario, assertion := steps[index], steps[index+1]
			wantInputs := map[string]string{"command": test.command}
			if test.token != "" {
				wantInputs["token"] = test.token
			}
			if scenario.Uses != "./.action" || !reflect.DeepEqual(scenario.With, wantInputs) || scenario.Continue != test.failure || scenario.If != "" || assertion.If != "" || assertion.Continue || assertion.Run == "" {
				t.Fatalf("actual Action scenario must have a mandatory result assertion: %+v then %+v", scenario, assertion)
			}
			wantEnv := map[string]string{
				"ACTION_SHA": "${{ inputs.source-sha }}", "EXPECTED_CLI_VERSION": "${{ inputs.cli-version }}", "EXPECTED_CLI_SHA256": "${{ inputs.binary-sha256 }}",
			}
			for variable, output := range map[string]string{"CONFIG_SHA": "config-sha", "TARGET": "target", "CLI_VERSION": "cli-version", "CLI_SHA256": "cli-sha256"} {
				wantEnv[variable] = "${{ steps." + test.id + ".outputs." + output + " }}"
			}
			if test.id != "validate" {
				wantEnv["PLAN_OUTCOME"] = "${{ steps." + test.id + ".outcome }}"
				if assertion.Run != "bash .action/tests/github-actions/candidate-plan.sh "+test.assertion {
					t.Fatalf("scenario %s is missing its result/Summary assertion: %+v", test.id, assertion)
				}
			}
			if !reflect.DeepEqual(assertion.Env, wantEnv) {
				t.Fatalf("scenario %s must verify all four outputs and producer identity: %+v", test.id, assertion)
			}
		})
	}
	for _, s := range steps {
		if s.Uses != "" && !covered[s.ID] {
			t.Fatalf("Action invocation %q is missing a scenario/result assertion contract", s.ID)
		}
	}
	cleanup := steps[len(steps)-1]
	if cleanup.If != "always()" || cleanup.Continue || cleanup.Run != "bash .action/tests/github-actions/candidate-plan.sh stop" {
		t.Fatalf("owned API fixture must be stopped even after an earlier failure: %+v", cleanup)
	}
}
