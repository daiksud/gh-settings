package workspace

import (
	"github.com/daiksud/gh-settings/internal/github"
	"reflect"
	"testing"
)

func TestUIEnvironmentVariablesRoundTripAndIndependentOwnership(t *testing.T) {
	input := []byte(`version: daiksud/gh-settings@v0
repository:
  settings:
    environments:
      - name: Production
        variables: [{name: display_name, value: '日本語: hello'}]
      - name: Staging
        deployment_branches_and_tags: no_restriction
        variables: [{name: DISPLAY_NAME, value: staging}]
      - name: Empty
        variables: []
      - name: ExplicitCreation
`)
	ui, err := DecodeConfig(input)
	if err != nil {
		t.Fatal(err)
	}
	d, err := github.DeclarationFromUI(ui)
	if err != nil {
		t.Fatal(err)
	}
	if len(d.EnvironmentVariables) != 3 || d.EnvironmentVariables[0].Variables.Values()[0].Name != "DISPLAY_NAME" {
		t.Fatalf("variables: %#v", d.EnvironmentVariables)
	}
	policies := d.Environments.Values()
	if len(policies) != 2 || policies[0].Name != "ExplicitCreation" || policies[1].Name != "Staging" {
		t.Fatalf("variable-only selection changed policy: %#v", policies)
	}
	encoded, err := EncodeConfig(ui)
	if err != nil {
		t.Fatal(err)
	}
	again, err := DecodeConfig(encoded)
	if err != nil || !reflect.DeepEqual(ui, again) {
		t.Fatalf("roundtrip %s: %v", encoded, err)
	}
}

func TestUIEnvironmentVariablesValidation(t *testing.T) {
	for _, value := range []string{"null", "{}", "[null]", "[{name: A}]", "[{name: A, value: null}]", "[{name: A, value: 12}]", "[{name: A, value: false}]", "[{name: A, value: ''}]", "[{name: A, value: x, future: y}]", "[{name: GITHUB_A, value: x}]", "[{name: a, value: x}, {name: A, value: y}]"} {
		t.Run(value, func(t *testing.T) {
			ui, err := DecodeConfig([]byte("version: daiksud/gh-settings@v0\nrepository: {settings: {environments: [{name: Production, variables: " + value + "}]}}\n"))
			if err == nil {
				_, err = github.DeclarationFromUI(ui)
			}
			if err == nil {
				t.Fatal("accepted invalid variables")
			}
		})
	}
	for _, entries := range []string{"[{name: Production, variables: []}, {name: production}]", "[{name: '', variables: []}]"} {
		ui, err := DecodeConfig([]byte("version: daiksud/gh-settings@v0\nrepository: {settings: {environments: " + entries + "}}\n"))
		if err == nil {
			_, err = github.DeclarationFromUI(ui)
		}
		if err == nil {
			t.Fatal("accepted invalid scope")
		}
	}
	ui, err := DecodeConfig([]byte("version: daiksud/gh-settings@v0\nrepository: {settings: {environments: [{name: Production, variables: []}]}}\n"))
	if err != nil {
		t.Fatal(err)
	}
	d, err := github.DeclarationFromUI(ui)
	if err != nil || !d.Empty() {
		t.Fatalf("empty variables select work: %#v %v", d, err)
	}
}
