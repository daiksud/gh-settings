package workspace

import (
	"github.com/daiksud/gh-settings/internal/github"
	"reflect"
	"testing"
)

func TestPullRequestExemptionsYAML(t *testing.T) {
	prefix := "version: daiksud/gh-settings@v0\nrepository: {settings: {interaction_limits: {pull_request_limits: "
	for _, body := range []string{"{}", "{exemption_list: []}", "{exemption_list: [Alice, z-user]}"} {
		ui, err := DecodeConfig([]byte(prefix + body + "}}}\n"))
		if err != nil {
			t.Fatal(err)
		}
		d, err := github.DeclarationFromUI(ui)
		if err != nil {
			t.Fatal(err)
		}
		if !d.PullRequestLimits.Empty() || (d.PullRequestExemptions == nil) != (body == "{}") || d.Empty() != (body == "{}") {
			t.Fatalf("ownership %#v", d)
		}
		encoded, err := EncodeConfig(ui)
		if err != nil {
			t.Fatal(err)
		}
		again, err := DecodeConfig(encoded)
		if err != nil || !reflect.DeepEqual(ui, again) {
			t.Fatalf("roundtrip %s %v", encoded, err)
		}
	}
	for _, value := range []string{"null", "{}", "false", "[null]", "[12]", "[false]", "[{login: alice}]", "['']", "[alice, ALICE]", "['app[bot]']"} {
		ui, err := DecodeConfig([]byte(prefix + "{exemption_list: " + value + "}}}}\n"))
		if err == nil {
			_, err = github.DeclarationFromUI(ui)
		}
		if err == nil {
			t.Fatal("accepted", value)
		}
	}
}
