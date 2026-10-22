package githubactions

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"go.yaml.in/yaml/v4"
)

func TestPRMessageWorkflowIsIndependentAndReadOnly(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(root(), ".github/workflows/pr-message.yml"))
	if err != nil {
		t.Fatal(err)
	}
	var w struct {
		On map[string]struct {
			Types []string `yaml:"types"`
		} `yaml:"on"`
		Permissions map[string]string `yaml:"permissions"`
		Concurrency struct {
			Group  string `yaml:"group"`
			Cancel bool   `yaml:"cancel-in-progress"`
		} `yaml:"concurrency"`
		Jobs map[string]struct {
			Permissions map[string]string `yaml:"permissions"`
			Timeout     int               `yaml:"timeout-minutes"`
			Steps       []struct {
				Uses     string            `yaml:"uses"`
				Run      string            `yaml:"run"`
				With     map[string]string `yaml:"with"`
				Env      map[string]string `yaml:"env"`
				Continue bool              `yaml:"continue-on-error"`
			} `yaml:"steps"`
		} `yaml:"jobs"`
	}
	if err := yaml.Unmarshal(data, &w); err != nil {
		t.Fatal(err)
	}
	if len(w.On) != 1 || !slices.Equal(w.On["pull_request"].Types, []string{"opened", "synchronize", "reopened", "edited"}) {
		t.Fatal("messages must be checked on head/base changes and title/body edits")
	}
	if len(w.Permissions) != 0 || w.Concurrency.Group != "pr-message-${{ github.event.pull_request.number }}" || !w.Concurrency.Cancel {
		t.Fatal("message checks need an isolated latest-wins group, not the code-validation group")
	}
	job, ok := w.Jobs["pr-message"]
	if !ok || len(w.Jobs) != 1 || job.Timeout != 5 || !reflect.DeepEqual(job.Permissions, map[string]string{"contents": "read"}) || len(job.Steps) != 4 {
		t.Fatal("require one bounded read-only message job")
	}
	if job.Steps[0].Uses != "actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1" || job.Steps[0].With["persist-credentials"] != "false" {
		t.Fatal("message source checkout must use the pinned action without persisted credentials")
	}
	if job.Steps[1].Uses != "jdx/mise-action@2d8d4cafcbd33be2ea37d2b6f5ad595363d1f1ca" || job.Steps[1].With["install_args"] != "bun npm:@commitlint/cli" {
		t.Fatal("install only the existing pinned message tools")
	}
	if job.Steps[2].Run != "bash tests/github-actions/commitlint.sh" {
		t.Fatal("keep the existing message-policy regression suite")
	}
	validation := job.Steps[3]
	if !reflect.DeepEqual(validation.Env, map[string]string{
		"PR_TITLE": "${{ github.event.pull_request.title }}",
		"PR_BODY":  "${{ github.event.pull_request.body }}",
	}) || !strings.Contains(validation.Run, `"$PR_TITLE" "$PR_BODY"`) || !strings.Contains(validation.Run, "--strict") || strings.Contains(validation.Run, "${{") {
		t.Fatal("validate the complete message using quoted environment data, not shell interpolation")
	}
	for _, step := range job.Steps {
		if step.Continue {
			t.Fatal("invalid messages must fail the independent check")
		}
	}
	for _, forbidden := range []string{"secrets.", "gh pr ready", "acceptance-stage", "candidate-smoke", "upload-artifact", "download-artifact", "environment:"} {
		if strings.Contains(string(data), forbidden) {
			t.Fatalf("message validation must not run state transitions or product/privileged work: %s", forbidden)
		}
	}
}

func TestSettingsEnvironmentScopeBoundedCodeGateName(t *testing.T) {
	for _, test := range []struct {
		name, path, job, expression, extra string
		valid                              bool
	}{
		{"two literal outcomes", ".github/workflows/ci-pr.yml", "pass", prCodeGateName, "", true},
		{"another workflow", ".github/workflows/other.yml", "pass", prCodeGateName, "", false},
		{"another job", ".github/workflows/ci-pr.yml", "other", prCodeGateName, "", false},
		{"reserved job", ".github/workflows/ci-pr.yml", "gh-settings", prCodeGateName, "", false},
		{"reserved outcome", ".github/workflows/ci-pr.yml", "pass", strings.Replace(prCodeGateName, "'metadata-only'", "'gh-settings'", 1), "", false},
		{"arbitrary expression", ".github/workflows/ci-pr.yml", "pass", "${{ vars.CHECK_NAME }}", "", false},
		{"privileged environment", ".github/workflows/ci-pr.yml", "pass", prCodeGateName, "    environment: gh-settings\n", false},
		{"hidden reusable environment", ".github/workflows/ci-pr.yml", "pass", prCodeGateName, "    uses: example/repo/.github/workflows/other.yml@main\n", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			files := map[string][]byte{
				settingsWorkflowPath:      []byte("on: {push: {branches: [main]}}\njobs:\n  apply:\n    environment: gh-settings\n"),
				settingsGuardWorkflowPath: []byte("jobs:\n  gh-settings:\n    name: gh-settings\n"),
				test.path:                 []byte(fmt.Sprintf("jobs:\n  %s:\n    name: \"%s\"\n%s", test.job, test.expression, test.extra)),
			}
			err := checkSettingsEnvironmentScope(files)
			if (err == nil) != test.valid {
				t.Fatalf("valid=%t, got %v", test.valid, err)
			}
		})
	}
}
