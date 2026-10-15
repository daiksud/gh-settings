package githubactions

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"go.yaml.in/yaml/v4"
)

type actionField struct {
	Description string  `yaml:"description"`
	Required    bool    `yaml:"required"`
	Default     *string `yaml:"default"`
	Value       string  `yaml:"value"`
}
type compositeStep struct {
	Name  string            `yaml:"name"`
	ID    string            `yaml:"id"`
	Shell string            `yaml:"shell"`
	If    string            `yaml:"if"`
	Env   map[string]string `yaml:"env"`
	Run   string            `yaml:"run"`
}
type composite struct {
	Name        string                 `yaml:"name"`
	Author      string                 `yaml:"author"`
	Description string                 `yaml:"description"`
	Inputs      map[string]actionField `yaml:"inputs"`
	Outputs     map[string]actionField `yaml:"outputs"`
	Runs        struct {
		Using string          `yaml:"using"`
		Steps []compositeStep `yaml:"steps"`
	} `yaml:"runs"`
}

func TestRootCompositeValidate(t *testing.T) {
	result := runComposite(t, "validate", "", nil)
	if result.status != 0 {
		t.Fatalf("status=%d\n%s", result.status, result.log)
	}
	want := map[string]string{"config-sha": result.sha, "target": "owner/config", "cli-version": "v0.5.0", "cli-sha256": result.digest}
	if !reflect.DeepEqual(result.outputs, want) {
		t.Fatalf("outputs=%v, want %v", result.outputs, want)
	}
	if result.calls != "validate\n" {
		t.Fatalf("settings calls=%q", result.calls)
	}
	if result.summary != "" {
		t.Fatalf("unexpected Summary: %s", result.summary)
	}
}

func TestRootCompositePlan(t *testing.T) {
	result := runComposite(t, "plan", "fixture-token", nil)
	if result.status != 0 || len(result.outputs) != 4 {
		t.Fatalf("status=%d outputs=%v\n%s", result.status, result.outputs, result.log)
	}
	if result.calls != "validate\nplan\n" {
		t.Fatalf("settings calls=%q", result.calls)
	}
	for _, text := range []string{"Settings review", "before → desired", result.sha, "owner/config", "workflow-owner/workflow", "action-owner/action", "approved-action-ref", strings.Repeat("b", 40), "original-actor", "rerun-actor"} {
		if !strings.Contains(result.summary, text) {
			t.Errorf("Summary missing %q: %s", text, result.summary)
		}
	}
	if strings.Contains(result.summary+result.log, "fixture-token") {
		t.Fatal("token escaped execution environment")
	}
}

func TestRootCompositeApply(t *testing.T) {
	result := runComposite(t, "apply", "fixture-token", nil)
	if result.status != 0 || len(result.outputs) != 4 || result.calls != "apply\n" || result.summary != "" {
		t.Fatalf("status=%d outputs=%v calls=%q Summary=%s\n%s", result.status, result.outputs, result.calls, result.summary, result.log)
	}
}

type compositeResult struct {
	status                           int
	outputs                          map[string]string
	log, calls, summary, sha, digest string
}

func runComposite(t *testing.T, command, token string, extra map[string]string) compositeResult {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root(), "action.yml"))
	if err != nil {
		t.Fatal(err)
	}
	var action composite
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(&action); err != nil {
		t.Fatal(err)
	}
	if action.Name == "" || action.Description == "" || action.Runs.Using != "composite" {
		t.Fatal("missing composite metadata")
	}
	if len(action.Inputs) != 2 || !action.Inputs["command"].Required || action.Inputs["command"].Default != nil || action.Inputs["token"].Default != nil {
		t.Fatal("input contract changed")
	}
	if len(action.Outputs) != 4 {
		t.Fatal("public output contract changed")
	}
	fixture := t.TempDir()
	caller, packagePath, bin := filepath.Join(fixture, "caller"), filepath.Join(fixture, "action package"), filepath.Join(fixture, "bin")
	for _, dir := range []string{caller, packagePath, bin, filepath.Join(fixture, "runner temp")} {
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
	}
	write := func(path, value string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(value), 0700); err != nil {
			t.Fatal(err)
		}
	}
	asset := "#!/usr/bin/env bash\nset -euo pipefail\ncase \"$1\" in\n--version) [[ -z ${GH_TOKEN:-} && -z ${GITHUB_TOKEN:-} ]]; echo 'gh-settings v0.5.0' ;;\nvalidate) [[ -z ${GH_TOKEN:-} && -z ${GITHUB_TOKEN:-} ]]; echo validate >> \"$MOCK_CALLS\" ;;\nplan|apply) [[ ${GH_TOKEN:-} == fixture-token && -z ${GITHUB_TOKEN:-} ]]; echo \"$1\" >> \"$MOCK_CALLS\"; echo 'before → desired' ;;\n*) exit 99 ;;\nesac\n"
	digest := fmt.Sprintf("%x", sha256.Sum256([]byte(asset)))
	write(filepath.Join(fixture, "asset"), asset)
	write(filepath.Join(bin, "curl"), "#!/usr/bin/env bash\nset -euo pipefail\nwhile [[ $# -gt 0 ]]; do\nif [[ $1 == --output ]]; then cp \"$MOCK_ASSET\" \"$2\"; exit; fi\nshift\ndone\nexit 1\n")
	write(filepath.Join(bin, "sha256sum"), "#!/usr/bin/env bash\nexec shasum -a 256 \"$@\"\n")
	files, err := os.ReadDir(filepath.Join(root(), "action"))
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range files {
		content, err := os.ReadFile(filepath.Join(root(), "action", file.Name()))
		if err != nil {
			t.Fatal(err)
		}
		if file.Name() == "install-cli.sh" {
			content = []byte(strings.ReplaceAll(string(content), "b4a63eb8912657deabc5ea8909a7e6198d375c94cb0a9cd83f7d2179f7bbf5b4", digest))
		}
		if err := os.MkdirAll(filepath.Join(packagePath, "action"), 0700); err != nil {
			t.Fatal(err)
		}
		write(filepath.Join(packagePath, "action", file.Name()), string(content))
	}
	base := map[string]string{"PATH": bin + string(os.PathListSeparator) + os.Getenv("PATH"), "HOME": fixture, "GIT_CONFIG_NOSYSTEM": "1", "GIT_CONFIG_GLOBAL": filepath.Join(fixture, "gitconfig"), "GITHUB_WORKSPACE": caller, "RUNNER_TEMP": filepath.Join(fixture, "runner temp"), "RUNNER_OS": "Linux", "RUNNER_ARCH": "X64", "GITHUB_STEP_SUMMARY": filepath.Join(fixture, "summary"), "MOCK_ASSET": filepath.Join(fixture, "asset"), "MOCK_CALLS": filepath.Join(fixture, "calls")}
	environment := func(values map[string]string) []string {
		var env []string
		for key, value := range values {
			env = append(env, key+"="+value)
		}
		return env
	}
	git := func(args ...string) string {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = caller
		cmd.Env = environment(base)
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, output)
		}
		return strings.TrimSpace(string(output))
	}
	git("init", "--quiet")
	git("-c", "user.name=Fixture", "-c", "user.email=fixture@example.test", "-c", "commit.gpgsign=false", "commit", "--allow-empty", "--quiet", "-m", "fixture")
	git("remote", "add", "origin", "https://github.com/owner/config.git")
	sha := git("rev-parse", "HEAD")
	values := map[string]string{"inputs.command": command, "inputs.token": token, "inputs.token != ''": fmt.Sprint(token != ""), "github.action_path": packagePath,
		"github.actor": "original-actor", "github.triggering_actor": "rerun-actor", "github.event_name": "push", "github.run_id": "123", "github.run_attempt": "2", "github.workflow_ref": "workflow-owner/workflow/.github/workflows/plan.yml@refs/heads/main", "github.workflow_sha": strings.Repeat("b", 40), "github.action_repository": "action-owner/action", "github.action_ref": "approved-action-ref"}
	for key, value := range extra {
		base[key] = value
	}
	resolve := func(value string) string {
		t.Helper()
		if strings.HasPrefix(value, "${{ ") && strings.HasSuffix(value, " }}") {
			key := strings.TrimSuffix(strings.TrimPrefix(value, "${{ "), " }}")
			resolved, ok := values[key]
			if !ok {
				t.Fatalf("unbound fixture context %s", key)
			}
			return resolved
		}
		return value
	}
	result := compositeResult{outputs: map[string]string{}, sha: sha, digest: digest}
	for i, step := range action.Runs.Steps {
		if step.Shell != "bash" || step.Run == "" || strings.Contains(step.Run, "${{") {
			t.Fatal("composite command must use explicit Bash and env-bound contexts")
		}
		switch step.If {
		case "":
		case "inputs.command == 'validate'":
			if command != "validate" {
				continue
			}
		case "inputs.command == 'plan'":
			if command != "plan" {
				continue
			}
		case "inputs.command != 'validate'":
			if command == "validate" {
				continue
			}
		default:
			t.Fatalf("unmodeled selected-path condition %q", step.If)
		}
		env := map[string]string{}
		for key, value := range base {
			env[key] = value
		}
		for key, value := range step.Env {
			if strings.Contains(value, "inputs.token") && (key != "GH_TOKEN" || (step.If != "inputs.command == 'plan'" && step.If != "inputs.command != 'validate'")) {
				t.Fatal("token input outside authenticated execution boundary")
			}
			env[key] = resolve(value)
		}
		outputPath := filepath.Join(fixture, fmt.Sprintf("step-%d", i))
		env["GITHUB_OUTPUT"] = outputPath
		cmd := exec.Command("bash", "--noprofile", "--norc", "-e", "-o", "pipefail", "-c", step.Run)
		cmd.Dir = caller
		cmd.Env = environment(env)
		output, err := cmd.CombinedOutput()
		result.log += string(output)
		if err != nil {
			if exit, ok := err.(*exec.ExitError); ok {
				result.status = exit.ExitCode()
			} else {
				t.Fatal(err)
			}
			break
		}
		fields, _ := os.ReadFile(outputPath)
		for _, line := range strings.Split(strings.TrimSpace(string(fields)), "\n") {
			key, value, ok := strings.Cut(line, "=")
			if ok {
				values["steps."+step.ID+".outputs."+key] = value
			}
		}
	}
	for name, field := range action.Outputs {
		if field.Description == "" {
			t.Fatal("missing output description")
		}
		key := strings.TrimSuffix(strings.TrimPrefix(field.Value, "${{ "), " }}")
		if value := values[key]; value != "" {
			result.outputs[name] = value
		}
	}
	calls, _ := os.ReadFile(base["MOCK_CALLS"])
	result.calls = string(calls)
	summary, _ := os.ReadFile(base["GITHUB_STEP_SUMMARY"])
	result.summary = string(summary)
	return result
}
