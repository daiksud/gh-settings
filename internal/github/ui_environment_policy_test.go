package github

import (
	"reflect"
	"testing"

	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

func TestEnvironmentPolicyUI(t *testing.T) {
	policy := settings.EnvironmentDeclaration{Name: "Production", WaitTimer: rulesetPtr(0), PreventSelfReview: rulesetPtr(false), CanAdminsBypass: rulesetPtr(true), Reviewers: rulesetPtr([]settings.EnvironmentReviewer{{Type: "Team", ID: 12}, {Type: "User", ID: 11}}), DeploymentBranchPolicyDeclared: true, DeploymentBranchPolicy: &settings.EnvironmentBranchPolicy{CustomBranchPolicies: true}, DeploymentBranchPatterns: rulesetPtr([]settings.EnvironmentPattern{{Type: "branch", Name: "Release/*"}, {Type: "tag", Name: "Release/*"}})}
	want := uimodel.Environment{Name: "Production", DeploymentProtectionRules: &uimodel.EnvironmentDeploymentProtectionRules{WaitTimer: rulesetPtr(0), PreventSelfReview: rulesetPtr(false), AllowAdministratorsToBypassConfiguredProtectionRules: rulesetPtr(true), RequiredReviewers: rulesetPtr([]uimodel.EnvironmentReviewer{{Team: rulesetPtr(int64(12))}, {Person: rulesetPtr(int64(11))}})}, DeploymentBranchesAndTags: rulesetPtr(uimodel.SelectedBranchesAndTags), DeploymentBranchOrTagRules: rulesetPtr([]uimodel.EnvironmentDeploymentRule{{RefType: uimodel.EnvironmentBranch, NamePattern: "Release/*"}, {RefType: uimodel.EnvironmentTag, NamePattern: "Release/*"}})}
	got, err := EnvironmentPolicyUI(policy)
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("policy projection %#v; want %#v; error=%v", got, want, err)
	}
	*got.DeploymentProtectionRules.WaitTimer = 15
	*(*got.DeploymentProtectionRules.RequiredReviewers)[0].Team = 99
	(*got.DeploymentBranchOrTagRules)[0].NamePattern = "changed"
	if *policy.WaitTimer != 0 || (*policy.Reviewers)[0].ID != 12 || (*policy.DeploymentBranchPatterns)[0].Name != "Release/*" {
		t.Fatal("policy projection aliases caller")
	}
}

func TestEnvironmentPolicyUIRejectsIncompletePolicies(t *testing.T) {
	complete := uiEnvironmentObservedPolicy("Production")
	for _, mutate := range []func(*settings.EnvironmentDeclaration){
		func(p *settings.EnvironmentDeclaration) { p.WaitTimer = nil },
		func(p *settings.EnvironmentDeclaration) { p.Reviewers = nil },
		func(p *settings.EnvironmentDeclaration) { p.CanAdminsBypass = nil },
		func(p *settings.EnvironmentDeclaration) { p.PreventSelfReview = nil },
		func(p *settings.EnvironmentDeclaration) { p.DeploymentBranchPolicyDeclared = false },
		func(p *settings.EnvironmentDeclaration) {
			p.DeploymentBranchPolicy = &settings.EnvironmentBranchPolicy{CustomBranchPolicies: true}
		},
		func(p *settings.EnvironmentDeclaration) {
			p.Reviewers = rulesetPtr([]settings.EnvironmentReviewer{{Type: "App", ID: 17}})
		},
	} {
		policy := complete
		mutate(&policy)
		got, err := EnvironmentPolicyUI(policy)
		if err == nil || !reflect.DeepEqual(got, uimodel.Environment{}) {
			t.Fatalf("incomplete policy projected: %#v, %v", got, err)
		}
	}
}
