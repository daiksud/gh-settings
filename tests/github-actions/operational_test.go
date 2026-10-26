package githubactions

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"go.yaml.in/yaml/v4"
)

func TestManagedRepositoryMergePreferences(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(root(), ".github/settings.yml"))
	if err != nil {
		t.Fatal(err)
	}
	var config struct {
		Settings struct {
			Repository struct {
				AllowMergeCommit       *bool   `yaml:"allow_merge_commit"`
				AllowSquashMerge       *bool   `yaml:"allow_squash_merge"`
				AllowRebaseMerge       *bool   `yaml:"allow_rebase_merge"`
				SquashMergeCommitTitle string  `yaml:"squash_merge_commit_title"`
				SquashMergeCommitBody  string  `yaml:"squash_merge_commit_message"`
				MergeCommitTitle       *string `yaml:"merge_commit_title"`
				MergeCommitBody        *string `yaml:"merge_commit_message"`
			} `yaml:"repository"`
		} `yaml:"settings"`
	}
	if err := yaml.Unmarshal(data, &config); err != nil {
		t.Fatal(err)
	}
	repository := config.Settings.Repository
	if repository.AllowMergeCommit == nil || *repository.AllowMergeCommit || repository.AllowSquashMerge == nil || !*repository.AllowSquashMerge || repository.AllowRebaseMerge == nil || *repository.AllowRebaseMerge {
		t.Fatal("managed repository must allow squash merging only")
	}
	if repository.SquashMergeCommitTitle != "PR_TITLE" || repository.SquashMergeCommitBody != "PR_BODY" {
		t.Fatalf("managed squash messages are not PR title/body: %+v", repository)
	}
	if repository.MergeCommitTitle != nil || repository.MergeCommitBody != nil {
		t.Fatalf("disabled normal-merge metadata must remain unmanaged: %+v", repository)
	}
}

func TestOperationalPRPlan(t *testing.T) {
	w, text := readWorkflow(t, "../../.github/workflows/settings-pr.yml")
	j := w.Jobs["plan"]
	if len(j.Steps) != 3 || j.Steps[2].Uses != exampleConsumerAction || !reflect.DeepEqual(j.Steps[2].With, map[string]string{"command": "plan", "token": "${{ steps.app.outputs.token }}"}) {
		t.Fatal("operational PR must call the released public Action with its explicit App token")
	}
	if len(w.On) != 1 || w.On["pull_request_target"].Kind == 0 || len(w.Jobs) != 1 || j.Environment != "gh-settings" || j.If != settingsPRCondition || len(j.Env) != 0 || len(j.Outputs) != 0 || j.Continue || j.Runner != "ubuntu-24.04" || j.Timeout != 8 {
		t.Fatal("operational PR trust boundary changed")
	}
	if len(w.Permissions) != 0 || w.Defaults.Run.Shell != "bash" || !w.Concurrency.Cancel || w.Concurrency.Group != "settings-pr-${{ github.repository }}-${{ github.event.pull_request.number }}" {
		t.Fatal("operational PR permissions or concurrency changed")
	}
	if j.Steps[0].ID != "candidate" || j.Steps[0].Uses != "" || !reflect.DeepEqual(j.Steps[0].Env, map[string]string{"EXPECTED_HEAD": "${{ github.event.pull_request.head.sha }}", "PR_NUMBER": "${{ github.event.pull_request.number }}"}) || strings.Contains(text, "actions/checkout@") || strings.Contains(text, "github.token") {
		t.Fatal("candidate data must be prepared before credentials without checkout or token fallback")
	}
	app := j.Steps[1]
	if app.ID != "app" || app.Uses != "actions/create-github-app-token@64ab729a6294000f5aa4a4bc0b73d1f32bc45510" || !reflect.DeepEqual(app.With, map[string]string{
		"client-id": "${{ vars.ANSWER26_CLIENT_ID }}", "private-key": "${{ secrets.ANSWER26_PRIVATE_KEY }}",
		"owner": "${{ github.repository_owner }}", "repositories": "${{ github.event.repository.name }}",
		"permission-administration": "read", "permission-issues": "read",
	}) {
		t.Fatal("operational PR must use the existing repository-scoped App without new grants")
	}
	for _, scenario := range []struct {
		name, outcome string
		status        int
	}{
		{"differences", "changed", 0},
		{"no differences", "no-difference", 0},
		{"invalid configuration", "", 37},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			f := newCompositeFixture(t)
			f.base["GITHUB_EVENT_NAME"] = "pull_request_target"
			f.base["GITHUB_WORKFLOW_REF"] = "owner/config/.github/workflows/settings-pr.yml@refs/heads/main"
			f.base["GITHUB_WORKFLOW_SHA"] = strings.Repeat("b", 40)
			f.base["MOCK_PLAN_OUTCOME"] = scenario.outcome
			f.base["MOCK_VALIDATE_STATUS"] = fmt.Sprint(scenario.status)
			r := f.call(t, j.Steps[2], map[string]string{"steps.app.outputs.token": "app-fixture"}, nil)
			if r.status != scenario.status || r.api != "" || strings.Contains(r.log+r.summary+fmt.Sprint(r.outputs), "app-fixture") {
				t.Fatalf("operational PR result: %+v", r)
			}
			if scenario.status == 0 {
				if r.calls != "plan\n" || r.outputs["config-sha"] != f.sha || r.outputs["target"] != "owner/config" || r.outputs["cli-version"] != committedCLIVersion(t) || r.outputs["cli-sha256"] != f.digest || !strings.Contains(r.summary, strings.TrimPrefix(exampleConsumerAction, "daiksud/gh-settings@")) {
					t.Fatalf("operational PR identity or execution: %+v", r)
				}
			} else if r.calls != "plan\n" || len(r.outputs) != 0 {
				t.Fatalf("invalid configuration reached plan or published outputs: %+v", r)
			}
		})
	}
}

func TestOperationalMainConsumer(t *testing.T) {
	w, source := readWorkflow(t, "../../.github/workflows/settings-main.yml")
	apply := w.Jobs["apply"]
	if len(w.On) != 1 || w.On["push"].Kind == 0 || len(w.Jobs) != 1 || len(apply.Steps) != 3 {
		t.Fatal("operational main must use one job with checkout, App token and apply")
	}
	if apply.If != "github.event_name == 'push' && github.ref == 'refs/heads/main'" || apply.Environment != "gh-settings" || apply.Needs != "" ||
		apply.Runner != "ubuntu-24.04" || apply.Timeout != 5 || apply.Continue || len(apply.Env) != 0 || len(apply.Outputs) != 0 ||
		!reflect.DeepEqual(w.Permissions, map[string]string{"contents": "read"}) || w.Concurrency.Cancel ||
		w.Concurrency.Group != "settings-main-${{ github.repository }}" {
		t.Fatal("operational main approval, permissions or execution boundary changed")
	}
	if apply.Steps[0].Uses != "actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1" ||
		!reflect.DeepEqual(apply.Steps[0].With, map[string]string{"ref": "${{ github.sha }}", "persist-credentials": "false"}) {
		t.Fatal("operational main must check out the triggering commit without credentials")
	}
	appToken := apply.Steps[1]
	if appToken.ID != "app" || appToken.Uses != "actions/create-github-app-token@64ab729a6294000f5aa4a4bc0b73d1f32bc45510" ||
		!reflect.DeepEqual(appToken.With, map[string]string{
			"client-id":                 "${{ vars.ANSWER26_CLIENT_ID }}",
			"private-key":               "${{ secrets.ANSWER26_PRIVATE_KEY }}",
			"owner":                     "${{ github.repository_owner }}",
			"repositories":              "${{ github.event.repository.name }}",
			"permission-administration": "write",
			"permission-issues":         "write",
		}) {
		t.Fatal("operational App credentials, target scope or revocation changed")
	}
	if apply.Steps[2].Uses != exampleAction || !reflect.DeepEqual(apply.Steps[2].With, map[string]string{
		"command": "apply", "token": "${{ steps.app.outputs.token }}",
	}) || strings.Count(source, "steps.app.outputs.token") != 1 || strings.Contains(source, "verify-settings-handoff") ||
		strings.Contains(source, "run:") || !strings.Contains(source, "branches: [main]") {
		t.Fatal("operational action identity, binding or simplified workflow changed")
	}
	for _, test := range []struct {
		scenario string
		status   int
		minted   bool
		api      string
	}{
		{"approved", 0, true, "PATCH\n"},
		{"apply failure", 37, true, "PATCH\n"},
		{"invalid configuration", 37, true, ""},
		{"apply installation failure", 1, true, ""},
		{"waiting", 0, false, ""},
		{"rejected", 0, false, ""},
		{"pull request", 0, false, ""},
		{"other branch", 0, false, ""},
		{"tag push", 0, false, ""},
		{"mint failure", 1, false, ""},
	} {
		t.Run(test.scenario, func(t *testing.T) {
			result := runMainConsumer(t, test.scenario, "../../.github/workflows/settings-main.yml")
			result.assertNoCredentials(t)
			if result.status != test.status || result.minted != test.minted || result.revoked != test.minted || result.applied.api != test.api {
				t.Fatalf("operational apply scenario failed: %+v", result)
			}
		})
	}
}
