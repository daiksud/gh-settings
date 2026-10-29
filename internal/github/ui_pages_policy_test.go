package github

import (
	"reflect"
	"testing"

	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

func TestPagesPolicyUI(t *testing.T) {
	for _, domain := range []*string{nil, rulesetPtr("docs.example.com")} {
		policy := settings.PagesDeclaration{BuildType: rulesetPtr("legacy"), Source: &settings.PagesSource{Branch: "main", Path: "/docs"}, CNAME: &settings.PagesDomain{Value: domain}, HTTPSEnforced: rulesetPtr(false)}
		want := uimodel.Pages{BuildAndDeployment: &uimodel.PagesBuildAndDeployment{Source: rulesetPtr(uimodel.DeployFromABranch), Branch: rulesetPtr("main"), SelectFolder: rulesetPtr(uimodel.PagesDocs)}, CustomDomain: &uimodel.PagesCustomDomain{Value: domain}, EnforceHTTPS: rulesetPtr(false)}
		got, err := PagesPolicyUI(policy)
		if err != nil || !reflect.DeepEqual(got, want) {
			t.Fatalf("Pages projection %#v; want %#v; error=%v", got, want, err)
		}
		*got.BuildAndDeployment.Branch = "changed"
		*got.EnforceHTTPS = true
		if policy.Source.Branch != "main" || *policy.HTTPSEnforced {
			t.Fatal("Pages projection aliases caller")
		}
	}
	got, err := PagesPolicyUI(settings.PagesDeclaration{BuildType: rulesetPtr("workflow"), CNAME: &settings.PagesDomain{}, HTTPSEnforced: rulesetPtr(true)})
	want := uimodel.Pages{BuildAndDeployment: &uimodel.PagesBuildAndDeployment{Source: rulesetPtr(uimodel.GitHubActions)}, CustomDomain: &uimodel.PagesCustomDomain{}, EnforceHTTPS: rulesetPtr(true)}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("workflow projection %#v, %v", got, err)
	}
}

func TestPagesPolicyUIRejectsIncompletePolicies(t *testing.T) {
	for _, policy := range []settings.PagesDeclaration{
		{CNAME: &settings.PagesDomain{}, HTTPSEnforced: rulesetPtr(false)},
		{BuildType: rulesetPtr("workflow"), HTTPSEnforced: rulesetPtr(false)},
		{BuildType: rulesetPtr("workflow"), CNAME: &settings.PagesDomain{}},
		{BuildType: rulesetPtr("legacy"), CNAME: &settings.PagesDomain{}, HTTPSEnforced: rulesetPtr(false)},
		{BuildType: rulesetPtr("legacy"), Source: &settings.PagesSource{Branch: "main", Path: "/other"}, CNAME: &settings.PagesDomain{}, HTTPSEnforced: rulesetPtr(false)},
	} {
		got, err := PagesPolicyUI(policy)
		if err == nil || !reflect.DeepEqual(got, uimodel.Pages{}) {
			t.Fatalf("incomplete Pages projected: %#v, %v", got, err)
		}
	}
}
