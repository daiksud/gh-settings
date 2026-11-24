package github

import (
	"reflect"
	"testing"

	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

func TestUIPagesAcquiredProjection(t *testing.T) {
	for _, test := range []struct {
		api, ui, apiPath, uiFolder string
		domain                     *string
		https, public              bool
	}{
		{"legacy", "deploy_from_a_branch", "/", "root", nil, false, false},
		{"legacy", "deploy_from_a_branch", "/docs", "docs", pagesUIPtr("Docs.Example.com"), true, true},
		{"workflow", "github_actions", "ignored", "", nil, false, true},
	} {
		raw := settings.PagesState{BuildType: test.api, Source: &settings.PagesSource{Branch: "docs/source", Path: test.apiPath}, CNAME: test.domain, HTTPSEnforced: test.https, Public: test.public}
		acquired, err := settings.NewPagesObservation(raw)
		if err != nil {
			t.Fatal(err)
		}
		policy, err := acquired.Declaration()
		if err != nil {
			t.Fatal(err)
		}
		export := settings.Declaration{Pages: policy}
		if err := CheckExportAcquisition(export); err != nil {
			t.Fatal(err)
		}
		result, err := UIFromExportDeclaration(export)
		if err != nil || result.Settings.Pages == nil {
			t.Fatal(result, err)
		}
		view := result.Settings.Pages
		if *view.BuildAndDeployment.Source != uimodel.PagesSourceChoice(test.ui) || *view.EnforceHTTPS != test.https || view.CustomDomain == nil {
			t.Fatal(view, err)
		}
		if test.api == "workflow" {
			if view.BuildAndDeployment.Branch != nil || view.BuildAndDeployment.SelectFolder != nil {
				t.Fatal("inactive source exposed", view)
			}
		} else if *view.BuildAndDeployment.Branch != "docs/source" || *view.BuildAndDeployment.SelectFolder != uimodel.PagesFolderChoice(test.uiFolder) {
			t.Fatal(view)
		}
		if test.domain == nil {
			if view.CustomDomain.Value != nil {
				t.Fatal("known null replaced")
			}
		} else if *view.CustomDomain.Value != "docs.example.com" {
			t.Fatal(view)
		}
		d, err := PagesDeclaration(*view)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(d, policy) {
			t.Fatal(d, policy)
		}
		if view.BuildAndDeployment.Branch != nil {
			*view.BuildAndDeployment.Branch = "changed"
		}
		if view.CustomDomain.Value != nil {
			*view.CustomDomain.Value = "changed.example.com"
		}
		*view.EnforceHTTPS = !test.https
		if *policy.HTTPSEnforced != test.https || policy.Source != nil && policy.Source.Branch != "docs/source" || policy.CNAME.Value != nil && *policy.CNAME.Value != "docs.example.com" {
			t.Fatal("export mutation changed its input declaration", policy)
		}
		unchanged, err := acquired.Values()
		if err != nil || unchanged.Public != test.public || unchanged.Source != nil && unchanged.Source.Branch != "docs/source" || unchanged.CNAME != nil && *unchanged.CNAME != "docs.example.com" {
			t.Fatal(unchanged, err)
		}
	}
	if _, err := (settings.PagesObservation{}).Declaration(); err == nil {
		t.Fatal("missing acquisition defaulted")
	}
}

func TestPagesExportRejectsIncompleteAcquisition(t *testing.T) {
	for _, policy := range []settings.PagesDeclaration{
		{CNAME: &settings.PagesDomain{}, HTTPSEnforced: rulesetPtr(false)},
		{BuildType: rulesetPtr("workflow"), HTTPSEnforced: rulesetPtr(false)},
		{BuildType: rulesetPtr("workflow"), CNAME: &settings.PagesDomain{}},
		{BuildType: rulesetPtr("legacy"), CNAME: &settings.PagesDomain{}, HTTPSEnforced: rulesetPtr(false)},
		{BuildType: rulesetPtr("legacy"), Source: &settings.PagesSource{Branch: "main", Path: "/other"}, CNAME: &settings.PagesDomain{}, HTTPSEnforced: rulesetPtr(false)},
	} {
		if err := CheckExportAcquisition(settings.Declaration{Pages: policy}); err == nil {
			t.Fatalf("accepted incomplete Pages configuration: %#v", policy)
		}
	}
}
