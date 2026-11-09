package github

import (
	"encoding/json"
	"fmt"

	"github.com/daiksud/gh-settings/internal/settings"
)

// prepareRulesetQualityRules projects one acquired scalar, not a complete policy.
// The caller must still use strict policy validation before publishing evidence.
func prepareRulesetQualityRules(raw json.RawMessage) (json.RawMessage, settings.ObservationNormalization, error) {
	var rules []map[string]json.RawMessage
	if err := json.Unmarshal(raw, &rules); err != nil {
		return nil, settings.ObservationNormalization{}, err
	}
	if rules == nil {
		return nil, settings.ObservationNormalization{}, fmt.Errorf("Rulesets response requires a non-null rules array")
	}
	record := settings.ObservationNormalization{}
	seen := false
	for _, rule := range rules {
		var kind string
		if err := requiredRuleJSON(rule, "type", &kind); err != nil {
			return nil, settings.ObservationNormalization{}, err
		}
		if kind != "code_quality" {
			continue
		}
		if seen {
			return nil, settings.ObservationNormalization{}, fmt.Errorf("duplicate code_quality rule")
		}
		seen = true
		var parameters map[string]json.RawMessage
		if err := requiredRuleJSON(rule, "parameters", &parameters); err != nil {
			return nil, settings.ObservationNormalization{}, err
		}
		var severity string
		if err := requiredRuleJSON(parameters, "severity", &severity); err != nil {
			return nil, settings.ObservationNormalization{}, err
		}
		if _, supported := settings.RulesetQualitySeverityChoiceForValue(severity); supported {
			continue
		}
		fallback, _ := settings.RulesetQualitySeverityValueForChoice(settings.DefaultRulesetQualitySeverityChoice)
		value, err := json.Marshal(fallback)
		if err != nil {
			return nil, settings.ObservationNormalization{}, err
		}
		parameters["severity"] = value
		encoded, err := json.Marshal(parameters)
		if err != nil {
			return nil, settings.ObservationNormalization{}, err
		}
		rule["parameters"] = encoded
		record = settings.ObservationNormalization{Path: settings.RulesetQualitySeverityPath, RawValue: severity, Fallback: string(settings.DefaultRulesetQualitySeverityChoice)}
	}
	encoded, err := json.Marshal(rules)
	if err != nil {
		return nil, settings.ObservationNormalization{}, err
	}
	return encoded, record, nil
}
