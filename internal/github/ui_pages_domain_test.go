package github

import (
	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/settings/uimodel"
	"reflect"
	"testing"
)

func TestUIPagesCustomDomainAndHTTPS(t *testing.T) {
	for _, domain := range []*string{nil, pagesUIPtr("Docs.Example.com")} {
		got, err := PagesDeclaration(uimodel.Pages{CustomDomain: &uimodel.PagesCustomDomain{Value: domain}})
		if err != nil || !reflect.DeepEqual(got, settings.PagesDeclaration{CNAME: &settings.PagesDomain{Value: domain}}) {
			t.Fatal(got, err)
		}
	}
	domain := "docs.example.com"
	got, err := PagesDeclaration(uimodel.Pages{CustomDomain: &uimodel.PagesCustomDomain{Value: &domain}})
	if err != nil {
		t.Fatal(err)
	}
	domain = "changed.example.com"
	if *got.CNAME.Value != "docs.example.com" {
		t.Fatal("desired domain aliases caller")
	}
	for _, value := range []bool{false, true} {
		got, err := PagesDeclaration(uimodel.Pages{EnforceHTTPS: &value})
		if err != nil || !reflect.DeepEqual(got, settings.PagesDeclaration{HTTPSEnforced: &value}) {
			t.Fatal(got, err)
		}
	}
	for _, invalid := range []string{"", "localhost", "https://example.com", "bad_.example.com"} {
		if _, err := PagesDeclaration(uimodel.Pages{CustomDomain: &uimodel.PagesCustomDomain{Value: &invalid}}); err == nil {
			t.Fatal("invalid domain accepted", invalid)
		}
	}
	if got, err := PagesDeclaration(uimodel.Pages{}); err != nil || got.CNAME != nil || got.HTTPSEnforced != nil {
		t.Fatal("omission defaulted", got, err)
	}
}
