package github

import (
	"encoding/json"
	"fmt"

	"github.com/daiksud/gh-settings/internal/settings"
)

// decodeObservedRulesetPolicy projects acquired scalar, list and bypass values; mutation
// verification continues to use the strict decoder against raw responses.
func decodeObservedRulesetPolicy(raw map[string]json.RawMessage) (settings.RulesetDeclaration, settings.RulesetNormalizations, error) {
	if err := rejectUnsupportedRulesetScope(raw); err != nil {
		return settings.RulesetDeclaration{}, settings.RulesetNormalizations{}, err
	}
	var enforcement string
	if err := requiredRuleJSON(raw, "enforcement", &enforcement); err != nil {
		return settings.RulesetDeclaration{}, settings.RulesetNormalizations{}, err
	}
	if enforcement == "evaluate" {
		return settings.RulesetDeclaration{}, settings.RulesetNormalizations{}, fmt.Errorf("unsupported ruleset enforcement %q", enforcement)
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
	rules := raw["rules"]
	rules, approvals, err := prepareRulesetRequiredApprovalsRules(rules)
	if err != nil {
		return settings.RulesetDeclaration{}, settings.RulesetNormalizations{}, err
	}
	rules, methods, err := prepareRulesetMergeMethodsRules(rules)
	if err != nil {
		return settings.RulesetDeclaration{}, settings.RulesetNormalizations{}, err
	}
	rules, scanning, err := prepareRulesetScanningAlertsRules(rules)
	if err != nil {
		return settings.RulesetDeclaration{}, settings.RulesetNormalizations{}, err
	}
	rules, scanningSecurity, err := prepareRulesetScanningSecurityAlertsRules(rules)
	if err != nil {
		return settings.RulesetDeclaration{}, settings.RulesetNormalizations{}, err
	}
	copy["rules"] = rules
	actors, bypass, err := prepareRulesetBypassActors(raw["bypass_actors"])
	if err != nil {
		return settings.RulesetDeclaration{}, settings.RulesetNormalizations{}, err
	}
	copy["bypass_actors"] = actors
	policy, err := decodeRulesetPolicy(copy)
	if err != nil {
		return settings.RulesetDeclaration{}, settings.RulesetNormalizations{}, err
	}
	return policy, settings.RulesetNormalizations{Enforcement: record, RequiredApprovals: approvals, MergeMethods: methods, ScanningAlerts: scanning, ScanningSecurityAlerts: scanningSecurity, BypassModes: bypass}, nil
}
