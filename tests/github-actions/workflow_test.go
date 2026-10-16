package githubactions

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"go.yaml.in/yaml/v4"
)

type workflow struct {
	Name     string `yaml:"name"`
	Defaults struct {
		Run struct {
			Shell string `yaml:"shell"`
		} `yaml:"run"`
	} `yaml:"defaults"`
	On          map[string]yaml.Node `yaml:"on"`
	Env         map[string]string    `yaml:"env"`
	Permissions map[string]string    `yaml:"permissions"`
	Concurrency struct {
		Group  string `yaml:"group"`
		Cancel bool   `yaml:"cancel-in-progress"`
	} `yaml:"concurrency"`
	Jobs map[string]job `yaml:"jobs"`
}
type job struct {
	Timeout     int               `yaml:"timeout-minutes"`
	Needs       string            `yaml:"needs"`
	If          string            `yaml:"if"`
	Environment string            `yaml:"environment"`
	Runner      string            `yaml:"runs-on"`
	Env         map[string]string `yaml:"env"`
	Outputs     map[string]string `yaml:"outputs"`
	Permissions map[string]string `yaml:"permissions"`
	Continue    bool              `yaml:"continue-on-error"`
	Steps       []step            `yaml:"steps"`
}
type step struct {
	Name     string            `yaml:"name"`
	ID       string            `yaml:"id"`
	Uses     string            `yaml:"uses"`
	Run      string            `yaml:"run"`
	If       string            `yaml:"if"`
	Env      map[string]string `yaml:"env"`
	With     map[string]string `yaml:"with"`
	Continue bool              `yaml:"continue-on-error"`
}

func root() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Clean(filepath.Join(filepath.Dir(file), "../.."))
}
func readWorkflow(t *testing.T, name string) (workflow, string) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root(), "examples/github-actions", name))
	if err != nil {
		t.Fatal(err)
	}
	var w workflow
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err = decoder.Decode(&w); err != nil {
		t.Fatal(err)
	}
	return w, string(data)
}
func findStep(t *testing.T, j job, id string) step {
	t.Helper()
	for _, s := range j.Steps {
		if s.ID == id {
			return s
		}
	}
	t.Fatalf("missing step %s", id)
	return step{}
}
func TestWorkflowCredentialBoundaries(t *testing.T) {
	pr, prText := readWorkflow(t, "settings-pr.yml")
	main, mainText := readWorkflow(t, "settings-main.yml")
	if len(pr.On) != 1 || pr.On["pull_request"].Kind == 0 || len(main.On) != 1 || main.On["push"].Kind == 0 {
		t.Fatal("unexpected privileged trigger")
	}
	if strings.Contains(prText, "secrets.") || strings.Contains(prText, "environment:") || strings.Contains(prText, " apply") || strings.Contains(prText, "/apply.sh") {
		t.Fatal("privileged PR path")
	}
	for event, keys := range map[string]map[string]bool{"push": {"branches": true, "paths": true}, "pull_request": {"types": true, "paths": true}} {
		node := main.On[event]
		if event == "pull_request" {
			node = pr.On[event]
		}
		if node.Kind != yaml.MappingNode {
			t.Fatal("unsupported trigger shape")
		}
		for i := 0; i < len(node.Content); i += 2 {
			if !keys[node.Content[i].Value] {
				t.Fatal("unmodeled trigger filter")
			}
		}
	}
	if permissions := pr.Jobs["plan"].Permissions; permissions != nil && !reflect.DeepEqual(permissions, map[string]string{"contents": "read"}) {
		t.Fatal("PR read permission override")
	}
	var push struct {
		Branches []string `yaml:"branches"`
		Tags     []string `yaml:"tags"`
		Paths    []string `yaml:"paths"`
	}
	node := main.On["push"]
	if err := node.Decode(&push); err != nil {
		t.Fatal(err)
	}
	var pull struct {
		Types []string `yaml:"types"`
		Paths []string `yaml:"paths"`
	}
	prNode := pr.On["pull_request"]
	if err := prNode.Decode(&pull); err != nil {
		t.Fatal(err)
	}
	if len(pull.Types) != 0 {
		for _, required := range []string{"opened", "synchronize", "reopened"} {
			found := false
			for _, event := range pull.Types {
				if event == required {
					found = true
				}
			}
			if !found {
				t.Fatal("required PR event disabled")
			}
		}
	}
	if !reflect.DeepEqual(push.Paths, []string{".github/settings.yml", ".github/workflows/settings-main.yml", "examples/github-actions/**"}) || !reflect.DeepEqual(pull.Paths, []string{".github/settings.yml", ".github/workflows/settings-pr.yml", "examples/github-actions/**"}) {
		t.Fatal("required trigger paths changed")
	}
	if pr.Jobs["plan"].If != "" {
		t.Fatal("conditional PR job")
	}
	if main.Defaults.Run.Shell != "bash" || pr.Defaults.Run.Shell != "bash" {
		t.Fatal("runner shell contract changed")
	}
	if len(push.Branches) != 1 || push.Branches[0] != "main" || len(push.Tags) != 0 {
		t.Fatal("main trigger scope changed")
	}
	if len(pr.Permissions) != 1 || pr.Permissions["contents"] != "read" || len(main.Permissions) != 0 {
		t.Fatal("workflow permissions widened")
	}
	if !pr.Concurrency.Cancel || main.Concurrency.Cancel || main.Concurrency.Group != "settings-main-${{ github.repository }}" || pr.Concurrency.Group != "settings-pr-${{ github.repository }}-${{ github.event.pull_request.number }}" {
		t.Fatal("concurrency contract changed")
	}
	pin := regexp.MustCompile(`^[^@]+@[0-9a-f]{40}$`)
	for _, w := range []workflow{pr, main} {
		if len(w.Env) != 0 {
			t.Fatal("global token environment")
		}
		for _, j := range w.Jobs {
			for _, permission := range j.Permissions {
				if permission != "read" {
					t.Fatal("job permissions widened")
				}
			}
			if j.Runner != "ubuntu-24.04" || len(j.Env) != 0 || j.Continue || j.Timeout != 5 {
				t.Fatal("runner, job token or failure boundary changed")
			}
			for _, output := range j.Outputs {
				if strings.Contains(output, "token") || strings.Contains(output, "secret") {
					t.Fatal("credential job output")
				}
			}
			for _, s := range j.Steps {
				if s.Continue || (s.Uses != "" && !pin.MatchString(s.Uses)) {
					t.Fatal("unchecked Action or ignored failure")
				}
				if strings.Contains(s.Uses, "artifact") {
					t.Fatal("PR artifact promotion")
				}
				if strings.HasPrefix(s.Uses, "actions/checkout@") && (s.With["persist-credentials"] != "false" || s.With["repository"] != "${{ github.repository }}") {
					t.Fatal("checkout credential/target boundary")
				}
			}
		}
	}
	prepare, apply := main.Jobs["prepare"], main.Jobs["apply"]
	if pr.Jobs["plan"].Environment != "" || len(pr.Jobs["plan"].Outputs) != 0 || len(apply.Outputs) != 0 {
		t.Fatal("PR Environment or credential output contract changed")
	}
	if len(main.Jobs) != 2 || len(pr.Jobs) != 1 || len(pr.Jobs["plan"].Steps) != 2 || len(prepare.Steps) != 3 || len(apply.Steps) != 5 {
		t.Fatal("required job/step structure changed")
	}
	for _, j := range []job{pr.Jobs["plan"], prepare, apply} {
		for i, s := range j.Steps {
			if i == 0 {
				if len(s.Env) != 0 {
					t.Fatal("checkout credential environment")
				}
				expectedRef := "${{ github.sha }}"
				if j.Environment == "settings-apply" {
					expectedRef = "${{ needs.prepare.outputs.config-sha }}"
				}
				if !reflect.DeepEqual(s.With, map[string]string{"repository": "${{ github.repository }}", "ref": expectedRef, "persist-credentials": "false"}) {
					t.Fatal("checkout inputs/credential contract changed")
				}
			}
			if i == 0 && (s.Uses != "actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1" || s.With["ref"] == "") {
				t.Fatal("missing fixed checkout")
			}
			if s.If != "" && !(j.Environment == "settings-apply" && i == 4 && s.If == "always()") {
				t.Fatal("conditional required step")
			}
		}
	}
	if pr.Jobs["plan"].Steps[0].With["ref"] != "${{ github.sha }}" || prepare.Steps[0].With["ref"] != "${{ github.sha }}" {
		t.Fatal("moving PR/prepare checkout")
	}
	if len(pr.Jobs["plan"].Steps[1].Env) != 0 || len(findStep(t, apply, "app").Env) != 0 {
		t.Fatal("unexpected caller-step token environment")
	}
	prPlan := pr.Jobs["plan"].Steps[1]
	if prPlan.Uses != "daiksud/gh-settings@93d8217d146f5d430aaf90597119114218990e70" || prPlan.Run != "" || !reflect.DeepEqual(prPlan.With, map[string]string{"command": "plan", "token": "${{ github.token }}"}) {
		t.Fatal("PR public Action/read-token binding changed")
	}
	plan := findStep(t, prepare, "prepare")
	if plan.Uses != prPlan.Uses || plan.Run != "" || len(plan.Env) != 0 || !reflect.DeepEqual(plan.With, map[string]string{"command": "plan", "token": "${{ github.token }}"}) {
		t.Fatal("prepare public Action/read-token binding changed")
	}
	guardStep := findStep(t, prepare, "guard")
	if guardStep.Run == "" || guardStep.Uses != "" || len(guardStep.Env) != 0 || len(guardStep.With) != 0 {
		t.Fatal("prepare checkout guard credential boundary changed")
	}
	applyAction := findStep(t, apply, "apply")
	if applyAction.Uses != plan.Uses || applyAction.Run != "" || len(applyAction.Env) != 0 || !reflect.DeepEqual(applyAction.With, map[string]string{"command": "apply", "token": "${{ steps.app.outputs.token }}"}) {
		t.Fatal("apply public Action/token binding changed")
	}
	if !reflect.DeepEqual(prepare.Outputs, map[string]string{
		"config-sha": "${{ steps.prepare.outputs.config-sha }}", "cli-version": "${{ steps.prepare.outputs.cli-version }}", "cli-sha256": "${{ steps.prepare.outputs.cli-sha256 }}", "target": "${{ steps.prepare.outputs.target }}",
		"action-sha": "93d8217d146f5d430aaf90597119114218990e70",
	}) {
		t.Fatal("prepare output handoff changed")
	}
	if !reflect.DeepEqual(findStep(t, apply, "handoff").Env, map[string]string{
		"CONFIG_SHA": "${{ needs.prepare.outputs.config-sha }}", "EXPECTED_TARGET": "${{ needs.prepare.outputs.target }}", "EXPECTED_CLI_VERSION": "${{ needs.prepare.outputs.cli-version }}", "EXPECTED_CLI_SHA256": "${{ needs.prepare.outputs.cli-sha256 }}",
		"EXPECTED_ACTION_SHA": "${{ needs.prepare.outputs.action-sha }}", "REVIEWED_ACTION_SHA": "93d8217d146f5d430aaf90597119114218990e70",
	}) {
		t.Fatal("verify input handoff changed")
	}
	if !reflect.DeepEqual(apply.Steps[4].Env, map[string]string{"APPLY_OUTCOME": "${{ steps.apply.outcome }}"}) || apply.Steps[4].If != "always()" {
		t.Fatal("Summary outcome binding changed")
	}
	guard := "github.event_name == 'push' && github.ref == 'refs/heads/main'"
	if prepare.If != guard || apply.If != guard || apply.Needs != "prepare" || apply.Environment != "settings-apply" || prepare.Environment != "" {
		t.Fatal("prepare/approval/apply graph changed")
	}
	for _, s := range prepare.Steps {
		if strings.Contains(s.Uses+s.Run, "create-github-app-token") || strings.Contains(strings.Join(mapValues(s.Env), " "), "secrets.") {
			t.Fatal("prepare credential path")
		}
	}
	for _, j := range []job{prepare, apply} {
		if len(j.Permissions) != 1 || j.Permissions["contents"] != "read" {
			t.Fatal("missing required main read permission")
		}
	}
	indices := map[string]int{}
	for i, s := range apply.Steps {
		indices[s.ID] = i
	}
	if !(indices["handoff"] < indices["app"] && indices["app"] < indices["apply"]) {
		t.Fatal("credential step order changed")
	}
	if findStep(t, apply, "handoff").Run != "bash examples/github-actions/verify-action-handoff.sh" || strings.TrimSpace(apply.Steps[4].Run) != `printf '## Apply result\n\nCLI step outcome: %s. See the step log for errors.\n' "$APPLY_OUTCOME" >> "$GITHUB_STEP_SUMMARY"` {
		t.Fatal("privileged run command changed")
	}
	app := findStep(t, apply, "app")
	allowed := map[string]string{"client-id": "${{ vars.SETTINGS_APP_CLIENT_ID }}", "private-key": "${{ secrets.SETTINGS_APP_PRIVATE_KEY }}", "owner": "${{ github.repository_owner }}", "repositories": "${{ github.event.repository.name }}", "permission-administration": "write"}
	for key, value := range app.With {
		if key == "skip-token-revoke" && value == "false" {
			continue
		}
		if allowed[key] != value {
			t.Fatal("unexpected App input/permission")
		}
	}
	if app.Uses != "actions/create-github-app-token@64ab729a6294000f5aa4a4bc0b73d1f32bc45510" || app.With["repositories"] != "${{ github.event.repository.name }}" || app.With["owner"] != "${{ github.repository_owner }}" || app.With["permission-administration"] != "write" || app.With["private-key"] != "${{ secrets.SETTINGS_APP_PRIVATE_KEY }}" || (app.With["skip-token-revoke"] != "" && app.With["skip-token-revoke"] != "false") || app.With["client-id"] != "${{ vars.SETTINGS_APP_CLIENT_ID }}" {
		t.Fatal("App permission/key/revocation contract changed")
	}
	if strings.Count(mainText, "steps.app.outputs.token") != 1 || applyAction.With["token"] != "${{ steps.app.outputs.token }}" {
		t.Fatal("App token escaped apply step")
	}
	if apply.Steps[0].With["ref"] != "${{ needs.prepare.outputs.config-sha }}" {
		t.Fatal("moving configuration input")
	}
}
func mapValues(m map[string]string) []string {
	var v []string
	for _, s := range m {
		v = append(v, s)
	}
	return v
}
func TestMainPrepareCheckoutGuard(t *testing.T) {
	w, _ := readWorkflow(t, "settings-main.yml")
	guard := findStep(t, w.Jobs["prepare"], "guard")
	cmd := exec.Command("bash", "-e", "-o", "pipefail", "-c", `source tests/github-actions/fixture.sh
bash -e -o pipefail -c "$PREPARE_GUARD"
for context in 'pull_request refs/heads/main' 'push refs/heads/other' 'push refs/tags/main'; do
  read -r event ref <<< "$context"
  if GITHUB_EVENT_NAME="$event" GITHUB_REF="$ref" bash -e -o pipefail -c "$PREPARE_GUARD" > "$fixture/log" 2>&1; then exit 1; fi
done
if GITHUB_SHA=bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb bash -e -o pipefail -c "$PREPARE_GUARD" > "$fixture/log" 2>&1; then exit 1; fi
git remote set-url origin https://github.com/owner/other
if bash -e -o pipefail -c "$PREPARE_GUARD" > "$fixture/log" 2>&1; then exit 1; fi
[[ ! -e "$fixture/calls" && ! -e "$fixture/api" && ! -e "$GITHUB_OUTPUT" ]]`)
	cmd.Dir = root()
	cmd.Env = []string{"PATH=" + os.Getenv("PATH"), "HOME=" + os.Getenv("HOME"), "TMPDIR=" + os.Getenv("TMPDIR"), "PREPARE_GUARD=" + guard.Run}
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("prepare checkout guard failed: %v\n%s", err, output)
	}
}
func TestMainActionHandoff(t *testing.T) {
	w, _ := readWorkflow(t, "settings-main.yml")
	handoff := findStep(t, w.Jobs["apply"], "handoff")
	cmd := exec.Command("bash", filepath.Join(root(), "tests/github-actions/action-handoff.sh"))
	cmd.Dir = root()
	cmd.Env = []string{"PATH=" + os.Getenv("PATH"), "HOME=" + os.Getenv("HOME"), "TMPDIR=" + os.Getenv("TMPDIR"), "VERIFY_COMMAND=" + handoff.Run, "REVIEWED_ACTION_SHA=" + handoff.Env["REVIEWED_ACTION_SHA"]}
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("public Action caller handoff failed: %v\n%s", err, output)
	}
}
