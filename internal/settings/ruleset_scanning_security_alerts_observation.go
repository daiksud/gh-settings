package settings

import (
	"cmp"
	"fmt"
	"slices"

	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

const RulesetScanningSecurityAlertsPath = "repository.settings.rulesets.rules.require_code_scanning_results.code_scanning_tools.security_alerts"
const DefaultRulesetScanningSecurityAlertsChoice = uimodel.DefaultCodeScanningSecurityAlerts

func RulesetScanningSecurityAlertsChoiceForValue(value string) (uimodel.CodeScanningSecurityAlerts, bool) {
	switch value {
	case "none", "critical", "high_or_higher", "medium_or_higher", "all":
		return uimodel.CodeScanningSecurityAlerts(value), true
	}
	return "", false
}

func RulesetScanningSecurityAlertsValueForChoice(choice uimodel.CodeScanningSecurityAlerts) (string, bool) {
	_, ok := RulesetScanningSecurityAlertsChoiceForValue(string(choice))
	if !ok {
		return "", false
	}
	return string(choice), true
}

// RulesetScanningSecurityAlertsNormalization binds a raw threshold to an exact tool name.
type RulesetScanningSecurityAlertsNormalization struct {
	Tool     string
	Path     string
	RawValue string
	Fallback string
}

func (state RulesetState) validateScanningSecurityAlertsNormalizations() error {
	tools := map[string]CodeScanningTool{}
	if values := rulesetCodeScanningTools(state.Policy); values != nil {
		for _, tool := range *values {
			tools[tool.Tool] = tool
		}
	}
	seen := map[string]bool{}
	fallback, _ := RulesetScanningSecurityAlertsValueForChoice(DefaultRulesetScanningSecurityAlertsChoice)
	for _, record := range state.ScanningSecurityAlertsNormalizations {
		tool, found := tools[record.Tool]
		_, supported := RulesetScanningSecurityAlertsChoiceForValue(record.RawValue)
		if !found || seen[record.Tool] || tool.SecurityAlertsThreshold != fallback || record.Path != RulesetScanningSecurityAlertsPath || record.Fallback != string(DefaultRulesetScanningSecurityAlertsChoice) || supported {
			return fmt.Errorf("invalid Ruleset scanning security alerts observation provenance")
		}
		seen[record.Tool] = true
	}
	return nil
}

// OrderedRulesetScanningSecurityAlertsNormalizations owns records in exact tool-name order.
func OrderedRulesetScanningSecurityAlertsNormalizations(records []RulesetScanningSecurityAlertsNormalization) []RulesetScanningSecurityAlertsNormalization {
	result := slices.Clone(records)
	slices.SortFunc(result, func(a, b RulesetScanningSecurityAlertsNormalization) int { return cmp.Compare(a.Tool, b.Tool) })
	return result
}
