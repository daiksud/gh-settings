package settings

import "github.com/daiksud/gh-settings/internal/settings/uimodel"

const RulesetQualitySeverityPath = "repository.settings.rulesets.rules.require_code_quality_results.severity"
const DefaultRulesetQualitySeverityChoice = uimodel.DefaultCodeQualitySeverity

var rulesetQualitySeverityChoices = [...]struct {
	choice uimodel.CodeQualitySeverity
	value  string
}{
	{"errors", "errors"},
	{"warnings_and_higher", "warnings"},
	{"notes_and_higher", "notes"},
	{"all", "all"},
}

func RulesetQualitySeverityValueForChoice(choice uimodel.CodeQualitySeverity) (string, bool) {
	for _, option := range rulesetQualitySeverityChoices {
		if option.choice == choice {
			return option.value, true
		}
	}
	return "", false
}

func RulesetQualitySeverityChoiceForValue(value string) (uimodel.CodeQualitySeverity, bool) {
	for _, option := range rulesetQualitySeverityChoices {
		if option.value == value {
			return option.choice, true
		}
	}
	return "", false
}
