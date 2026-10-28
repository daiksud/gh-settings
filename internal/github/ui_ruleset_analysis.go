package github

import (
	"fmt"
	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

var uiRulesetScanningAlerts = map[uimodel.CodeScanningAlerts]string{"none": "none", "errors": "errors", "errors_and_warnings": "errors_and_warnings", "all": "all"}
var uiRulesetSecurityAlerts = map[uimodel.CodeScanningSecurityAlerts]string{"none": "none", "critical": "critical", "high_or_higher": "high_or_higher", "medium_or_higher": "medium_or_higher", "all": "all"}
var uiRulesetQualitySeverity = map[uimodel.CodeQualitySeverity]string{"errors": "errors", "warnings_and_higher": "warnings", "notes_and_higher": "notes", "all": "all"}

func declareUIRulesetChoice[T ~string](choice T, choices map[T]string, label string) (string, error) {
	value, ok := choices[choice]
	if !ok {
		return "", fmt.Errorf("unsupported UI %s choice %q", label, choice)
	}
	return value, nil
}

func observeUIRulesetChoice[T ~string](value string, choices map[T]string, label string) (T, error) {
	for choice, api := range choices {
		if api == value {
			return choice, nil
		}
	}
	return "", fmt.Errorf("%s is outside UI choices", label)
}

func declareUIRulesetScanning(ui uimodel.CodeScanningRule) (*settings.RuleParameters, error) {
	if ui.CodeScanningTools == nil {
		return nil, fmt.Errorf("require_code_scanning_results requires code_scanning_tools")
	}
	tools := make([]settings.CodeScanningTool, 0, len(*ui.CodeScanningTools))
	for _, tool := range *ui.CodeScanningTools {
		alerts, err := declareUIRulesetChoice(tool.Alerts, uiRulesetScanningAlerts, "alerts")
		if err != nil {
			return nil, err
		}
		security, err := declareUIRulesetChoice(tool.SecurityAlerts, uiRulesetSecurityAlerts, "security_alerts")
		if err != nil {
			return nil, err
		}
		tools = append(tools, settings.CodeScanningTool{Tool: tool.ToolName, AlertsThreshold: alerts, SecurityAlertsThreshold: security})
	}
	return &settings.RuleParameters{CodeScanningTools: &tools}, nil
}

func observeUIRulesetScanning(p *settings.RuleParameters) (*uimodel.CodeScanningRule, error) {
	tools := make([]uimodel.CodeScanningTool, 0, len(*p.CodeScanningTools))
	for _, tool := range *p.CodeScanningTools {
		alerts, err := observeUIRulesetChoice(tool.AlertsThreshold, uiRulesetScanningAlerts, "alerts")
		if err != nil {
			return nil, err
		}
		security, err := observeUIRulesetChoice(tool.SecurityAlertsThreshold, uiRulesetSecurityAlerts, "security_alerts")
		if err != nil {
			return nil, err
		}
		tools = append(tools, uimodel.CodeScanningTool{ToolName: tool.Tool, Alerts: alerts, SecurityAlerts: security})
	}
	return &uimodel.CodeScanningRule{CodeScanningTools: &tools}, nil
}

func declareUIRulesetQuality(ui uimodel.CodeQualityRule) (*settings.RuleParameters, error) {
	if ui.Severity == nil {
		return nil, fmt.Errorf("require_code_quality_results requires severity")
	}
	severity, err := declareUIRulesetChoice(*ui.Severity, uiRulesetQualitySeverity, "severity")
	if err != nil {
		return nil, err
	}
	return &settings.RuleParameters{Severity: &severity}, nil
}

func observeUIRulesetQuality(p *settings.RuleParameters) (*uimodel.CodeQualityRule, error) {
	severity, err := observeUIRulesetChoice(*p.Severity, uiRulesetQualitySeverity, "severity")
	if err != nil {
		return nil, err
	}
	return &uimodel.CodeQualityRule{Severity: &severity}, nil
}
