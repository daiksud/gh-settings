package settings

import (
	"testing"

	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

func TestRulesetQualitySeverityMappingKeepsUIAndNativeSpelling(t *testing.T) {
	if RulesetQualitySeverityPath != "repository.settings.rulesets.rules.require_code_quality_results.severity" || DefaultRulesetQualitySeverityChoice != uimodel.CodeQualitySeverity("errors") {
		t.Fatal("wrong quality path/default")
	}
	for _, test := range []struct {
		choice uimodel.CodeQualitySeverity
		native string
	}{{"errors", "errors"}, {"warnings_and_higher", "warnings"}, {"notes_and_higher", "notes"}, {"all", "all"}} {
		value, ok := RulesetQualitySeverityValueForChoice(test.choice)
		if !ok || value != test.native {
			t.Fatal(value, ok)
		}
		choice, ok := RulesetQualitySeverityChoiceForValue(test.native)
		if !ok || choice != test.choice {
			t.Fatal(choice, ok)
		}
		policy := basicPolicy("quality")
		policy.Rules = &[]RulesetRule{{Type: "code_quality", Parameters: &RuleParameters{Severity: rsText(test.native)}}}
		if err := ValidateRuleset(policy, true); err != nil {
			t.Fatal("native severity rejected", err)
		}
	}
	for _, choice := range []string{"warnings", "notes", "future", "", " \t", "Errors"} {
		if value, ok := RulesetQualitySeverityValueForChoice(uimodel.CodeQualitySeverity(choice)); ok || value != "" {
			t.Fatal("invalid desired UI choice", choice, value)
		}
	}
	for _, raw := range []string{"warnings_and_higher", "notes_and_higher", "future", "", " \t", "Errors"} {
		if choice, ok := RulesetQualitySeverityChoiceForValue(raw); ok || choice != "" {
			t.Fatal("invalid native value", raw, choice)
		}
	}
}
