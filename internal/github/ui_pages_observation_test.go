package github

import (
	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/settings/uimodel"
	"reflect"
	"testing"
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
		view, err := PagesObservation(acquired)
		if err != nil || view.Public != test.public || *view.Pages.BuildAndDeployment.Source != uimodel.PagesSourceChoice(test.ui) || *view.Pages.EnforceHTTPS != test.https || view.Pages.CustomDomain == nil {
			t.Fatal(view, err)
		}
		if test.api == "workflow" {
			if view.Pages.BuildAndDeployment.Branch != nil || view.Pages.BuildAndDeployment.SelectFolder != nil {
				t.Fatal("inactive source exposed", view)
			}
		} else if *view.Pages.BuildAndDeployment.Branch != "docs/source" || *view.Pages.BuildAndDeployment.SelectFolder != uimodel.PagesFolderChoice(test.uiFolder) {
			t.Fatal(view)
		}
		if test.domain == nil {
			if view.Pages.CustomDomain.Value != nil {
				t.Fatal("known null replaced")
			}
		} else if *view.Pages.CustomDomain.Value != "docs.example.com" {
			t.Fatal(view)
		}
		d, err := PagesDeclaration(view.Pages)
		if err != nil {
			t.Fatal(err)
		}
		expected, err := acquired.Declaration()
		if err != nil || !reflect.DeepEqual(d, expected) {
			t.Fatal(d, expected, err)
		}
		if view.Pages.BuildAndDeployment.Branch != nil {
			*view.Pages.BuildAndDeployment.Branch = "changed"
		}
		if view.Pages.CustomDomain.Value != nil {
			*view.Pages.CustomDomain.Value = "changed.example.com"
		}
		unchanged, err := acquired.Values()
		if err != nil || unchanged.Source != nil && unchanged.Source.Branch != "docs/source" || unchanged.CNAME != nil && *unchanged.CNAME != "docs.example.com" {
			t.Fatal(unchanged, err)
		}
	}
	if _, err := PagesObservation(settings.PagesObservation{}); err == nil {
		t.Fatal("missing acquisition defaulted")
	}
}
