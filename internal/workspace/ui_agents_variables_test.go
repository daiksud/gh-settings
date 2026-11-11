package workspace

import (
	"reflect"
	"testing"

	"github.com/daiksud/gh-settings/internal/github"
	"github.com/daiksud/gh-settings/internal/settings"
	"go.yaml.in/yaml/v4"
)

func TestUIAgentsVariablesRoundTrip(t *testing.T) {
	input := []byte("version: daiksud/gh-settings@v0\nrepository:\n  settings:\n    actions:\n      variables:\n        - name: SHARED\n          value: actions\n    agents:\n      variables:\n        - name: shared\n          value: agents\n")
	got, err := DecodeConfig(input)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := EncodeConfig(got)
	if err != nil {
		t.Fatal(err)
	}
	var wantDocument, gotDocument any
	if err := yaml.Unmarshal(input, &wantDocument); err != nil {
		t.Fatal(err)
	}
	if err := yaml.Unmarshal(encoded, &gotDocument); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(wantDocument, gotDocument) {
		t.Fatalf("family values changed: %s", encoded)
	}
}

func TestUIAgentsVariablesDeclaration(t *testing.T) {
	for _, actions := range []string{"", "actions: {variables: [{name: SHARED, value: actions}]}, "} {
		model, err := DecodeConfig([]byte("version: daiksud/gh-settings@v0\nrepository: {settings: {" + actions + "agents: {variables: [{name: shared, value: agents}]}}}\n"))
		if err != nil {
			t.Fatal(err)
		}
		desired, err := github.DeclarationFromUI(model)
		if err != nil {
			t.Fatal(err)
		}
		want := []settings.RepositoryVariable{{Name: "SHARED", Value: "agents"}}
		if desired.Empty() || desired.AgentsVariables == nil || !reflect.DeepEqual(desired.AgentsVariables.Values(), want) {
			t.Fatalf("lost Agents intent: %#v", desired)
		}
		if actions != "" && (desired.Variables == nil || !reflect.DeepEqual(desired.Variables.Values(), []settings.RepositoryVariable{{Name: "SHARED", Value: "actions"}})) {
			t.Fatal("Actions intent changed")
		}
	}
}

func TestUIAgentsVariablesPresenceAndValidation(t *testing.T) {
	for _, body := range []string{"{}", "{agents: {}}", "{agents: {variables: []}}"} {
		got, err := DecodeConfig([]byte("version: daiksud/gh-settings@v0\nrepository: {settings: " + body + "}\n"))
		if err != nil {
			t.Fatal(err)
		}
		desired, err := github.DeclarationFromUI(got)
		if err != nil || !desired.Empty() {
			t.Fatalf("empty selection managed: %#v %v", desired, err)
		}
	}
	for _, body := range []string{
		"null", "[]", "{future: true}",
		"{variables: null}", "{variables: {}}", "{variables: [null]}",
		"{variables: [{name: A}]}", "{variables: [{name: A, value: null}]}",
		"{variables: [{name: A, value: ''}]}", "{variables: [{name: A, value: 12}]}",
		"{variables: [{name: A, value: false}]}", "{variables: [{name: A, value: x, future: y}]}",
		"{variables: [{name: '', value: x}]}", "{variables: [{name: '2A', value: x}]}",
		"{variables: [{name: GITHUB_X, value: x}]}", "{variables: [{name: 'A-B', value: x}]}",
		"{variables: [{name: 日本語, value: x}]}", "{variables: [{name: a, value: x}, {name: A, value: y}]}",
	} {
		t.Run(body, func(t *testing.T) {
			got, err := DecodeConfig([]byte("version: daiksud/gh-settings@v0\nrepository: {settings: {agents: " + body + "}}\n"))
			if err == nil {
				_, err = github.DeclarationFromUI(got)
			}
			if err == nil {
				t.Fatal("accepted invalid Agents variables")
			}
		})
	}
}
