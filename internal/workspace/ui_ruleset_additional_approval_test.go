package workspace

import (
	"strings"
	"testing"
)

func TestUIConfigRulesetAdditionalApproval(t *testing.T) {
	for _, value := range []string{"false", "true"} {
		input := "version: daiksud/gh-settings@v0\nrepository: {settings: {rulesets: [{ruleset_name: main, rules: {require_a_pull_request_before_merging: {require_an_additional_approval_for_unattributed_copilot_pull_requests: " + value + "}}}]}}\n"
		config, err := DecodeConfig([]byte(input))
		if err != nil {
			t.Fatal(err)
		}
		encoded, err := EncodeConfig(config)
		if err != nil || !strings.Contains(string(encoded), "require_an_additional_approval_for_unattributed_copilot_pull_requests: "+value) {
			t.Fatalf("lost explicit %s: %s, %v", value, encoded, err)
		}
	}
}
