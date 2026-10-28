package github

import (
	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/settings/uimodel"
	"reflect"
	"testing"
)

func TestUIEnvironmentDeploymentChoices(t *testing.T) {
	for _, test := range []struct {
		ui     string
		policy *settings.EnvironmentBranchPolicy
	}{
		{"no_restriction", nil},
		{"protected_branches_only", &settings.EnvironmentBranchPolicy{ProtectedBranches: true, CustomBranchPolicies: false}},
		{"selected_branches_and_tags", &settings.EnvironmentBranchPolicy{ProtectedBranches: false, CustomBranchPolicies: true}},
	} {
		got, err := EnvironmentDeclaration(uimodel.Environment{Name: "Production", DeploymentBranchesAndTags: environmentUIPtr(uimodel.EnvironmentDeploymentChoice(test.ui))})
		if err != nil || !reflect.DeepEqual(got, settings.EnvironmentDeclaration{Name: "Production", DeploymentBranchPolicy: test.policy, DeploymentBranchPolicyDeclared: true}) {
			t.Fatal(got, err)
		}
	}
	if _, err := EnvironmentDeclaration(uimodel.Environment{Name: "Production", DeploymentBranchesAndTags: environmentUIPtr(uimodel.EnvironmentDeploymentChoice("custom_branch_policies"))}); err == nil {
		t.Fatal("API token accepted")
	}
}
func TestUIEnvironmentTypedPatternsAndOwnership(t *testing.T) {
	patterns := []uimodel.EnvironmentDeploymentRule{{RefType: "tag", NamePattern: "Release/*"}, {RefType: "branch", NamePattern: "Release/*"}}
	got, err := EnvironmentDeclaration(uimodel.Environment{Name: "Production", DeploymentBranchOrTagRules: &patterns})
	want := []settings.EnvironmentPattern{{Type: "branch", Name: "Release/*"}, {Type: "tag", Name: "Release/*"}}
	if err != nil || !reflect.DeepEqual(*got.DeploymentBranchPatterns, want) || got.DeploymentBranchPolicyDeclared {
		t.Fatal(got, err)
	}
	patterns[0].NamePattern = "changed"
	if !reflect.DeepEqual(*got.DeploymentBranchPatterns, want) {
		t.Fatal("pattern aliases caller")
	}
	empty, err := EnvironmentDeclaration(uimodel.Environment{Name: "Production", DeploymentBranchesAndTags: environmentUIPtr(uimodel.EnvironmentDeploymentChoice("selected_branches_and_tags")), DeploymentBranchOrTagRules: environmentUIPtr([]uimodel.EnvironmentDeploymentRule{})})
	if err != nil || empty.DeploymentBranchPatterns == nil || len(*empty.DeploymentBranchPatterns) != 0 {
		t.Fatal(empty, err)
	}
	for _, mode := range []uimodel.EnvironmentDeploymentChoice{"no_restriction", "protected_branches_only"} {
		if _, err := EnvironmentDeclaration(uimodel.Environment{Name: "Production", DeploymentBranchesAndTags: &mode, DeploymentBranchOrTagRules: environmentUIPtr([]uimodel.EnvironmentDeploymentRule{})}); err == nil {
			t.Fatal("incompatible mode accepted")
		}
	}
	for _, entries := range [][]uimodel.EnvironmentDeploymentRule{
		{{RefType: "future", NamePattern: "main"}}, {{RefType: "branch", NamePattern: ""}}, {{RefType: "branch", NamePattern: "main"}, {RefType: "branch", NamePattern: "main"}},
	} {
		if _, err := EnvironmentDeclaration(uimodel.Environment{Name: "Production", DeploymentBranchOrTagRules: &entries}); err == nil {
			t.Fatal("invalid patterns accepted")
		}
	}
	desired, err := EnvironmentDeclarations([]uimodel.Environment{{Name: "Production", DeploymentBranchOrTagRules: environmentUIPtr([]uimodel.EnvironmentDeploymentRule{})}})
	if err != nil {
		t.Fatal(err)
	}
	complete := settings.EnvironmentDeclaration{Name: "Production", WaitTimer: environmentUIPtr(0), Reviewers: environmentUIPtr([]settings.EnvironmentReviewer{}), PreventSelfReview: environmentUIPtr(false), CanAdminsBypass: environmentUIPtr(false), DeploymentBranchPolicyDeclared: true}
	observed, err := settings.NewEnvironmentObservation([]settings.EnvironmentState{{ID: 7, Policy: complete}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := settings.ReconcileEnvironments(observed, &desired); err == nil {
		t.Fatal("mode prerequisite invented")
	}
	complete.DeploymentBranchPolicy = &settings.EnvironmentBranchPolicy{CustomBranchPolicies: true}
	observed, err = settings.NewEnvironmentObservation([]settings.EnvironmentState{{ID: 7, Policy: complete}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := settings.ReconcileEnvironments(observed, &desired); err != nil {
		t.Fatal(err)
	}
	if _, err := EnvironmentDeclarations([]uimodel.Environment{{Name: "Production"}, {Name: "production"}}); err == nil {
		t.Fatal("duplicate identity accepted")
	}
}
