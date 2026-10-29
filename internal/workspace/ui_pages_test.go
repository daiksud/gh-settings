package workspace

import (
	"reflect"
	"testing"

	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

func TestUIConfigPagesControls(t *testing.T) {
	input := `version: daiksud/gh-settings@v0
repository:
  settings:
    pages:
      build_and_deployment:
        source: deploy_from_a_branch
        branch: main
        select_folder: root
      custom_domain: docs.example.com
      enforce_https: false
`
	want := &uimodel.Pages{BuildAndDeployment: &uimodel.PagesBuildAndDeployment{Source: uiConfigPtr(uimodel.DeployFromABranch), Branch: uiConfigPtr("main"), SelectFolder: uiConfigPtr(uimodel.PagesRoot)}, CustomDomain: &uimodel.PagesCustomDomain{Value: uiConfigPtr("docs.example.com")}, EnforceHTTPS: uiConfigPtr(false)}
	got, err := DecodeConfig([]byte(input))
	if err != nil || !reflect.DeepEqual(got.Settings.Pages, want) {
		t.Fatalf("Pages controls changed: %#v, %v", got, err)
	}
	encoded, err := EncodeConfig(got)
	if err != nil {
		t.Fatal(err)
	}
	again, err := DecodeConfig(encoded)
	if err != nil || !reflect.DeepEqual(again, got) {
		t.Fatalf("Pages round trip: %s, %v", encoded, err)
	}
}

func TestUIConfigPagesDomainPresence(t *testing.T) {
	for _, test := range []struct {
		input, encoded string
		want           *uimodel.PagesCustomDomain
	}{
		{"{}", "version: daiksud/gh-settings@v0\nrepository:\n    settings:\n        pages: {}\n", nil},
		{"{custom_domain: null}", "version: daiksud/gh-settings@v0\nrepository:\n    settings:\n        pages:\n            custom_domain: null\n", &uimodel.PagesCustomDomain{}},
		{"{custom_domain: ''}", "version: daiksud/gh-settings@v0\nrepository:\n    settings:\n        pages:\n            custom_domain: \"\"\n", &uimodel.PagesCustomDomain{Value: uiConfigPtr("")}},
	} {
		got, err := DecodeConfig([]byte("version: daiksud/gh-settings@v0\nrepository: {settings: {pages: " + test.input + "}}\n"))
		if err != nil || !reflect.DeepEqual(got.Settings.Pages.CustomDomain, test.want) {
			t.Fatalf("domain presence %s: %#v, %v", test.input, got, err)
		}
		encoded, err := EncodeConfig(got)
		if err != nil || string(encoded) != test.encoded {
			t.Fatalf("domain golden: %s, %v", encoded, err)
		}
	}
}

func TestUIConfigPagesRejectsInvalidSyntax(t *testing.T) {
	for _, controls := range []string{
		"build_type: workflow", "cname: example.com", "public: true", "https_enforced: true",
		"custom_domain: {value: example.com}", "custom_domain: false", "enforce_https: null",
		"build_and_deployment: {source: null}", "build_and_deployment: {select_folder: false}",
	} {
		if _, err := DecodeConfig([]byte("version: daiksud/gh-settings@v0\nrepository: {settings: {pages: {" + controls + "}}}\n")); err == nil {
			t.Fatalf("accepted invalid Pages syntax: %s", controls)
		}
	}
}
