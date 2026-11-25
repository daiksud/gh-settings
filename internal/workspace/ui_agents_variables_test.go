package workspace

import (
	"reflect"
	"testing"

	"github.com/daiksud/gh-settings/internal/github"
	"github.com/daiksud/gh-settings/internal/settings"
	"go.yaml.in/yaml/v4"
)

func TestUIAgentsVariablesRoundTrip(t *testing.T) {
	input := []byte(`version: daiksud/gh-settings@v0
repository:
  settings:
    secrets_and_variables:
      agents:
        variables: [{name: shared, value: agents}]
      actions:
        variables: [{name: SHARED, value: actions}]
    deploy_keys: []
`)
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
	root, err := loadYAMLRoot(encoded)
	if err != nil {
		t.Fatal(err)
	}
	settings := mappingValue(mappingValue(root, "repository"), "settings")
	if settings.Content[0].Value != "deploy_keys" || settings.Content[2].Value != "secrets_and_variables" {
		t.Fatalf("unexpected settings order:\n%s", encoded)
	}
	group := mappingValue(settings, "secrets_and_variables")
	if group.Content[0].Value != "actions" || group.Content[2].Value != "agents" {
		t.Fatalf("unexpected variable pages order:\n%s", encoded)
	}
}

func TestUIAgentsVariablesDeclaration(t *testing.T) {
	for _, actions := range []string{"", "actions: {variables: [{name: SHARED, value: actions}]}, "} {
		model, err := DecodeConfig([]byte("version: daiksud/gh-settings@v0\nrepository: {settings: {secrets_and_variables: {" + actions + "agents: {variables: [{name: shared, value: agents}]}}}}\n"))
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
	for _, tc := range []struct {
		body        string
		listPresent bool
	}{
		{"{}", false},
		{"{secrets_and_variables: {agents: {}}}", false},
		{"{secrets_and_variables: {agents: {variables: []}}}", true},
	} {
		got, err := DecodeConfig([]byte("version: daiksud/gh-settings@v0\nrepository: {settings: " + tc.body + "}\n"))
		if err != nil {
			t.Fatal(err)
		}
		desired, err := github.DeclarationFromUI(got)
		if err != nil || !desired.Empty() || (desired.AgentsVariables != nil) != tc.listPresent {
			t.Fatalf("%s: unexpected empty selection: %#v %v", tc.body, desired, err)
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
			got, err := DecodeConfig([]byte("version: daiksud/gh-settings@v0\nrepository: {settings: {secrets_and_variables: {agents: " + body + "}}}\n"))
			if err == nil {
				_, err = github.DeclarationFromUI(got)
			}
			if err == nil {
				t.Fatal("accepted invalid Agents variables")
			}
		})
	}
}
