package github

import (
	"fmt"
	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

func declareUIRulesetScanning(ui uimodel.CodeScanningRule) (*settings.RuleParameters, error) {
	if ui.CodeScanningTools == nil {
		return nil, fmt.Errorf("require_code_scanning_results requires code_scanning_tools")
	}
	tools := make([]settings.CodeScanningTool, 0, len(*ui.CodeScanningTools))
	for _, tool := range *ui.CodeScanningTools {
		alerts, valid := settings.RulesetScanningAlertsValueForChoice(tool.Alerts)
		if !valid {
			return nil, fmt.Errorf("unsupported UI alerts choice %q", tool.Alerts)
		}
		security, valid := settings.RulesetScanningSecurityAlertsValueForChoice(tool.SecurityAlerts)
		if !valid {
			return nil, fmt.Errorf("unsupported UI security_alerts choice %q", tool.SecurityAlerts)
		}
		tools = append(tools, settings.CodeScanningTool{Tool: tool.ToolName, AlertsThreshold: alerts, SecurityAlertsThreshold: security})
	}
	return &settings.RuleParameters{CodeScanningTools: &tools}, nil
}

func observeUIRulesetScanning(p *settings.RuleParameters) (*uimodel.CodeScanningRule, error) {
	tools := make([]uimodel.CodeScanningTool, 0, len(*p.CodeScanningTools))
	for _, tool := range *p.CodeScanningTools {
		alerts, valid := settings.RulesetScanningAlertsChoiceForValue(tool.AlertsThreshold)
		if !valid {
			return nil, fmt.Errorf("alerts is outside UI choices")
		}
		security, valid := settings.RulesetScanningSecurityAlertsChoiceForValue(tool.SecurityAlertsThreshold)
		if !valid {
			return nil, fmt.Errorf("security_alerts is outside UI choices")
		}
		tools = append(tools, uimodel.CodeScanningTool{ToolName: tool.Tool, Alerts: alerts, SecurityAlerts: security})
	}
	return &uimodel.CodeScanningRule{CodeScanningTools: &tools}, nil
}

func declareUIRulesetQuality(ui uimodel.CodeQualityRule) (*settings.RuleParameters, error) {
	if ui.Severity == nil {
		return nil, fmt.Errorf("require_code_quality_results requires severity")
	}
	severity, valid := settings.RulesetQualitySeverityValueForChoice(*ui.Severity)
	if !valid {
		return nil, fmt.Errorf("unsupported UI severity choice %q", *ui.Severity)
	}
	return &settings.RuleParameters{Severity: &severity}, nil
}

func observeUIRulesetQuality(p *settings.RuleParameters) (*uimodel.CodeQualityRule, error) {
	severity, valid := settings.RulesetQualitySeverityChoiceForValue(*p.Severity)
	if !valid {
		return nil, fmt.Errorf("severity is outside UI choices")
	}
	return &uimodel.CodeQualityRule{Severity: &severity}, nil
}
