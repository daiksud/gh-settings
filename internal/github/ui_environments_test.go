package github

import (
	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/settings/uimodel"
	"reflect"
	"testing"
)

func environmentUIPtr[T any](value T) *T { return &value }
func TestUIEnvironmentBuiltInSingletons(t *testing.T) {
	for _, test := range []struct {
		ui   uimodel.EnvironmentDeploymentProtectionRules
		want settings.EnvironmentDeclaration
	}{
		{uimodel.EnvironmentDeploymentProtectionRules{WaitTimer: environmentUIPtr(0)}, settings.EnvironmentDeclaration{Name: "Production", WaitTimer: environmentUIPtr(0)}},
		{uimodel.EnvironmentDeploymentProtectionRules{WaitTimer: environmentUIPtr(43200)}, settings.EnvironmentDeclaration{Name: "Production", WaitTimer: environmentUIPtr(43200)}},
		{uimodel.EnvironmentDeploymentProtectionRules{PreventSelfReview: environmentUIPtr(false)}, settings.EnvironmentDeclaration{Name: "Production", PreventSelfReview: environmentUIPtr(false)}},
		{uimodel.EnvironmentDeploymentProtectionRules{PreventSelfReview: environmentUIPtr(true)}, settings.EnvironmentDeclaration{Name: "Production", PreventSelfReview: environmentUIPtr(true)}},
		{uimodel.EnvironmentDeploymentProtectionRules{AllowAdministratorsToBypassConfiguredProtectionRules: environmentUIPtr(false)}, settings.EnvironmentDeclaration{Name: "Production", CanAdminsBypass: environmentUIPtr(false)}},
		{uimodel.EnvironmentDeploymentProtectionRules{AllowAdministratorsToBypassConfiguredProtectionRules: environmentUIPtr(true)}, settings.EnvironmentDeclaration{Name: "Production", CanAdminsBypass: environmentUIPtr(true)}},
		{uimodel.EnvironmentDeploymentProtectionRules{RequiredReviewers: environmentUIPtr([]uimodel.EnvironmentReviewer{})}, settings.EnvironmentDeclaration{Name: "Production", Reviewers: environmentUIPtr([]settings.EnvironmentReviewer{})}},
	} {
		got, err := EnvironmentDeclaration(uimodel.Environment{Name: "Production", DeploymentProtectionRules: &test.ui})
		if err != nil || !reflect.DeepEqual(got, test.want) {
			t.Fatal(got, test.want, err)
		}
	}
	got, err := EnvironmentDeclaration(uimodel.Environment{Name: "Production"})
	if err != nil || !reflect.DeepEqual(got, settings.EnvironmentDeclaration{Name: "Production"}) {
		t.Fatal(got, err)
	}
	for _, name := range []string{"", " bad", "bad\nname"} {
		if _, err := EnvironmentDeclaration(uimodel.Environment{Name: name}); err == nil {
			t.Fatal("invalid identity accepted")
		}
	}
	for _, value := range []int{-1, 43201} {
		if _, err := EnvironmentDeclaration(uimodel.Environment{Name: "Production", DeploymentProtectionRules: &uimodel.EnvironmentDeploymentProtectionRules{WaitTimer: &value}}); err == nil {
			t.Fatal("invalid timer accepted")
		}
	}
}
func TestUIEnvironmentTypedReviewersAndOwnership(t *testing.T) {
	reviewers := []uimodel.EnvironmentReviewer{{Person: environmentUIPtr(int64(7))}, {Team: environmentUIPtr(int64(8))}}
	timer := 15
	got, err := EnvironmentDeclaration(uimodel.Environment{Name: "Production", DeploymentProtectionRules: &uimodel.EnvironmentDeploymentProtectionRules{RequiredReviewers: &reviewers, WaitTimer: &timer}})
	if err != nil || !reflect.DeepEqual(*got.Reviewers, []settings.EnvironmentReviewer{{Type: "Team", ID: 8}, {Type: "User", ID: 7}}) {
		t.Fatal(got, err)
	}
	*reviewers[0].Person = 9
	timer = 30
	if *got.WaitTimer != 15 || (*got.Reviewers)[1].ID != 7 {
		t.Fatal("declaration aliases caller")
	}
	for _, entry := range []uimodel.EnvironmentReviewer{{}, {Person: environmentUIPtr(int64(0))}, {Team: environmentUIPtr(int64(-1))}, {Person: environmentUIPtr(int64(7)), Team: environmentUIPtr(int64(8))}} {
		if _, err := EnvironmentDeclaration(uimodel.Environment{Name: "Production", DeploymentProtectionRules: &uimodel.EnvironmentDeploymentProtectionRules{RequiredReviewers: environmentUIPtr([]uimodel.EnvironmentReviewer{entry})}}); err == nil {
			t.Fatal("invalid principal accepted")
		}
	}
	if _, err := EnvironmentDeclaration(uimodel.Environment{Name: "Production", DeploymentProtectionRules: &uimodel.EnvironmentDeploymentProtectionRules{RequiredReviewers: environmentUIPtr([]uimodel.EnvironmentReviewer{}), PreventSelfReview: environmentUIPtr(true)}}); err == nil {
		t.Fatal("empty reviewer dependency accepted")
	}
}
