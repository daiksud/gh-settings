package githubactions

import (
	"reflect"
	"strings"
	"testing"

	"github.com/daiksud/gh-settings/internal/settings"
)

const operationalAction = "daiksud/gh-settings@01cd465f5c4ff506b39de70261707d5c6fa22943"

func TestManagedRepositoryEnvironments(t *testing.T) {
	declaration := readManagedDeclaration(t)
	if len(declaration.EnvironmentVariables) != 0 {
		t.Fatal("credential Environment variables must remain unmanaged")
	}
	if declaration.Variables == nil || declaration.Variables.Len() != 0 ||
		declaration.AgentsVariables == nil || declaration.AgentsVariables.Len() != 0 {
		t.Fatal("repository Actions and Agents variables must both be explicitly empty")
	}
	if declaration.Environments == nil || declaration.Environments.Len() != 1 {
		t.Fatal("only the settings credential Environment must be declared")
	}
	want := settings.EnvironmentDeclaration{
		Name:                           "gh-settings",
		Reviewers:                      &[]settings.EnvironmentReviewer{{Type: "User", ID: 155234749}},
		PreventSelfReview:              new(false),
		WaitTimer:                      new(0),
		CustomRules:                    &[]int64{},
		CanAdminsBypass:                new(false),
		DeploymentBranchPolicyDeclared: true,
		DeploymentBranchPolicy:         &settings.EnvironmentBranchPolicy{CustomBranchPolicies: true},
		DeploymentBranchPatterns:       &[]settings.EnvironmentPattern{{Type: "branch", Name: "main"}},
	}
	if got := declaration.Environments.Values()[0]; !reflect.DeepEqual(got, want) {
		t.Fatalf("credential Environment protections or deployment refs changed: %+v", got)
	}
}

func TestManagedRepositoryMergePreferences(t *testing.T) {
	repository := readManagedDeclaration(t).Repository
	if repository.AllowMergeCommit == nil || *repository.AllowMergeCommit || repository.AllowSquashMerge == nil || !*repository.AllowSquashMerge || repository.AllowRebaseMerge == nil || *repository.AllowRebaseMerge {
		t.Fatal("managed repository must allow squash merging only")
	}
	if repository.SquashMergeCommitTitle == nil || *repository.SquashMergeCommitTitle != "PR_TITLE" || repository.SquashMergeCommitMessage == nil || *repository.SquashMergeCommitMessage != "PR_BODY" {
		t.Fatalf("managed squash messages are not PR title/body: %+v", repository)
	}
	if repository.MergeCommitTitle != nil || repository.MergeCommitMessage != nil {
		t.Fatalf("disabled normal-merge metadata must remain unmanaged: %+v", repository)
	}
}

func TestOperationalMainConsumer(t *testing.T) {
	w, source := readWorkflow(t, "../../.github/workflows/settings-apply.yml")
	apply := w.Jobs["settings-apply"]
	if len(w.On) != 1 || w.On["workflow_dispatch"].Kind == 0 || len(w.Jobs) != 1 || len(apply.Steps) != 3 {
		t.Fatal("operational manual apply must use one job with checkout, App token and apply")
	}
	if apply.If != "github.event_name == 'workflow_dispatch' && github.ref == 'refs/heads/main'" || apply.Environment != "gh-settings" || apply.Needs != "" ||
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
	if appToken.ID != "app" || !strings.HasPrefix(appToken.Uses, "actions/create-github-app-token@") ||
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
	if apply.Steps[2].Uses != operationalAction || !reflect.DeepEqual(apply.Steps[2].With, map[string]string{
		"command": "apply", "token": "${{ steps.app.outputs.token }}",
	}) || strings.Count(source, "steps.app.outputs.token") != 1 ||
		strings.Contains(source, "run:") || !strings.Contains(source, "workflow_dispatch:") {
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
		{"automatic push", 0, false, ""},
		{"pull request", 0, false, ""},
		{"other branch", 0, false, ""},
		{"tag push", 0, false, ""},
		{"mint failure", 1, false, ""},
	} {
		t.Run(test.scenario, func(t *testing.T) {
			result := runMainConsumer(t, test.scenario, "../../.github/workflows/settings-apply.yml")
			result.assertNoCredentials(t)
			if result.status != test.status || result.minted != test.minted || result.revoked != test.minted || result.applied.api != test.api {
				t.Fatalf("operational apply scenario failed: %+v", result)
			}
		})
	}
}
