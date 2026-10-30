package settings

import "fmt"

// RawEnforcement returns acquired policy only after validating complete evidence.
func (state RulesetState) RawEnforcement() (string, error) {
	if err := ValidateRuleset(state.Policy, true); err != nil {
		return "", err
	}
	if state.ID <= 0 {
		return "", fmt.Errorf("Ruleset observation requires a positive ID")
	}
	record := state.EnforcementNormalization
	if record == (ObservationNormalization{}) {
		return *state.Policy.Enforcement, nil
	}
	fallback, _ := RulesetEnforcementValueForChoice(DefaultRulesetEnforcementChoice)
	_, supported := RulesetEnforcementChoiceForValue(record.RawValue)
	if *state.Policy.Enforcement != fallback || record.Path != RulesetEnforcementPath || record.Fallback != string(DefaultRulesetEnforcementChoice) || supported || state.Policy.ID != nil && *state.Policy.ID != state.ID {
		return "", fmt.Errorf("invalid Ruleset enforcement observation provenance")
	}
	return record.RawValue, nil
}
