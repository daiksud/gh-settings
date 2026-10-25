package githubactions

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"go.yaml.in/yaml/v4"
)

func TestCandidateCompositeCallers(t *testing.T) {
	for _, name := range []string{"ci-pr.yml", "ci-push.yml"} {
		t.Run(name, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join(root(), ".github/workflows", name))
			if err != nil {
				t.Fatal(err)
			}
			var w struct{ Jobs map[string]yaml.Node }
			if err := yaml.Unmarshal(data, &w); err != nil {
				t.Fatal(err)
			}
			if _, exists := w.Jobs["composite-smoke"]; exists {
				t.Fatal("public Action acceptance must be a matrix leg, not a separate CI job")
			}
			var acceptance struct {
				Needs       []string          `yaml:"needs"`
				Runner      string            `yaml:"runs-on"`
				Permissions map[string]string `yaml:"permissions"`
				Steps       []step            `yaml:"steps"`
			}
			acceptanceNode := w.Jobs["acceptance-stage"]
			if err := acceptanceNode.Decode(&acceptance); err != nil {
				t.Fatal(err)
			}
			wantNeeds := []string{"resolve", "commit-stage"}
			if name == "ci-pr.yml" {
				wantNeeds = append(wantNeeds, "ready")
			}
			if !reflect.DeepEqual(acceptance.Needs, wantNeeds) ||
				acceptance.Runner != "${{ matrix.runner }}" ||
				!reflect.DeepEqual(acceptance.Permissions, map[string]string{"contents": "read"}) ||
				len(acceptance.Steps) != 3 {
				t.Fatalf("acceptance stage must depend on the same-run producer and ready state: %+v", acceptance)
			}
			checkout, download, candidate := acceptance.Steps[0], acceptance.Steps[1], acceptance.Steps[2]
			if checkout.Uses != "actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1" ||
				!reflect.DeepEqual(checkout.With, map[string]string{
					"ref": "${{ github.sha }}", "path": ".action", "persist-credentials": "false",
				}) {
				t.Fatalf("acceptance must checkout the exact producer source without credentials: %+v", checkout)
			}
			if !strings.HasPrefix(download.Uses, "actions/download-artifact@") ||
				!reflect.DeepEqual(download.With, map[string]string{
					"name": "ci-${{ matrix.target }}-${{ github.sha }}",
					"path": "${{ runner.temp }}/product", "digest-mismatch": "error",
				}) {
				t.Fatalf("acceptance must download the same run's manifested artifact: %+v", download)
			}
			if candidate.Uses != "./.action/.github/actions/ci-acceptance" ||
				!reflect.DeepEqual(candidate.With, map[string]string{
					"target": "${{ matrix.target }}", "kind": "${{ matrix.kind }}",
					"source-sha": "${{ github.sha }}", "artifact-directory": "${{ runner.temp }}/product",
				}) {
				t.Fatalf("acceptance must verify candidate identity and execute each matrix kind: %+v", candidate)
			}
			for _, step := range acceptance.Steps {
				if step.Run != "" || step.Continue || step.If != "" || len(step.Env) != 0 {
					t.Fatalf("untrusted candidate step must not have a privileged or skipped path: %+v", step)
				}
			}
			if name == "ci-pr.yml" {
				return // PR code-result is exercised in pr_code_gate_test.go.
			}
			var pass struct {
				Needs []string `yaml:"needs"`
				If    string   `yaml:"if"`
				Steps []step   `yaml:"steps"`
			}
			passNode := w.Jobs["pass"]
			if err := passNode.Decode(&pass); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(pass.Needs, []string{"resolve", "commit-stage", "acceptance-stage"}) ||
				pass.If != "always() && !cancelled()" || len(pass.Steps) != 1 ||
				pass.Steps[0].Env["RESULTS"] != "${{ toJSON(needs) }}" ||
				!strings.Contains(pass.Steps[0].Run, "length == 3 and all(.[]; .result == \"success\")") {
				t.Fatalf("push pass must require all stages without treating cancellation as success: %+v", pass)
			}
		})
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
		{id: "private", command: "plan", token: "plan-fixture-read", assertion: "assert private"},
		{id: "private-read-failure", command: "plan", token: "plan-fixture-read", assertion: "assert private-read-failure", failure: true},
		{id: "private-invalid-config", command: "plan", token: "plan-fixture-read", assertion: "assert private-invalid-config", failure: true},
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
			if strings.HasPrefix(test.id, "private") {
				wantInputs["plan-reporting"] = "status-only"
			}
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
			if strings.HasPrefix(test.id, "private") {
				wantEnv["PRIVATE_PLAN_OUTCOME"] = "${{ steps." + test.id + ".outputs.plan-outcome }}"
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
