package workspace

import (
	"strings"
	"testing"
)

func TestUIRulesetCoverageNumbersRoundTrip(t *testing.T) {
	input := "version: daiksud/gh-settings@v0\nrepository: {settings: {rulesets: [{ruleset_name: coverage, target_branches: {include_default_branch: true}, rules: {restrict_code_coverage: {minimum_line_coverage_percentage: 80.5, maximum_line_coverage_drop: 0}}}]}}\n"
	config, err := DecodeConfig([]byte(input))
	if err != nil {
		t.Fatal(err)
	}
	data, err := EncodeConfig(config)
	if err != nil || !strings.Contains(string(data), "minimum_line_coverage_percentage: 80.5") || !strings.Contains(string(data), "maximum_line_coverage_drop: 0") {
		t.Fatal(string(data), err)
	}
}

func TestUIRulesetCoverageRejectsWrongShapes(t *testing.T) {
	for _, value := range []string{
		"false", "true", "null", "[]",
		"{minimum_coverage: 80}", "{max_coverage_drop: 2}",
		"{minimum_line_coverage_percentage: false}",
		"{minimum_line_coverage_percentage: '80'}",
		"{minimum_line_coverage_percentage: null}",
		"{minimum_line_coverage_percentage: .nan}",
		"{maximum_line_coverage_drop: .inf}",
		"{maximum_line_coverage_drop: ''}",
	} {
		input := "version: daiksud/gh-settings@v0\nrepository: {settings: {rulesets: [{ruleset_name: coverage, rules: {restrict_code_coverage: " + value + "}}]}}\n"
		if _, err := DecodeConfig([]byte(input)); err == nil {
			t.Fatal("accepted invalid coverage input", value)
		}
	}
}
