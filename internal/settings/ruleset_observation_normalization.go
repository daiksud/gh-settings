package settings

import "fmt"

func (state RulesetState) validateNormalizationEvidence() error {
	if err := ValidateRuleset(state.Policy, true); err != nil {
		return err
	}
	if state.ID <= 0 {
		return fmt.Errorf("Ruleset observation requires a positive ID")
	}
	if (state.EnforcementNormalization != (ObservationNormalization{}) || state.QualitySeverityNormalization != (ObservationNormalization{}) || state.RequiredApprovalsNormalization != (RulesetRequiredApprovalsNormalization{}) || !state.MergeMethodsNormalization.IsZero() || len(state.BypassModeNormalizations) != 0 || len(state.TeamApprovalsNormalizations) != 0 || len(state.ScanningAlertsNormalizations) != 0) && state.Policy.ID != nil && *state.Policy.ID != state.ID {
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
	if record := state.RequiredApprovalsNormalization; record != (RulesetRequiredApprovalsNormalization{}) {
		count := rulesetRequiredApprovals(state.Policy)
		if count == nil || *count != DefaultRulesetRequiredApprovals || record.Path != RulesetRequiredApprovalsPath || record.Fallback != DefaultRulesetRequiredApprovals || RulesetRequiredApprovalsSupported(record.RawValue) {
			return fmt.Errorf("invalid Ruleset required approvals observation provenance")
		}
	}
	if err := state.validateMergeMethodsNormalization(); err != nil {
		return err
	}
	if err := state.validateTeamApprovalsNormalizations(); err != nil {
		return err
	}
	if err := state.validateScanningAlertsNormalizations(); err != nil {
		return err
	}
	return state.validateBypassModeNormalizations()
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
