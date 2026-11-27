package workspace

import (
	"reflect"
	"testing"
)

func TestUIEnvironmentCustomRulesPresenceAndRoundtrip(t *testing.T) {
	for _, text := range []string{"[19, 7]", "[]"} {
		input := []byte("version: daiksud/gh-settings@v0\nrepository: {settings: {environments: [{name: prod, deployment_protection_rules: {enable_custom_rules_with_github_apps: " + text + "}}]}}\n")
		value, err := DecodeConfig(input)
		if err != nil {
			t.Fatal(err)
		}
		if (*value.Settings.Environments)[0].DeploymentProtectionRules.EnableCustomRulesWithGitHubApps == nil {
			t.Fatal("lost explicit selection")
		}
		encoded, err := EncodeConfig(value)
		if err != nil {
			t.Fatal(err)
		}
		again, err := DecodeConfig(encoded)
		if err != nil || !reflect.DeepEqual(again, value) {
			t.Fatal(string(encoded), err)
		}
	}
	for _, text := range []string{"null", "true", "[null]", "['7']", "[1.5]", "[9223372036854775808]"} {
		if _, err := DecodeConfig([]byte("version: daiksud/gh-settings@v0\nrepository: {settings: {environments: [{name: prod, deployment_protection_rules: {enable_custom_rules_with_github_apps: " + text + "}}]}}\n")); err == nil {
			t.Fatal("accepted invalid App IDs", text)
		}
	}
}
