package workspace

import (
	"reflect"
	"testing"

	"github.com/daiksud/gh-settings/internal/github"
)

func TestUIRepositoryVariablesRoundTrip(t *testing.T) {
	input := []byte("version: daiksud/gh-settings@v0\nrepository:\n  settings:\n    actions:\n      variables:\n        - name: display_name\n          value: '日本語: hello'\n        - name: SECOND\n          value: present\n")
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
	for _, body := range []string{"{}", "{actions: {variables: []}}"} {
		got, err := DecodeConfig([]byte("version: daiksud/gh-settings@v0\nrepository: {settings: " + body + "}\n"))
		if err != nil {
			t.Fatal(err)
		}
		desired, err := github.DeclarationFromUI(got)
		if err != nil || !desired.Empty() {
			t.Fatalf("empty selection managed: %#v %v", desired, err)
		}
	}
	for _, value := range []string{"[{name: A, value: ''}]", "null", "{}", "[null]", "[{name: A}]", "[{name: A, value: null}]", "[{name: A, value: 12}]", "[{name: A, value: false}]", "[{name: A, value: x, future: y}]", "[{name: '', value: x}]", "[{name: '2A', value: x}]", "[{name: GITHUB_X, value: x}]", "[{name: 'A-B', value: x}]", "[{name: 日本語, value: x}]", "[{name: a, value: x}, {name: A, value: y}]"} {
		t.Run(value, func(t *testing.T) {
			got, err := DecodeConfig([]byte("version: daiksud/gh-settings@v0\nrepository: {settings: {actions: {variables: " + value + "}}}\n"))
			if err == nil {
				_, err = github.DeclarationFromUI(got)
			}
			if err == nil {
				t.Fatal("accepted invalid variables")
			}
		})
	}
}
