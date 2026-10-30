package settings

import (
	"testing"

	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

func TestRulesetEnforcementMappingSeparatesUIAndNativeChoices(t *testing.T) {
	if RulesetEnforcementPath != "repository.settings.rulesets.enforcement_status" || DefaultRulesetEnforcementChoice != uimodel.Disabled {
		t.Fatal("wrong field/default")
	}
	for _, test := range []struct {
		choice uimodel.EnforcementStatus
		native string
	}{{uimodel.Active, "active"}, {uimodel.Disabled, "disabled"}} {
		value, ok := RulesetEnforcementValueForChoice(test.choice)
		if !ok || value != test.native {
			t.Fatal(value, ok)
		}
		choice, ok := RulesetEnforcementChoiceForValue(test.native)
		if !ok || choice != test.choice {
			t.Fatal(choice, ok)
		}
	}
	for _, raw := range []string{"evaluate", "FUTURE_MODE", "", " \t", "ACTIVE"} {
		if choice, ok := RulesetEnforcementChoiceForValue(raw); ok || choice != "" {
			t.Fatal("unsupported UI value", raw, choice)
		}
		if value, ok := RulesetEnforcementValueForChoice(uimodel.EnforcementStatus(raw)); ok || value != "" {
			t.Fatal("unsupported desired choice", raw, value)
		}
	}
	policy := basicPolicy("native")
	policy.Enforcement = rsText("evaluate")
	if err := ValidateRuleset(policy, true); err != nil {
		t.Fatal("native evaluate contract changed", err)
	}
}
