package github

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/daiksud/gh-settings/internal/settings"
)

func normalizeScanningAlerts(parameters map[string]json.RawMessage, records *settings.RulesetNormalizations) (bool, error) {
	return normalizeScanningTools(parameters, "alerts_threshold", func(tool, raw string) (string, bool) {
		if _, supported := settings.RulesetScanningAlertsChoiceForValue(raw); supported {
			return "", false
		}
		choice := settings.DefaultRulesetScanningAlertsChoice
		records.ScanningAlerts = append(records.ScanningAlerts, settings.RulesetScanningAlertsNormalization{
			Tool: tool, Path: settings.RulesetScanningAlertsPath, RawValue: raw, Fallback: string(choice),
		})
		fallback, _ := settings.RulesetScanningAlertsValueForChoice(choice)
		return fallback, true
	})
}

func normalizeScanningSecurityAlerts(parameters map[string]json.RawMessage, records *settings.RulesetNormalizations) (bool, error) {
	return normalizeScanningTools(parameters, "security_alerts_threshold", func(tool, raw string) (string, bool) {
		if _, supported := settings.RulesetScanningSecurityAlertsChoiceForValue(raw); supported {
			return "", false
		}
		choice := settings.DefaultRulesetScanningSecurityAlertsChoice
		records.ScanningSecurityAlerts = append(records.ScanningSecurityAlerts, settings.RulesetScanningSecurityAlertsNormalization{
			Tool: tool, Path: settings.RulesetScanningSecurityAlertsPath, RawValue: raw, Fallback: string(choice),
		})
		fallback, _ := settings.RulesetScanningSecurityAlertsValueForChoice(choice)
		return fallback, true
	})
}

func normalizeScanningTools(parameters map[string]json.RawMessage, threshold string, fallbackFor func(tool, raw string) (string, bool)) (bool, error) {
	var tools []map[string]json.RawMessage
	if err := requiredRuleJSON(parameters, "code_scanning_tools", &tools); err != nil {
		return false, err
	}
	seen := map[string]bool{}
	changed := false
	for _, tool := range tools {
		var name, raw string
		if err := requiredRuleJSON(tool, "tool", &name); err != nil {
			return false, err
		}
		if strings.TrimSpace(name) == "" || seen[name] {
			return false, fmt.Errorf("invalid or duplicate code scanning tool")
		}
		seen[name] = true
		if err := requiredRuleJSON(tool, threshold, &raw); err != nil {
			return false, err
		}
		if fallback, needed := fallbackFor(name, raw); needed {
			if err := setRuleJSON(tool, threshold, fallback); err != nil {
				return false, err
			}
			changed = true
		}
	}
	if !changed {
		return false, nil
	}
	return true, setRuleJSON(parameters, "code_scanning_tools", tools)
}
