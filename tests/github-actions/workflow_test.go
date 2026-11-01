package githubactions

import (
	"bytes"
	"os"
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
	Shell    string            `yaml:"shell"`
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

const (
	exampleAction         = "daiksud/gh-settings@cccd06a151c2dce3b9eb71520d3c01b4c73ac92f"
	exampleConsumerAction = "daiksud/gh-settings@6dbdbb3ab11cbc0ef257fef089d6915d31d003d6"
)

func TestWorkflowCredentialBoundaries(t *testing.T) {
	pr, prText := readWorkflow(t, "settings-pr.yml")
	main, mainText := readWorkflow(t, "settings-main.yml")
	if len(pr.On) != 1 || pr.On["pull_request_target"].Kind == 0 || len(main.On) != 1 || main.On["push"].Kind == 0 {
		t.Fatal("unexpected privileged trigger")
	}
	if strings.Contains(prText, "command: apply") || strings.Contains(prText, "actions/checkout@") || strings.Contains(prText, "uses: ./") {
		t.Fatal("PR executes candidate code or applies settings")
	}
	for event, keys := range map[string]map[string]bool{"push": {"branches": true, "paths": true}, "pull_request_target": {"types": true, "paths": true, "branches": true}} {
		node := main.On[event]
		if event == "pull_request_target" {
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
		Types    []string `yaml:"types"`
		Paths    []string `yaml:"paths"`
		Branches []string `yaml:"branches"`
	}
	prNode := pr.On["pull_request_target"]
	if err := prNode.Decode(&pull); err != nil {
		t.Fatal(err)
	}
	if len(pull.Types) != 0 {
		for _, required := range []string{"opened", "synchronize", "reopened"} {
			found := false
			for _, event := range pull.Types {
				found = found || event == required
			}
			if !found {
				t.Fatal("required PR event disabled")
			}
		}
	}
	if !reflect.DeepEqual(push.Paths, []string{".github/settings.yml", ".github/workflows/settings-main.yml"}) || !reflect.DeepEqual(pull.Paths, []string{".github/settings.yml", ".github/workflows/settings-pr.yml", "examples/github-actions/**"}) {
		t.Fatal("required trigger paths changed")
	}
	if len(push.Branches) != 1 || push.Branches[0] != "main" || len(push.Tags) != 0 {
		t.Fatal("main trigger scope changed")
	}
	if !reflect.DeepEqual(pull.Branches, []string{"main"}) {
		t.Fatal("PR trusted-base scope changed")
	}
	if !pr.Concurrency.Cancel || main.Concurrency.Cancel || main.Concurrency.Group != "settings-main-${{ github.repository }}" || pr.Concurrency.Group != "settings-pr-${{ github.repository }}-${{ github.event.pull_request.number }}" {
		t.Fatal("concurrency contract changed")
	}
	pin := regexp.MustCompile(`^[^@]+@[0-9a-f]{40}$`)
	for index, w := range []workflow{pr, main} {
		if len(w.Env) != 0 || (index == 0 && len(w.Permissions) != 0) || (index == 1 && !reflect.DeepEqual(w.Permissions, map[string]string{"contents": "read"})) {
			t.Fatal("global token environment or widened workflow permissions")
		}
		for _, j := range w.Jobs {
			if j.Permissions != nil && !reflect.DeepEqual(j.Permissions, map[string]string{"contents": "read"}) {
				t.Fatal("job permissions widened")
			}
			if j.Runner != "ubuntu-24.04" || len(j.Env) != 0 || j.Continue || (index == 0 && j.Timeout != 8) || (index == 1 && j.Timeout != 5) || len(j.Outputs) != 0 {
				t.Fatal("runner, credential output or failure boundary changed")
			}
			for _, s := range j.Steps {
				if index == 0 && s.ID == "candidate" {
					if s.Run == "" || s.Uses != "" || s.Continue || s.If != "" || strings.Contains(s.Run, "${{") {
						t.Fatal("untrusted expression or ignored candidate failure")
					}
					if !reflect.DeepEqual(s.Env, map[string]string{"GH_TOKEN": "${{ steps.app.outputs.token }}", "EXPECTED_HEAD": "${{ github.event.pull_request.head.sha }}", "PR_NUMBER": "${{ github.event.pull_request.number }}"}) {
						t.Fatal("candidate token/identity boundary changed")
					}
					continue
				}
				if s.Continue || !pin.MatchString(s.Uses) || s.Run != "" || s.If != "" || len(s.Env) != 0 {
					t.Fatal("unchecked Action, caller script, credential environment or ignored failure")
				}
				if strings.Contains(s.Uses, "artifact") {
					t.Fatal("PR artifact promotion")
				}
			}
		}
	}
	plan, apply := pr.Jobs["plan"], main.Jobs["apply"]
	if len(main.Jobs) != 1 || len(pr.Jobs) != 1 || len(plan.Steps) != 3 || len(apply.Steps) != 3 {
		t.Fatal("examples must keep one bounded plan/apply job")
	}
	if plan.If != "github.event_name == 'pull_request_target' && github.event.pull_request.base.ref == 'main'" || plan.Environment != "settings-plan" || plan.Needs != "" || pr.Defaults.Run.Shell != "bash" {
		t.Fatal("PR approval or execution boundary changed")
	}
	guard := "github.event_name == 'push' && github.ref == 'refs/heads/main'"
	if apply.If != guard || apply.Needs != "" || apply.Environment != "settings-apply" {
		t.Fatal("main event or approval boundary changed")
	}
	for _, j := range []job{apply} {
		checkout := j.Steps[0]
		if checkout.Uses != "actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1" || checkout.With["ref"] != "${{ github.sha }}" || checkout.With["persist-credentials"] != "false" {
			t.Fatal("checkout must use the exact triggering commit without persisted credentials")
		}
		if repository := checkout.With["repository"]; repository != "" && repository != "${{ github.repository }}" {
			t.Fatal("checkout target changed from the containing repository")
		}
		for key := range checkout.With {
			if key != "repository" && key != "ref" && key != "persist-credentials" {
				t.Fatal("unexpected checkout input")
			}
		}
	}
	if plan.Steps[2].Uses != exampleConsumerAction || !reflect.DeepEqual(plan.Steps[2].With, map[string]string{"command": "plan", "token": "${{ steps.app.outputs.token }}"}) {
		t.Fatal("PR pinned public Action/App-token binding changed")
	}
	planApp := plan.Steps[0]
	if planApp.ID != "app" || planApp.Uses != "actions/create-github-app-token@64ab729a6294000f5aa4a4bc0b73d1f32bc45510" || !reflect.DeepEqual(planApp.With, map[string]string{"client-id": "${{ vars.SETTINGS_APP_CLIENT_ID }}", "private-key": "${{ secrets.SETTINGS_APP_PRIVATE_KEY }}", "owner": "${{ github.repository_owner }}", "repositories": "${{ github.event.repository.name }}", "permission-contents": "read", "permission-administration": "read"}) {
		t.Fatal("plan App installation/permission scope changed")
	}
	app := apply.Steps[1]
	if app.ID != "app" || app.Uses != "actions/create-github-app-token@64ab729a6294000f5aa4a4bc0b73d1f32bc45510" || !reflect.DeepEqual(app.With, map[string]string{
		"client-id": "${{ vars.SETTINGS_APP_CLIENT_ID }}", "private-key": "${{ secrets.SETTINGS_APP_PRIVATE_KEY }}",
		"owner": "${{ github.repository_owner }}", "repositories": "${{ github.event.repository.name }}", "permission-administration": "write",
	}) {
		t.Fatal("caller App scope, credential binding or default revocation changed")
	}
	if apply.Steps[2].Uses != exampleConsumerAction || !reflect.DeepEqual(apply.Steps[2].With, map[string]string{"command": "apply", "token": "${{ steps.app.outputs.token }}"}) || strings.Count(mainText, "steps.app.outputs.token") != 1 {
		t.Fatal("App token escaped the apply step")
	}
}
