package github

import (
	"encoding/json"
	"fmt"

	"github.com/daiksud/gh-settings/internal/settings"
)

// prepareRulesetRequiredApprovalsRules projects one acquired scalar, not a complete policy.
// The caller must still use strict policy validation before publishing evidence.
func prepareRulesetRequiredApprovalsRules(raw json.RawMessage) (json.RawMessage, settings.RulesetRequiredApprovalsNormalization, error) {
	var rules []map[string]json.RawMessage
	if err := json.Unmarshal(raw, &rules); err != nil {
		return nil, settings.RulesetRequiredApprovalsNormalization{}, err
	}
	if rules == nil {
		return nil, settings.RulesetRequiredApprovalsNormalization{}, fmt.Errorf("Rulesets response requires a non-null rules array")
	}
	record := settings.RulesetRequiredApprovalsNormalization{}
	seen := false
	for _, rule := range rules {
		var kind string
		if err := requiredRuleJSON(rule, "type", &kind); err != nil {
			return nil, settings.RulesetRequiredApprovalsNormalization{}, err
		}
		if kind != "pull_request" {
			continue
		}
		if seen {
			return nil, settings.RulesetRequiredApprovalsNormalization{}, fmt.Errorf("duplicate pull_request rule")
		}
		seen = true
		var parameters map[string]json.RawMessage
		if err := requiredRuleJSON(rule, "parameters", &parameters); err != nil {
			return nil, settings.RulesetRequiredApprovalsNormalization{}, err
		}
		var count int
		if err := requiredRuleJSON(parameters, "required_approving_review_count", &count); err != nil {
			return nil, settings.RulesetRequiredApprovalsNormalization{}, err
		}
		if settings.RulesetRequiredApprovalsSupported(count) {
			continue
		}
		fallback := settings.DefaultRulesetRequiredApprovals
		value, err := json.Marshal(fallback)
		if err != nil {
			return nil, settings.RulesetRequiredApprovalsNormalization{}, err
		}
		parameters["required_approving_review_count"] = value
		encoded, err := json.Marshal(parameters)
		if err != nil {
			return nil, settings.RulesetRequiredApprovalsNormalization{}, err
		}
		rule["parameters"] = encoded
		record = settings.RulesetRequiredApprovalsNormalization{Path: settings.RulesetRequiredApprovalsPath, RawValue: count, Fallback: settings.DefaultRulesetRequiredApprovals}
	}
	encoded, err := json.Marshal(rules)
	if err != nil {
		return nil, settings.RulesetRequiredApprovalsNormalization{}, err
	}
	return encoded, record, nil
}
