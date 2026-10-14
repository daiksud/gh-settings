package githubactions

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"go.yaml.in/yaml/v4"
)

type workflow struct {
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
	if err = yaml.Unmarshal(data, &w); err != nil {
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
	var push struct {
		Branches []string `yaml:"branches"`
		Tags     []string `yaml:"tags"`
	}
	node := main.On["push"]
	if err := node.Decode(&push); err != nil {
		t.Fatal(err)
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
			if j.Runner != "ubuntu-24.04" || len(j.Env) != 0 || j.Continue {
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
	if !(indices["cli"] < indices["app"] && indices["app"] < indices["apply"]) {
		t.Fatal("credential step order changed")
	}
	if findStep(t, apply, "apply").Env["SETTINGS_CLI"] != "${{ steps.cli.outputs.binary-path }}" {
		t.Fatal("verified apply CLI binding changed")
	}
	app := findStep(t, apply, "app")
	if app.Uses != "actions/create-github-app-token@64ab729a6294000f5aa4a4bc0b73d1f32bc45510" || app.With["repositories"] != "${{ github.event.repository.name }}" || app.With["owner"] != "${{ github.repository_owner }}" || app.With["permission-administration"] != "write" || app.With["private-key"] != "${{ secrets.SETTINGS_APP_PRIVATE_KEY }}" || (app.With["skip-token-revoke"] != "" && app.With["skip-token-revoke"] != "false") || app.With["client-id"] != "${{ vars.SETTINGS_APP_CLIENT_ID }}" {
		t.Fatal("App permission/key/revocation contract changed")
	}
	if strings.Count(mainText, "steps.app.outputs.token") != 1 || findStep(t, apply, "apply").Env["GH_TOKEN"] != "${{ steps.app.outputs.token }}" {
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
func TestComposedOfflineWorkflow(t *testing.T) {
	w, _ := readWorkflow(t, "settings-main.yml")
	prepare, apply := w.Jobs["prepare"], w.Jobs["apply"]
	var summary step
	for _, s := range apply.Steps {
		if s.If == "always()" {
			summary = s
		}
	}
	if summary.Run == "" {
		t.Fatal("missing outcome Summary")
	}
	for _, scenario := range []string{"approved", "failed-prepare", "denied", "waiting", "target-mismatch", "missing-handoff", "unverified-binary", "missing-key", "apply-failure"} {
		t.Run(scenario, func(t *testing.T) {
			cmd := exec.Command("bash", filepath.Join(root(), "tests/github-actions/integration.sh"))
			cmd.Dir = root()
			// Real secrets are not passed to the fixture process.
			cmd.Env = []string{"PATH=" + os.Getenv("PATH"), "HOME=" + os.Getenv("HOME"), "TMPDIR=" + os.Getenv("TMPDIR"), "SCENARIO=" + scenario,
				"PREPARE_COMMAND=" + findStep(t, prepare, "prepare").Run, "VERIFY_COMMAND=" + findStep(t, apply, "cli").Run,
				"APPLY_COMMAND=" + findStep(t, apply, "apply").Run, "SUMMARY_COMMAND=" + summary.Run,
				"TOKEN_PERMISSION=" + findStep(t, apply, "app").With["permission-administration"]}
			if output, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("%v\n%s", err, output)
			}
		})
	}
}
