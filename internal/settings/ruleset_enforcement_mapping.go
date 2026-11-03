package settings

import "github.com/daiksud/gh-settings/internal/settings/uimodel"

const RulesetEnforcementPath = "repository.settings.rulesets.enforcement_status"
const DefaultRulesetEnforcementChoice = uimodel.DefaultRulesetEnforcementStatus

var rulesetEnforcementChoices = [...]struct {
	choice uimodel.EnforcementStatus
	value  string
}{
	{uimodel.Active, "active"},
	{uimodel.Disabled, "disabled"},
}

func RulesetEnforcementValueForChoice(choice uimodel.EnforcementStatus) (string, bool) {
	for _, option := range rulesetEnforcementChoices {
		if option.choice == choice {
			return option.value, true
		}
	}
	return "", false
}

func RulesetEnforcementChoiceForValue(value string) (uimodel.EnforcementStatus, bool) {
	for _, option := range rulesetEnforcementChoices {
		if option.value == value {
			return option.choice, true
		}
	}
	return "", false
}
