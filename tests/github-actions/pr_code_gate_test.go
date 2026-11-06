package githubactions

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"go.yaml.in/yaml/v4"
)

func TestPRWorkflowScreeningCheck(t *testing.T) {
	source, err := os.ReadFile(filepath.Join(root(), ".github/workflows/pr.yml"))
	if err != nil {
		t.Fatal(err)
	}
	var w struct {
		Name        string               `yaml:"name"`
		On          map[string]yaml.Node `yaml:"on"`
		Permissions map[string]string    `yaml:"permissions"`
		Jobs        map[string]yaml.Node `yaml:"jobs"`
		Concurrency struct {
			Group  string `yaml:"group"`
			Cancel bool   `yaml:"cancel-in-progress"`
		} `yaml:"concurrency"`
	}
	if err := yaml.Unmarshal(source, &w); err != nil {
		t.Fatal(err)
	}
	if w.Name != "pr" || len(w.On) != 1 || w.On["pull_request"].Tag != "!!null" {
		t.Fatal("pr must use default pull_request events without filters or extra event types")
	}
	if w.Concurrency.Group != "pr-${{ github.event.pull_request.number }}" || !w.Concurrency.Cancel {
		t.Fatal("each PR must validate only its current candidate")
	}
	if len(w.Jobs) != 1 || len(w.Permissions) != 0 {
		t.Fatal("PR validation must contain only screening with no workflow write permissions")
	}
	if strings.Contains(string(source), "/statuses/") || strings.Contains(string(source), "gh pr ready") {
		t.Fatal("screening must use its native job result without publishing statuses or changing PR state")
	}
	var check struct {
		Name        string            `yaml:"name"`
		Needs       yaml.Node         `yaml:"needs"`
		If          string            `yaml:"if"`
		Runner      string            `yaml:"runs-on"`
		Strategy    yaml.Node         `yaml:"strategy"`
		Continue    bool              `yaml:"continue-on-error"`
		Permissions map[string]string `yaml:"permissions"`
		Steps       []step            `yaml:"steps"`
	}
	checkNode, ok := w.Jobs["screening"]
	if !ok {
		t.Fatal("the native required check must be named screening")
	}
	if err := checkNode.Decode(&check); err != nil {
		t.Fatal(err)
	}
	if check.Name != "" || check.Needs.Kind != 0 || check.If != "" || check.Strategy.Kind != 0 || check.Continue ||
		check.Runner != "ubuntu-slim" || !reflect.DeepEqual(check.Permissions, map[string]string{"contents": "read"}) {
		t.Fatal("screening must run unconditionally as one read-only job and preserve failures")
	}
	for _, s := range check.Steps {
		if s.If != "" || s.Continue || s.Env["GH_TOKEN"] != "" || s.Env["GITHUB_TOKEN"] != "" {
			t.Fatalf("runtime screening steps must preserve failure propagation without supplied credentials: %+v", s)
		}
	}
	if len(check.Steps) == 0 || !strings.HasPrefix(check.Steps[0].Uses, "actions/checkout@") ||
		len(check.Steps[0].With) != 0 {
		t.Fatal("screening must check out the pull request candidate using default inputs")
	}
}
