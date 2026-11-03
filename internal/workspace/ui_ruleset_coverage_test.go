package workspace

import (
	"reflect"
	"strings"
	"testing"
)

func TestRulesetCoverageNumericCodec(t *testing.T) {
	base := "version: daiksud/gh-settings@v0\nrepository: {settings: {rulesets: [{ruleset_name: coverage, enforcement_status: active, target_branches: {}, rules: {restrict_code_coverage: {minimum_line_coverage_percentage: VALUE, maximum_line_coverage_drop: 0}}}]}}\n"
	for _, value := range []string{"0", "80", "99.75", "1e1"} {
		input := strings.Replace(base, "VALUE", value, 1)
		ui, err := DecodeConfig([]byte(input))
		if err != nil {
			t.Fatal(err)
		}
		encoded, err := EncodeConfig(ui)
		if err != nil {
			t.Fatal(err)
		}
		again, err := DecodeConfig(encoded)
		if err != nil || !reflect.DeepEqual(ui, again) {
			t.Fatal(err, string(encoded))
		}
	}
	for _, value := range []string{`"80"`, "true", "null", "[]", ".inf", ".nan"} {
		if _, err := DecodeConfig([]byte(strings.Replace(base, "VALUE", value, 1))); err == nil {
			t.Fatal(value)
		}
	}
}
