package githubactions

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"go.yaml.in/yaml/v4"
)

// PR validation composes checks directly without transferring artifacts.
func TestPRScreeningOnly(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(root(), ".github/workflows/pr.yml"))
	if err != nil {
		t.Fatal(err)
	}
	var workflow struct {
		Jobs map[string]struct {
			Steps []step `yaml:"steps"`
		} `yaml:"jobs"`
	}
	if err := yaml.Unmarshal(data, &workflow); err != nil {
		t.Fatal(err)
	}
	var actions []string
	for _, s := range workflow.Jobs["screening"].Steps {
		if strings.HasPrefix(s.Uses, "actions/checkout@") {
			continue
		}
		actions = append(actions, s.Uses)
	}
	want := []string{
		"jdx/mise-action@b44d12e7d0d8ee7461bb893e1b1993500433185b",
		"actions/setup-go@b7ad1dad31e06c5925ef5d2fc7ad053ef454303e",
		"./.github/actions/lint",
		"./.github/actions/unit-test",
		"./.github/actions/build",
	}
	if !reflect.DeepEqual(actions, want) {
		t.Fatalf("screening must run independent source checks and build without artifact upload: %v", actions)
	}
	if _, err := os.Stat(filepath.Join(root(), ".github/actions/commit-stage")); !os.IsNotExist(err) {
		t.Fatal("the superseded commit-stage Action must be removed")
	}
}

// Main produces release-ready artifacts after native and public-Action
// inspection. PR development candidates do not establish release readiness.
func TestMainReleaseReadyCandidateWorkflow(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(root(), ".github/workflows/ci-push.yml"))
	if err != nil {
		t.Fatal(err)
	}
	source := string(data)
	for _, required := range []string{
		"name: ci-push",
		"branches: [main]",
		"fetch-depth: 0",
		"uses: ./.github/actions/release-metadata",
		"uses: ./.github/actions/lint",
		"uses: ./.github/actions/unit-test",
		"uses: ./.github/actions/build",
		"release-artifact.sh write",
		"uses: actions/upload-artifact@",
		"uses: ./.action/.github/actions/release-acceptance",
		"needs.resolve.outputs.version",
		"needs.resolve.outputs.build-matrix",
		"needs.resolve.outputs.acceptance-matrix",
		"matrix.target",
		"source-sha:",
		"always() && !cancelled()",
		"length == 3 and all(.[]; .result == \"success\")",
	} {
		if !strings.Contains(source, required) {
			t.Fatalf("main release-ready validation lacks %q", required)
		}
	}
	var w struct {
		Jobs map[string]yaml.Node
	}
	if err := yaml.Unmarshal(data, &w); err != nil {
		t.Fatal(err)
	}
	if len(w.Jobs) != 4 {
		t.Fatalf("want exact resolve/build/inspection/gate jobs, got %v", w.Jobs)
	}
	for _, name := range []string{"resolve", "commit-stage", "acceptance-stage", "ci-push"} {
		if _, exists := w.Jobs[name]; !exists {
			t.Fatalf("missing %s", name)
		}
	}
}

func TestCandidateCompositeScenarios(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(root(), ".github/actions/e2e-test/action.yml"))
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
	boundaryChecks := 0
	for _, s := range steps {
		if strings.Contains(s.Run, "TestBuilt(CompositeExport|CandidateExportFixture|CompositeEnvironmentPrune|CompositeWorkflowPermissions|CompositeActionsExecution|CompositeCollaborators|CompositeRulesetCoverage|CompositeEnvironmentApps|CompositeAIScanRepositories|CompositeCollaboratorPrune|CompositeClassicLock|CompositeClassicPrune|CompositeOIDCSubject|CompositeClassicAdmin|CompositeAutolinks|CompositeClassicDeployments|CompositeDeployKeys|CompositeClassicStatusChecks|CompositeClassicReviews|CompositeClassicForkSync|CompositeClassicForcePush|CompositeRunnerLabels|CompositeWebhooks|CompositeWebhookCreation|CompositeWebhookPrune|CompositeWebhookPruneMixed)") {
			boundaryChecks++
			if s.Continue || s.If != "" || !reflect.DeepEqual(s.Env, map[string]string{"BINARY_PATH": "${{ inputs.binary-path }}", "GOTOOLCHAIN": "local"}) || !strings.Contains(s.Run, `-binary "$BINARY_PATH"`) {
				t.Fatal("built composite checks must consume the supplied product and fail the candidate on regression")
			}
		}
	}
	if boundaryChecks != 1 {
		t.Fatal("candidate must run the supplied-product export boundary checks")
	}
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
		{id: "export-first", command: "export", token: "export-fixture-read", assertion: "assert-export first"},
		{id: "export-repeat", command: "export", token: "export-fixture-read", assertion: "assert-export repeat"},
		{id: "export-invalid", command: "export", token: "export-fixture-read", assertion: "assert-export invalid", failure: true},
		{id: "export-symlink", command: "export", token: "export-fixture-read", assertion: "assert-export symlink", failure: true},
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
			if strings.HasPrefix(test.id, "export-") {
				wantEnv["CONFIG_PATH"] = "${{ steps." + test.id + ".outputs.config-path }}"
				if !reflect.DeepEqual(scenario.Env, map[string]string{"GH_DEBUG": "api"}) {
					t.Fatal("export must exercise inherited HTTP debug removal")
				}
			}
			if strings.HasPrefix(test.id, "private") || strings.HasPrefix(test.id, "export-") {
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
