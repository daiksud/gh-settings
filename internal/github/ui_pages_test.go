package github

import (
	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/settings/uimodel"
	"reflect"
	"testing"
)

func pagesUIPtr[T any](value T) *T { return &value }
func TestUIPagesSourcesAndFolders(t *testing.T) {
	if got, err := PagesDeclaration(uimodel.Pages{}); err != nil || !reflect.DeepEqual(got, settings.PagesDeclaration{}) {
		t.Fatal(got, err)
	}
	for _, test := range []struct{ ui, api string }{{"github_actions", "workflow"}, {"deploy_from_a_branch", "legacy"}} {
		got, err := PagesDeclaration(uimodel.Pages{BuildAndDeployment: &uimodel.PagesBuildAndDeployment{Source: pagesUIPtr(uimodel.PagesSourceChoice(test.ui))}})
		if err != nil || !reflect.DeepEqual(got, settings.PagesDeclaration{BuildType: &test.api}) {
			t.Fatal(got, err)
		}
	}
	for _, test := range []struct{ ui, api string }{{"root", "/"}, {"docs", "/docs"}} {
		branch := "docs/source"
		got, err := PagesDeclaration(uimodel.Pages{BuildAndDeployment: &uimodel.PagesBuildAndDeployment{Branch: &branch, SelectFolder: pagesUIPtr(uimodel.PagesFolderChoice(test.ui))}})
		if err != nil || !reflect.DeepEqual(got, settings.PagesDeclaration{Source: &settings.PagesSource{Branch: "docs/source", Path: test.api}}) {
			t.Fatal(got, err)
		}
		branch = "changed"
		if got.Source.Branch != "docs/source" {
			t.Fatal("caller aliases desired source")
		}
	}
	for _, b := range []uimodel.PagesBuildAndDeployment{
		{Branch: pagesUIPtr("main")}, {SelectFolder: pagesUIPtr(uimodel.PagesFolderChoice("root"))},
		{Source: pagesUIPtr(uimodel.PagesSourceChoice("workflow"))},
		{Branch: pagesUIPtr("main"), SelectFolder: pagesUIPtr(uimodel.PagesFolderChoice("/docs"))},
		{Source: pagesUIPtr(uimodel.PagesSourceChoice("github_actions")), Branch: pagesUIPtr("main"), SelectFolder: pagesUIPtr(uimodel.PagesFolderChoice("root"))},
	} {
		if _, err := PagesDeclaration(uimodel.Pages{BuildAndDeployment: &b}); err == nil {
			t.Fatal("invalid source accepted", b)
		}
	}
	o, err := settings.NewPagesObservation(settings.PagesState{BuildType: "workflow"})
	if err != nil {
		t.Fatal(err)
	}
	partial, err := PagesDeclaration(uimodel.Pages{BuildAndDeployment: &uimodel.PagesBuildAndDeployment{Source: pagesUIPtr(uimodel.PagesSourceChoice("deploy_from_a_branch"))}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := settings.PreparePages(o, partial); err == nil {
		t.Fatal("transition invented branch source")
	}
	full, err := PagesDeclaration(uimodel.Pages{BuildAndDeployment: &uimodel.PagesBuildAndDeployment{Source: pagesUIPtr(uimodel.PagesSourceChoice("deploy_from_a_branch")), Branch: pagesUIPtr("main"), SelectFolder: pagesUIPtr(uimodel.PagesFolderChoice("root"))}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := settings.PreparePages(o, full); err != nil {
		t.Fatal(err)
	}
}
