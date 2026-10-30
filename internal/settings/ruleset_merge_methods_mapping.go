package settings

import "github.com/daiksud/gh-settings/internal/settings/uimodel"

const RulesetMergeMethodsPath = "repository.settings.rulesets.rules.require_a_pull_request_before_merging.allowed_merge_methods"

var rulesetMergeMethodChoices = [...]struct {
	choice uimodel.MergeMethod
	value  string
}{
	{uimodel.Merge, "merge"},
	{uimodel.Squash, "squash"},
	{uimodel.Rebase, "rebase"},
}

func RulesetMergeMethodValueForChoice(choice uimodel.MergeMethod) (string, bool) {
	for _, option := range rulesetMergeMethodChoices {
		if option.choice == choice {
			return option.value, true
		}
	}
	return "", false
}

func RulesetMergeMethodChoiceForValue(value string) (uimodel.MergeMethod, bool) {
	for _, option := range rulesetMergeMethodChoices {
		if option.value == value {
			return option.choice, true
		}
	}
	return "", false
}

// DefaultRulesetAllowedMergeMethods returns an owned observation fallback.
// Desired-object declaration defaults have their separate ownership contract.
func DefaultRulesetAllowedMergeMethods() []string {
	choices := uimodel.DefaultRulesetMergeMethods()
	values := make([]string, 0, len(choices))
	for _, choice := range choices {
		value, _ := RulesetMergeMethodValueForChoice(choice)
		values = append(values, value)
	}
	return values
}
