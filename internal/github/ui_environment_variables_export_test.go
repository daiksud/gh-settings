package github

import (
	"github.com/daiksud/gh-settings/internal/settings"
	"testing"
)

func TestEnvironmentVariablesExportIndependentScopes(t *testing.T) {
	empty := ""
	topics := []string{}
	timer := 0
	no := false
	yes := true
	reviewers := []settings.EnvironmentReviewer{}
	policy, err := settings.NewEnvironmentDeclarations([]settings.EnvironmentDeclaration{{Name: "Production", WaitTimer: &timer, PreventSelfReview: &no, CanAdminsBypass: &yes, Reviewers: &reviewers, DeploymentBranchPolicyDeclared: true}})
	if err != nil {
		t.Fatal(err)
	}
	variables, _ := settings.NewVariableDeclarations([]settings.RepositoryVariable{{Name: "A", Value: "production"}})
	staging, _ := settings.NewVariableDeclarations([]settings.RepositoryVariable{{Name: "A", Value: "staging"}})
	ui, err := UIFromExportDeclaration(settings.Declaration{Repository: settings.RepositoryDeclaration{Description: &empty, Homepage: &empty}, Topics: settings.TopicsDeclaration{Names: &topics}, Environments: &policy, EnvironmentVariables: []settings.EnvironmentVariableDeclaration{{Environment: "production", Variables: variables}, {Environment: "Staging", Variables: staging}}})
	if err != nil {
		t.Fatal(err)
	}
	environments := *ui.Settings.Environments
	if len(environments) != 2 || environments[0].Name != "Production" || environments[0].Variables == nil || environments[0].DeploymentProtectionRules == nil || environments[1].DeploymentProtectionRules != nil || environments[1].DeploymentBranchesAndTags != nil {
		t.Fatalf("projection changed policy ownership: %#v", environments)
	}
	d, err := DeclarationFromUI(ui)
	if err != nil || d.Environments.Len() != 1 || len(d.EnvironmentVariables) != 2 || d.EnvironmentVariables[1].Variables.Values()[0].Value != "staging" {
		t.Fatalf("reimport: %#v %v", d, err)
	}
}
