package githubactions

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"go.yaml.in/yaml/v4"
)

func TestPRScreeningOnly(t *testing.T) {
	w, _ := readWorkflow(t, "../../.github/workflows/pr.yml")
	var local []step
	for _, step := range w.Jobs["verify"].Steps {
		if strings.Contains(step.Uses, "artifact") || strings.Contains(step.Run, "upload") {
			t.Fatal("PR screening must not upload deployable artifacts")
		}
		if strings.HasPrefix(step.Uses, "./") {
			local = append(local, step)
		}
	}
	if len(local) != 1 || local[0].Uses != "./.github/actions/screening" || len(local[0].With) != 0 {
		t.Fatal("PR screening must build the CLI with its development version")
	}
	screening, _ := readReleaseAction(t, "screening")
	var actions []string
	for _, step := range screening.Runs.Steps {
		if step.If != "" || step.Continue {
			t.Fatal("every screening check must report its result")
		}
		actions = append(actions, step.Uses)
	}
	if !reflect.DeepEqual(actions, []string{"./.github/actions/lint", "./.github/actions/unit-test", "./.github/actions/build"}) {
		t.Fatalf("screening must lint, test and then build once: %v", actions)
	}
}

func TestCandidateWorkflowInspectsNativeBuildsByID(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(root(), ".github/workflows/candidate.yml"))
	if err != nil {
		t.Fatal(err)
	}
	var w struct {
		On   map[string]yaml.Node
		Jobs map[string]struct {
			Needs       []string
			Permissions map[string]string
			Steps       []step
			Strategy    struct {
				Matrix struct{ Include []map[string]string }
			}
		}
	}
	if err := yaml.Unmarshal(data, &w); err != nil {
		t.Fatal(err)
	}
	if len(w.On) != 1 || w.On["workflow_call"].Kind != yaml.MappingNode {
		t.Fatal("candidates must be built only through an explicit workflow call")
	}
	for osName, target := range map[string]string{"linux": "linux-amd64", "macos": "darwin-arm64"} {
		job := w.Jobs["screening-"+osName]
		if len(job.Steps) < 3 {
			t.Fatalf("missing native %s build", osName)
		}
		checkout, build, transfer := job.Steps[0], job.Steps[len(job.Steps)-2], job.Steps[len(job.Steps)-1]
		action := "./.github/actions/build"
		if osName == "linux" {
			action = "./.github/actions/screening"
		}
		if !reflect.DeepEqual(job.Permissions, map[string]string{"contents": "read"}) ||
			checkout.With["ref"] != "${{ inputs.source-sha }}" || checkout.With["persist-credentials"] != "false" ||
			build.Uses != action || build.With["version"] != "${{ inputs.version }}" || build.Continue || build.If != "" ||
			transfer.Uses != "./.github/actions/package-candidate" || transfer.With["binary-path"] != "${{ steps.build.outputs.binary-path }}" ||
			transfer.With["target"] != target || transfer.With["source-sha"] != "${{ inputs.source-sha }}" || transfer.With["version"] != "${{ inputs.version }}" {
			t.Fatalf("native %s screening must package the frozen versioned build", osName)
		}
	}
	inspect := w.Jobs["inspection"]
	lanes := []map[string]string{
		{"target": "linux-amd64", "runner": "ubuntu-slim", "artifact-id": "${{ needs.screening-linux.outputs.artifact-id }}"},
		{"target": "darwin-arm64", "runner": "macos-15", "artifact-id": "${{ needs.screening-macos.outputs.artifact-id }}"},
		{"target": "candidate-action", "runner": "ubuntu-24.04", "artifact-id": "${{ needs.screening-linux.outputs.artifact-id }}"},
	}
	if !reflect.DeepEqual(inspect.Needs, []string{"screening-linux", "screening-macos"}) || !reflect.DeepEqual(inspect.Strategy.Matrix.Include, lanes) ||
		len(inspect.Steps) != 2 || inspect.Steps[0].With["path"] != ".action" ||
		inspect.Steps[1].Uses != "./.action/.github/actions/release-acceptance" || inspect.Steps[1].With["artifact-id"] != "${{ matrix.artifact-id }}" {
		t.Fatal("each native and public Action inspection must consume its own producer's artifact ID")
	}
	acceptance, _ := readReleaseAction(t, "release-acceptance")
	var downloads []step
	for _, s := range acceptance.Runs.Steps {
		if strings.HasPrefix(s.Uses, "actions/download-artifact@") {
			downloads = append(downloads, s)
		}
	}
	if len(downloads) != 1 || downloads[0].With["artifact-ids"] != "${{ inputs.artifact-id }}" || downloads[0].With["digest-mismatch"] != "error" || downloads[0].With["name"] != "" {
		t.Fatal("inspection must retrieve an immutable candidate and fail on archive digest mismatch")
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
		if strings.Contains(s.Run, "go test -tags=acceptance ./tests/github-actions") {
			boundaryChecks++
			command := `(cd .action && go test -tags=acceptance ./tests/github-actions -run '^TestBuilt' -count=1 -binary "$BINARY_PATH")`
			if s.Continue || s.If != "" || !reflect.DeepEqual(s.Env, map[string]string{"BINARY_PATH": "${{ inputs.binary-path }}", "GOTOOLCHAIN": "local"}) || !strings.Contains("\n"+s.Run+"\n", "\n"+command+"\n") {
				t.Fatal("built composite checks must consume the supplied product and fail the candidate on regression")
			}
		}
	}
	if boundaryChecks != 1 {
		t.Fatal("candidate must run all supplied-product acceptance checks")
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
