package workspace

import (
	"reflect"
	"testing"

	"github.com/daiksud/gh-settings/internal/github"
)

func TestUIRepositoryVariablesRoundTrip(t *testing.T) {
	input := []byte(`version: daiksud/gh-settings@v0
repository:
  settings:
    secrets_and_variables:
      actions:
        variables:
          - name: display_name
            value: '日本語: hello'
          - name: SECOND
            value: present
`)
	got, err := DecodeConfig(input)
	if err != nil {
		t.Fatal(err)
	}
	desired, err := github.DeclarationFromUI(got)
	if err != nil {
		t.Fatal(err)
	}
	values := desired.Variables.Values()
	if len(values) != 2 || values[0].Name != "DISPLAY_NAME" || values[0].Value != "日本語: hello" || values[1].Value != "present" {
		t.Fatalf("lost variables: %#v", values)
	}
	encoded, err := EncodeConfig(got)
	if err != nil {
		t.Fatal(err)
	}
	again, err := DecodeConfig(encoded)
	if err != nil || !reflect.DeepEqual(got, again) {
		t.Fatalf("round trip: %s %v", encoded, err)
	}
}

func TestUIRepositoryVariablesPresenceAndValidation(t *testing.T) {
	for _, tc := range []struct {
		body        string
		listPresent bool
	}{
		{"{}", false},
		{"{secrets_and_variables: {actions: {}}}", false},
		{"{secrets_and_variables: {actions: {variables: []}}}", true},
	} {
		got, err := DecodeConfig([]byte("version: daiksud/gh-settings@v0\nrepository: {settings: " + tc.body + "}\n"))
		if err != nil {
			t.Fatal(err)
		}
		desired, err := github.DeclarationFromUI(got)
		if err != nil || !desired.Empty() || (desired.Variables != nil) != tc.listPresent {
			t.Fatalf("%s: unexpected empty selection: %#v %v", tc.body, desired, err)
		}
	}
	for _, value := range []string{"[{name: A, value: ''}]", "null", "{}", "[null]", "[{name: A}]", "[{name: A, value: null}]", "[{name: A, value: 12}]", "[{name: A, value: false}]", "[{name: A, value: x, future: y}]", "[{name: '', value: x}]", "[{name: '2A', value: x}]", "[{name: GITHUB_X, value: x}]", "[{name: 'A-B', value: x}]", "[{name: 日本語, value: x}]", "[{name: a, value: x}, {name: A, value: y}]"} {
		t.Run(value, func(t *testing.T) {
			got, err := DecodeConfig([]byte("version: daiksud/gh-settings@v0\nrepository: {settings: {secrets_and_variables: {actions: {variables: " + value + "}}}}\n"))
			if err == nil {
				_, err = github.DeclarationFromUI(got)
			}
			if err == nil {
				t.Fatal("accepted invalid variables")
			}
		})
	}
}

func TestRepositoryVariablesRejectOtherSettingsPaths(t *testing.T) {
	for _, body := range []string{
		"actions: {variables: []}",
		"agents: {variables: []}",
		"agents: {variables: []}, secrets_and_variables: {agents: {variables: []}}",
		"secrets_and_variables: {actions: {environment_variables: []}}",
	} {
		if _, err := DecodeConfig([]byte("version: daiksud/gh-settings@v0\nrepository: {settings: {" + body + "}}\n")); err == nil {
			t.Errorf("accepted unsupported variable location: %s", body)
		}
	}
}
