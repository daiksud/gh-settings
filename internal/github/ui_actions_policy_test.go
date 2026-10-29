package github

import (
	"reflect"
	"testing"

	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

func TestActionsExecutionPolicyUI(t *testing.T) {
	policy := settings.ActionsPolicyDeclaration{Enabled: rulesetPtr(true), AllowedActions: rulesetPtr("selected"), SHAPinningRequired: rulesetPtr(false), SelectedActions: settings.SelectedActionsDeclaration{GitHubOwnedAllowed: rulesetPtr(false), VerifiedAllowed: rulesetPtr(true), PatternsAllowed: rulesetPtr([]string{"example/a@v1", "!example/b@*", "example/a@v1"})}}
	want := uimodel.ActionsGeneral{ActionsPermissions: rulesetPtr(uimodel.AllowOwnerAndSelectNonOwnerActionsAndReusableWorkflows), RequireActionsToBePinnedToAFullLengthCommitSHA: rulesetPtr(false), AllowActionsCreatedByGitHub: rulesetPtr(false), AllowActionsByMarketplaceVerifiedCreators: rulesetPtr(true), AllowOrBlockSpecifiedActionsAndReusableWorkflows: rulesetPtr([]string{"example/a@v1", "!example/b@*", "example/a@v1"})}
	got, err := ActionsExecutionPolicyUI(policy)
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("Actions execution projection %#v; want %#v; error=%v", got, want, err)
	}
	(*got.AllowOrBlockSpecifiedActionsAndReusableWorkflows)[0] = "changed"
	if (*policy.SelectedActions.PatternsAllowed)[0] != "example/a@v1" {
		t.Fatal("Actions projection aliases policy")
	}
	*policy.Enabled = false
	got, err = ActionsExecutionPolicyUI(policy)
	want = uimodel.ActionsGeneral{ActionsPermissions: rulesetPtr(uimodel.DisableActions), RequireActionsToBePinnedToAFullLengthCommitSHA: rulesetPtr(false)}
	if err != nil || !reflect.DeepEqual(got, want) || policy.SelectedActions.PatternsAllowed == nil {
		t.Fatalf("disabled latent policy became visible: %#v, %v", got, err)
	}
	got, err = ActionsExecutionPolicyUI(settings.ActionsPolicyDeclaration{})
	if err != nil || !reflect.DeepEqual(got, uimodel.ActionsGeneral{}) {
		t.Fatalf("omitted execution became desired defaults: %#v, %v", got, err)
	}
}

func TestActionsExecutionPolicyUIRejectsIncompletePolicy(t *testing.T) {
	complete := settings.ActionsPolicyDeclaration{Enabled: rulesetPtr(true), AllowedActions: rulesetPtr("selected"), SHAPinningRequired: rulesetPtr(false), SelectedActions: settings.SelectedActionsDeclaration{GitHubOwnedAllowed: rulesetPtr(false), VerifiedAllowed: rulesetPtr(true), PatternsAllowed: rulesetPtr([]string{})}}
	for _, mutate := range []func(*settings.ActionsPolicyDeclaration){
		func(p *settings.ActionsPolicyDeclaration) { p.Enabled = nil },
		func(p *settings.ActionsPolicyDeclaration) { p.AllowedActions = nil },
		func(p *settings.ActionsPolicyDeclaration) { p.SHAPinningRequired = nil },
		func(p *settings.ActionsPolicyDeclaration) { p.SelectedActions.GitHubOwnedAllowed = nil },
		func(p *settings.ActionsPolicyDeclaration) { p.SelectedActions.VerifiedAllowed = nil },
		func(p *settings.ActionsPolicyDeclaration) { p.SelectedActions.PatternsAllowed = nil },
		func(p *settings.ActionsPolicyDeclaration) {
			p.Enabled, p.AllowedActions, p.SHAPinningRequired = nil, nil, nil
		},
	} {
		policy := complete
		mutate(&policy)
		got, err := ActionsExecutionPolicyUI(policy)
		if err == nil || !reflect.DeepEqual(got, uimodel.ActionsGeneral{}) {
			t.Fatalf("incomplete Actions projected: %#v, %v", got, err)
		}
	}
}

func TestActionsWorkflowPolicyUI(t *testing.T) {
	base := uimodel.ActionsGeneral{ActionsPermissions: rulesetPtr(uimodel.DisableActions)}
	for _, test := range []struct {
		api string
		ui  uimodel.WorkflowPermissionChoice
	}{
		{"read", uimodel.ReadRepositoryContentsAndPackagesPermissions}, {"write", uimodel.ReadAndWritePermissions},
	} {
		got, err := WithActionsWorkflowPolicyUI(base, settings.ActionsPolicyDeclaration{DefaultWorkflowPermissions: &test.api, CanApprovePullRequestReviews: rulesetPtr(false)})
		want := uimodel.ActionsGeneral{ActionsPermissions: rulesetPtr(uimodel.DisableActions), WorkflowPermissions: &test.ui, AllowGitHubActionsToCreateAndApprovePullRequests: rulesetPtr(false)}
		if err != nil || !reflect.DeepEqual(got, want) {
			t.Fatalf("workflow projection %#v; want %#v; error=%v", got, want, err)
		}
	}
	got, err := WithActionsWorkflowPolicyUI(base, settings.ActionsPolicyDeclaration{})
	if err != nil || !reflect.DeepEqual(got, base) {
		t.Fatalf("omitted workflow changed controls: %#v, %v", got, err)
	}
	for _, policy := range []settings.ActionsPolicyDeclaration{
		{DefaultWorkflowPermissions: rulesetPtr("read")}, {CanApprovePullRequestReviews: rulesetPtr(false)},
		{DefaultWorkflowPermissions: rulesetPtr("unknown"), CanApprovePullRequestReviews: rulesetPtr(false)},
	} {
		got, err := WithActionsWorkflowPolicyUI(base, policy)
		if err == nil || !reflect.DeepEqual(got, uimodel.ActionsGeneral{}) {
			t.Fatalf("incomplete workflow projected: %#v, %v", got, err)
		}
	}
}

func TestActionsRetentionPolicyUI(t *testing.T) {
	base := uimodel.ActionsGeneral{AllowGitHubActionsToCreateAndApprovePullRequests: rulesetPtr(false)}
	for _, days := range []int{1, 30, 400} {
		got, err := WithActionsRetentionPolicyUI(base, &days)
		want := uimodel.ActionsGeneral{AllowGitHubActionsToCreateAndApprovePullRequests: rulesetPtr(false), CheckWorkflowRunStatusArtifactAndLogRetention: rulesetPtr(days)}
		if err != nil || !reflect.DeepEqual(got, want) {
			t.Fatalf("retention projection %#v; want %#v; error=%v", got, want, err)
		}
		*got.CheckWorkflowRunStatusArtifactAndLogRetention = 15
		if days != *want.CheckWorkflowRunStatusArtifactAndLogRetention {
			t.Fatal("retention projection aliases actual days")
		}
	}
	got, err := WithActionsRetentionPolicyUI(base, nil)
	if err != nil || !reflect.DeepEqual(got, base) {
		t.Fatalf("omitted retention changed controls: %#v, %v", got, err)
	}
	for _, days := range []int{0, 401} {
		got, err := WithActionsRetentionPolicyUI(base, &days)
		if err == nil || !reflect.DeepEqual(got, uimodel.ActionsGeneral{}) {
			t.Fatalf("invalid retention projected: %#v, %v", got, err)
		}
	}
}
