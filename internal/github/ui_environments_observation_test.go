package github

import (
	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/settings/uimodel"
	"reflect"
	"testing"
)

func uiEnvironmentObservedPolicy(name string) settings.EnvironmentDeclaration {
	return settings.EnvironmentDeclaration{Name: name, WaitTimer: environmentUIPtr(0), PreventSelfReview: environmentUIPtr(false), CanAdminsBypass: environmentUIPtr(false), Reviewers: environmentUIPtr([]settings.EnvironmentReviewer{}), DeploymentBranchPolicyDeclared: true}
}
func TestUIEnvironmentCollectionPresence(t *testing.T) {
	absent, err := EnvironmentCollectionDeclaration(nil)
	if err != nil || absent != nil {
		t.Fatal(absent, err)
	}
	empty := []uimodel.Environment{}
	present, err := EnvironmentCollectionDeclaration(&empty)
	if err != nil || present == nil || present.Len() != 0 {
		t.Fatal(present, err)
	}
}
func TestUIEnvironmentAcquiredCollectionAndMetadata(t *testing.T) {
	plain := uiEnvironmentObservedPolicy("Zed")
	protected := uiEnvironmentObservedPolicy("Beta")
	protected.DeploymentBranchPolicy = &settings.EnvironmentBranchPolicy{ProtectedBranches: true}
	protected.WaitTimer = environmentUIPtr(15)
	protected.CanAdminsBypass = environmentUIPtr(true)
	custom := uiEnvironmentObservedPolicy("alpha")
	custom.DeploymentBranchPolicy = &settings.EnvironmentBranchPolicy{CustomBranchPolicies: true}
	custom.Reviewers = environmentUIPtr([]settings.EnvironmentReviewer{{Type: "User", ID: 7}, {Type: "Team", ID: 8}})
	custom.PreventSelfReview = environmentUIPtr(true)
	custom.DeploymentBranchPatterns = environmentUIPtr([]settings.EnvironmentPattern{{Type: "tag", Name: "Release/*"}, {Type: "branch", Name: "main"}})
	rawPatterns := []settings.EnvironmentPatternState{{ID: 31, Pattern: settings.EnvironmentPattern{Type: "tag", Name: "Release/*"}}, {ID: 32, Pattern: settings.EnvironmentPattern{Type: "branch", Name: "main"}}}
	acquired, err := settings.NewEnvironmentObservation([]settings.EnvironmentState{{ID: 3, Policy: plain}, {ID: 2, Policy: protected}, {ID: 1, Policy: custom, Patterns: rawPatterns}})
	if err != nil {
		t.Fatal(err)
	}
	view, err := EnvironmentsObservation(acquired)
	if err != nil || len(view) != 3 {
		t.Fatal(view, err)
	}
	if view[0].Environment.Name != "alpha" || view[0].ID != 1 || !reflect.DeepEqual(view[0].Patterns, rawPatterns) || view[1].Environment.Name != "Beta" || view[2].Environment.Name != "Zed" {
		t.Fatal(view)
	}
	for i, choice := range []string{"selected_branches_and_tags", "protected_branches_only", "no_restriction"} {
		if string(*view[i].Environment.DeploymentBranchesAndTags) != choice {
			t.Fatal(i, view[i])
		}
	}
	alpha := view[0].Environment
	if !*alpha.DeploymentProtectionRules.PreventSelfReview || *alpha.DeploymentProtectionRules.AllowAdministratorsToBypassConfiguredProtectionRules || *alpha.DeploymentProtectionRules.WaitTimer != 0 {
		t.Fatal(alpha)
	}
	reviewers := *alpha.DeploymentProtectionRules.RequiredReviewers
	if len(reviewers) != 2 || reviewers[0].Team == nil || *reviewers[0].Team != 8 || reviewers[1].Person == nil || *reviewers[1].Person != 7 {
		t.Fatal(reviewers)
	}
	if !reflect.DeepEqual(*alpha.DeploymentBranchOrTagRules, []uimodel.EnvironmentDeploymentRule{{RefType: "branch", NamePattern: "main"}, {RefType: "tag", NamePattern: "Release/*"}}) || view[1].Environment.DeploymentBranchOrTagRules != nil || view[2].Environment.DeploymentBranchOrTagRules != nil {
		t.Fatal(view)
	}
	*reviewers[0].Team = 99
	view[0].Patterns[0].ID = 99
	(*alpha.DeploymentBranchOrTagRules)[0].NamePattern = "changed"
	unchanged, err := acquired.Values()
	if err != nil || unchanged[2].Patterns[0].ID != 31 || (*unchanged[2].Policy.Reviewers)[0].ID != 8 || (*unchanged[2].Policy.DeploymentBranchPatterns)[0].Name != "main" {
		t.Fatal(unchanged, err)
	}
	emptyCustom := uiEnvironmentObservedPolicy("empty")
	emptyCustom.DeploymentBranchPolicy = &settings.EnvironmentBranchPolicy{CustomBranchPolicies: true}
	emptyCustom.DeploymentBranchPatterns = environmentUIPtr([]settings.EnvironmentPattern{})
	acquired, err = settings.NewEnvironmentObservation([]settings.EnvironmentState{{ID: 4, Policy: emptyCustom}})
	if err != nil {
		t.Fatal(err)
	}
	view, err = EnvironmentsObservation(acquired)
	if err != nil || view[0].Environment.DeploymentBranchOrTagRules == nil || len(*view[0].Environment.DeploymentBranchOrTagRules) != 0 {
		t.Fatal(view, err)
	}
}
func TestUIEnvironmentAcquisitionFailuresNoPartialCollection(t *testing.T) {
	if _, err := EnvironmentsObservation(settings.EnvironmentObservation{}); err == nil {
		t.Fatal("unacquired collection defaulted")
	}
	valid := uiEnvironmentObservedPolicy("valid")
	partial := uiEnvironmentObservedPolicy("partial")
	partial.DeploymentBranchPolicy = &settings.EnvironmentBranchPolicy{CustomBranchPolicies: true}
	raw, err := settings.NewEnvironmentObservation([]settings.EnvironmentState{{ID: 1, Policy: valid}, {ID: 2, Policy: partial}})
	if err != nil {
		t.Fatal(err)
	}
	view, err := EnvironmentsObservation(raw)
	if err == nil || len(view) != 0 {
		t.Fatal("partial custom acquisition became clearing intent", view, err)
	}
	mismatch := uiEnvironmentObservedPolicy("mismatch")
	mismatch.DeploymentBranchPolicy = &settings.EnvironmentBranchPolicy{CustomBranchPolicies: true}
	mismatch.DeploymentBranchPatterns = environmentUIPtr([]settings.EnvironmentPattern{{Type: "branch", Name: "main"}})
	raw, err = settings.NewEnvironmentObservation([]settings.EnvironmentState{{ID: 1, Policy: valid}, {ID: 2, Policy: mismatch}})
	if err != nil {
		t.Fatal(err)
	}
	view, err = EnvironmentsObservation(raw)
	if err == nil || len(view) != 0 {
		t.Fatal("missing pattern identity metadata accepted", view, err)
	}

}
