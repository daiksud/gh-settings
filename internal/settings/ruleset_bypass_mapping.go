package settings

import "github.com/daiksud/gh-settings/internal/settings/uimodel"

const RulesetBypassModePath = "repository.settings.rulesets.bypass_list.bypass_mode"
const DefaultRulesetBypassModeChoice = uimodel.DefaultBypassMode

var rulesetBypassModeChoices = [...]struct {
	choice uimodel.BypassMode
	value  string
}{
	{uimodel.Always, "always"},
	{uimodel.Exempt, "exempt"},
	{uimodel.ForPullRequestsOnly, "pull_request"},
}

func RulesetBypassModeValueForChoice(choice uimodel.BypassMode) (string, bool) {
	for _, option := range rulesetBypassModeChoices {
		if option.choice == choice {
			return option.value, true
		}
	}
	return "", false
}

func RulesetBypassModeChoiceForValue(value string) (uimodel.BypassMode, bool) {
	for _, option := range rulesetBypassModeChoices {
		if option.value == value {
			return option.choice, true
		}
	}
	return "", false
}
