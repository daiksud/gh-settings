package github

import (
	"encoding/json"
	"fmt"
	"github.com/daiksud/gh-settings/internal/settings"
	"strings"
)

// prepareRulesetScanningAlertsRules projects complete strings, never tool identities.
// Strict whole-policy validation must succeed before the evidence is published.
func prepareRulesetScanningAlertsRules(raw json.RawMessage) (json.RawMessage, []settings.RulesetScanningAlertsNormalization, error) {
	var rules []map[string]json.RawMessage
	if err := json.Unmarshal(raw, &rules); err != nil {
		return nil, nil, err
	}
	if rules == nil {
		return nil, nil, fmt.Errorf("Rulesets response requires a non-null rules array")
	}
	var records []settings.RulesetScanningAlertsNormalization
	seenRule := false
	for _, rule := range rules {
		var kind string
		if err := requiredRuleJSON(rule, "type", &kind); err != nil {
			return nil, nil, err
		}
		if kind != "code_scanning" {
			continue
		}
		if seenRule {
			return nil, nil, fmt.Errorf("duplicate code_scanning rule")
		}
		seenRule = true
		var parameters map[string]json.RawMessage
		if err := requiredRuleJSON(rule, "parameters", &parameters); err != nil {
			return nil, nil, err
		}
		var tools []map[string]json.RawMessage
		if err := requiredRuleJSON(parameters, "code_scanning_tools", &tools); err != nil {
			return nil, nil, err
		}
		seen := map[string]bool{}
		changed := false
		for _, tool := range tools {
			var name, alerts string
			if err := requiredRuleJSON(tool, "tool", &name); err != nil {
				return nil, nil, err
			}
			if strings.TrimSpace(name) == "" || seen[name] {
				return nil, nil, fmt.Errorf("invalid or duplicate code scanning tool")
			}
			seen[name] = true
			if err := requiredRuleJSON(tool, "alerts_threshold", &alerts); err != nil {
				return nil, nil, err
			}
			if _, supported := settings.RulesetScanningAlertsChoiceForValue(alerts); supported {
				continue
			}
			fallback, _ := settings.RulesetScanningAlertsValueForChoice(settings.DefaultRulesetScanningAlertsChoice)
			encoded, err := json.Marshal(fallback)
			if err != nil {
				return nil, nil, err
			}
			tool["alerts_threshold"] = encoded
			records = append(records, settings.RulesetScanningAlertsNormalization{Tool: name, Path: settings.RulesetScanningAlertsPath, RawValue: alerts, Fallback: string(settings.DefaultRulesetScanningAlertsChoice)})
			changed = true
		}
		if changed {
			encoded, err := json.Marshal(tools)
			if err != nil {
				return nil, nil, err
			}
			parameters["code_scanning_tools"] = encoded
			encoded, err = json.Marshal(parameters)
			if err != nil {
				return nil, nil, err
			}
			rule["parameters"] = encoded
		}
	}
	encoded, err := json.Marshal(rules)
	if err != nil {
		return nil, nil, err
	}
	return encoded, records, nil
}
