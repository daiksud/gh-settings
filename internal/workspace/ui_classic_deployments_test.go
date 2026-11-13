package workspace

import (
	"github.com/daiksud/gh-settings/internal/github"
	"reflect"
	"testing"
)

func TestClassicDeploymentsLiteralCodec(t *testing.T) {
	const prefix = "version: daiksud/gh-settings@v0\nrepository: {settings: {branch_protection_rules: [{id: 7, branch_name_pattern: 'release/**', require_deployments_to_succeed_before_merging: "
	for _, value := range []string{"{}", "{enabled: false}", "{enabled: true, environments: []}", "{enabled: true, environments: [staging, production]}"} {
		ui, err := DecodeConfig([]byte(prefix + value + "}]}}\n"))
		if err != nil {
			t.Fatal(value, err)
		}
		if _, err := github.DeclarationFromUI(ui); err != nil {
			t.Fatal(value, err)
		}
		data, err := EncodeConfig(ui)
		if err != nil {
			t.Fatal(err)
		}
		again, err := DecodeConfig(data)
		if err != nil || !reflect.DeepEqual(ui, again) {
			t.Fatal(string(data), err)
		}
	}
}

func TestClassicDeploymentInvalidLiteral(t *testing.T) {
	const prefix = "version: daiksud/gh-settings@v0\nrepository: {settings: {branch_protection_rules: [{branch_name_pattern: 'release/**', require_deployments_to_succeed_before_merging: "
	for _, value := range []string{"null", "true", "[]", "{enabled: null}", "{enabled: 'false'}", "{enabled: true}", "{environments: []}", "{enabled: false, environments: []}", "{enabled: true, environments: null}", "{enabled: true, environments: [null]}", "{enabled: true, environments: [production, production]}", "{enabled: true, environments: ['']}", "{requiresDeployments: true}"} {
		ui, err := DecodeConfig([]byte(prefix + value + "}]}}\n"))
		if err == nil {
			_, err = github.DeclarationFromUI(ui)
		}
		if err == nil {
			t.Fatal("accepted invalid group", value)
		}
	}
}
