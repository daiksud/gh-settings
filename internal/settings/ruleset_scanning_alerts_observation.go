package settings

import (
	"cmp"
	"fmt"
	"slices"

	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

const RulesetScanningAlertsPath = "repository.settings.rulesets.rules.require_code_scanning_results.code_scanning_tools.alerts"
const DefaultRulesetScanningAlertsChoice = uimodel.DefaultCodeScanningAlerts

func RulesetScanningAlertsChoiceForValue(value string) (uimodel.CodeScanningAlerts, bool) {
	switch value {
	case "none", "errors", "errors_and_warnings", "all":
		return uimodel.CodeScanningAlerts(value), true
	}
	return "", false
}

func RulesetScanningAlertsValueForChoice(choice uimodel.CodeScanningAlerts) (string, bool) {
	_, ok := RulesetScanningAlertsChoiceForValue(string(choice))
	if !ok {
		return "", false
	}
	return string(choice), true
}

// RulesetScanningAlertsNormalization binds a raw threshold to an exact tool name.
type RulesetScanningAlertsNormalization struct {
	Tool     string
	Path     string
	RawValue string
	Fallback string
}

func rulesetCodeScanningTools(policy RulesetDeclaration) *[]CodeScanningTool {
	if policy.Rules != nil {
		for _, rule := range *policy.Rules {
			if rule.Type == "code_scanning" && rule.Parameters != nil {
				return rule.Parameters.CodeScanningTools
			}
		}
	}
	return nil
}

// RawCodeScanningTools returns owned original thresholds after validating evidence.
// Nil means an absent scanning rule; a present empty tool list stays empty.
func (state RulesetState) RawCodeScanningTools() ([]CodeScanningTool, error) {
	if err := state.validateNormalizationEvidence(); err != nil {
		return nil, err
	}
	tools := rulesetCodeScanningTools(cloneRuleset(state.Policy))
	if tools == nil {
		return nil, nil
	}
	for _, record := range state.ScanningAlertsNormalizations {
		for i := range *tools {
			if (*tools)[i].Tool == record.Tool {
				(*tools)[i].AlertsThreshold = record.RawValue
			}
		}
	}
	for _, record := range state.ScanningSecurityAlertsNormalizations {
		for i := range *tools {
			if (*tools)[i].Tool == record.Tool {
				(*tools)[i].SecurityAlertsThreshold = record.RawValue
			}
		}
	}
	return *tools, nil
}

func (state RulesetState) validateScanningAlertsNormalizations() error {
	tools := map[string]CodeScanningTool{}
	if values := rulesetCodeScanningTools(state.Policy); values != nil {
		for _, tool := range *values {
			tools[tool.Tool] = tool
		}
	}
	seen := map[string]bool{}
	fallback, _ := RulesetScanningAlertsValueForChoice(DefaultRulesetScanningAlertsChoice)
	for _, record := range state.ScanningAlertsNormalizations {
		tool, found := tools[record.Tool]
		_, supported := RulesetScanningAlertsChoiceForValue(record.RawValue)
		if !found || seen[record.Tool] || tool.AlertsThreshold != fallback || record.Path != RulesetScanningAlertsPath || record.Fallback != string(DefaultRulesetScanningAlertsChoice) || supported {
			return fmt.Errorf("invalid Ruleset scanning alerts observation provenance")
		}
		seen[record.Tool] = true
	}
	return nil
}

// OrderedRulesetScanningAlertsNormalizations owns records in exact tool-name order.
func OrderedRulesetScanningAlertsNormalizations(records []RulesetScanningAlertsNormalization) []RulesetScanningAlertsNormalization {
	result := slices.Clone(records)
	slices.SortFunc(result, func(a, b RulesetScanningAlertsNormalization) int { return cmp.Compare(a.Tool, b.Tool) })
	return result
}
