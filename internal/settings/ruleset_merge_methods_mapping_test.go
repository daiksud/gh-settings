package settings

import (
	"reflect"
	"testing"

	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

func TestRulesetMergeMethodMappingAndOwnedDefault(t *testing.T) {
	if RulesetMergeMethodsPath != "repository.settings.rulesets.rules.require_a_pull_request_before_merging.allowed_merge_methods" {
		t.Fatal("wrong merge methods path")
	}
	want := []string{"merge", "squash", "rebase"}
	defaults := DefaultRulesetAllowedMergeMethods()
	if !reflect.DeepEqual(defaults, want) {
		t.Fatal("wrong observation default", defaults)
	}
	defaults[0] = "corrupted"
	if !reflect.DeepEqual(DefaultRulesetAllowedMergeMethods(), want) {
		t.Fatal("default storage escaped")
	}
	for _, method := range want {
		value, ok := RulesetMergeMethodValueForChoice(uimodel.MergeMethod(method))
		if !ok || value != method {
			t.Fatal("wrong native method", method, value, ok)
		}
		choice, ok := RulesetMergeMethodChoiceForValue(method)
		if !ok || string(choice) != method {
			t.Fatal("wrong UI method", method, choice, ok)
		}
		policy := reviewPolicy()
		ruleOf(t, policy, "pull_request").AllowedMergeMethods = rsList(method)
		if err := ValidateRuleset(policy, true); err != nil {
			t.Fatal("supported singleton rejected", method, err)
		}
	}
	for _, raw := range []string{"MERGE", "future", "", " \t", "squash_merge"} {
		if value, ok := RulesetMergeMethodValueForChoice(uimodel.MergeMethod(raw)); ok || value != "" {
			t.Fatal("unsupported UI method accepted", raw, value)
		}
		if choice, ok := RulesetMergeMethodChoiceForValue(raw); ok || choice != "" {
			t.Fatal("unsupported native method accepted", raw, choice)
		}
	}
	for _, methods := range [][]string{{}, {"merge", "merge"}, {"merge", "future"}, {"future"}} {
		policy := reviewPolicy()
		ruleOf(t, policy, "pull_request").AllowedMergeMethods = &methods
		if err := ValidateRuleset(policy, true); err == nil {
			t.Fatal("strict native validation relaxed", methods)
		}
	}
}
