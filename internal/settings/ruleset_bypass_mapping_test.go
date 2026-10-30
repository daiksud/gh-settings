package settings

import (
	"testing"

	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

func TestRulesetBypassModeMappingKeepsUIAndNativeSpelling(t *testing.T) {
	if RulesetBypassModePath != "repository.settings.rulesets.bypass_list.bypass_mode" || DefaultRulesetBypassModeChoice != uimodel.BypassMode("always") {
		t.Fatal("wrong bypass mode path/default")
	}
	for _, test := range []struct {
		choice uimodel.BypassMode
		native string
	}{{"always", "always"}, {"exempt", "exempt"}, {"for_pull_requests_only", "pull_request"}} {
		value, ok := RulesetBypassModeValueForChoice(test.choice)
		if !ok || value != test.native {
			t.Fatal("wrong native bypass mode", test.choice, value, ok)
		}
		choice, ok := RulesetBypassModeChoiceForValue(test.native)
		if !ok || choice != test.choice {
			t.Fatal("wrong UI bypass mode", test.native, choice, ok)
		}
		policy := basicPolicy("bypass")
		policy.BypassActors = &[]BypassActor{{ActorType: "Team", ActorID: rsID(9), BypassMode: rsText(test.native)}}
		if err := ValidateRuleset(policy, true); err != nil {
			t.Fatal("native bypass mode rejected", err)
		}
	}
	for _, choice := range []string{"pull_request", "future", "", " \t", "Always"} {
		if value, ok := RulesetBypassModeValueForChoice(uimodel.BypassMode(choice)); ok || value != "" {
			t.Fatal("invalid desired UI bypass mode", choice, value)
		}
	}
	for _, raw := range []string{"for_pull_requests_only", "future", "", " \t", "Always"} {
		if choice, ok := RulesetBypassModeChoiceForValue(raw); ok || choice != "" {
			t.Fatal("invalid native bypass mode", raw, choice)
		}
		policy := basicPolicy("bypass")
		policy.BypassActors = &[]BypassActor{{ActorType: "Team", ActorID: rsID(9), BypassMode: rsText(raw)}}
		if err := ValidateRuleset(policy, true); err == nil {
			t.Fatal("invalid native bypass mode accepted", raw)
		}
	}
}
