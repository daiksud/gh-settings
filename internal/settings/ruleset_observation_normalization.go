package settings

import "fmt"

func (state RulesetState) validateNormalizationEvidence() error {
	if err := ValidateRuleset(state.Policy, true); err != nil {
		return err
	}
	if state.ID <= 0 {
		return fmt.Errorf("Ruleset observation requires a positive ID")
	}
	if (state.EnforcementNormalization != (ObservationNormalization{}) || state.QualitySeverityNormalization != (ObservationNormalization{})) && state.Policy.ID != nil && *state.Policy.ID != state.ID {
		return fmt.Errorf("Ruleset observation provenance conflicts with policy ID")
	}
	if record := state.EnforcementNormalization; record != (ObservationNormalization{}) {
		fallback, _ := RulesetEnforcementValueForChoice(DefaultRulesetEnforcementChoice)
		_, supported := RulesetEnforcementChoiceForValue(record.RawValue)
		if *state.Policy.Enforcement != fallback || record.Path != RulesetEnforcementPath || record.Fallback != string(DefaultRulesetEnforcementChoice) || supported {
			return fmt.Errorf("invalid Ruleset enforcement observation provenance")
		}
	}
	if record := state.QualitySeverityNormalization; record != (ObservationNormalization{}) {
		severity := rulesetQualitySeverity(state.Policy)
		fallback, _ := RulesetQualitySeverityValueForChoice(DefaultRulesetQualitySeverityChoice)
		_, supported := RulesetQualitySeverityChoiceForValue(record.RawValue)
		if severity == nil || *severity != fallback || record.Path != RulesetQualitySeverityPath || record.Fallback != string(DefaultRulesetQualitySeverityChoice) || supported {
			return fmt.Errorf("invalid Ruleset quality observation provenance")
		}
	}
	return nil
}

func rulesetQualitySeverity(policy RulesetDeclaration) *string {
	if policy.Rules != nil {
		for _, rule := range *policy.Rules {
			if rule.Type == "code_quality" && rule.Parameters != nil {
				return rule.Parameters.Severity
			}
		}
	}
	return nil
}
