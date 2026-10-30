package github

import (
	"encoding/json"

	"github.com/daiksud/gh-settings/internal/settings"
)

// decodeObservedRulesetPolicy projects acquired enforcement, approval count and Quality severity; mutation
// verification continues to use the strict decoder against raw responses.
func decodeObservedRulesetPolicy(raw map[string]json.RawMessage) (settings.RulesetDeclaration, settings.RulesetNormalizations, error) {
	var enforcement string
	if err := requiredRuleJSON(raw, "enforcement", &enforcement); err != nil {
		return settings.RulesetDeclaration{}, settings.RulesetNormalizations{}, err
	}
	copy := make(map[string]json.RawMessage, len(raw))
	for key, value := range raw {
		copy[key] = value
	}
	record := settings.ObservationNormalization{}
	if _, supported := settings.RulesetEnforcementChoiceForValue(enforcement); !supported {
		fallback, _ := settings.RulesetEnforcementValueForChoice(settings.DefaultRulesetEnforcementChoice)
		value, err := json.Marshal(fallback)
		if err != nil {
			return settings.RulesetDeclaration{}, settings.RulesetNormalizations{}, err
		}
		copy["enforcement"] = value
		record = settings.ObservationNormalization{Path: settings.RulesetEnforcementPath, RawValue: enforcement, Fallback: string(settings.DefaultRulesetEnforcementChoice)}
	}
	rules, quality, err := prepareRulesetQualityRules(raw["rules"])
	if err != nil {
		return settings.RulesetDeclaration{}, settings.RulesetNormalizations{}, err
	}
	rules, approvals, err := prepareRulesetRequiredApprovalsRules(rules)
	if err != nil {
		return settings.RulesetDeclaration{}, settings.RulesetNormalizations{}, err
	}
	copy["rules"] = rules
	policy, err := decodeRulesetPolicy(copy)
	if err != nil {
		return settings.RulesetDeclaration{}, settings.RulesetNormalizations{}, err
	}
	return policy, settings.RulesetNormalizations{Enforcement: record, QualitySeverity: quality, RequiredApprovals: approvals}, nil
}
