package github

import (
	"reflect"
	"testing"

	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

func TestUIRulesetScanningChoices(t *testing.T) {
	for _, alerts := range []struct{ ui, api string }{{"none", "none"}, {"errors", "errors"}, {"errors_and_warnings", "errors_and_warnings"}, {"all", "all"}} {
		for _, security := range []struct{ ui, api string }{{"none", "none"}, {"critical", "critical"}, {"high_or_higher", "high_or_higher"}, {"medium_or_higher", "medium_or_higher"}, {"all", "all"}} {
			tools := []uimodel.CodeScanningTool{{ToolName: "CodeQL", Alerts: uimodel.CodeScanningAlerts(alerts.ui), SecurityAlerts: uimodel.CodeScanningSecurityAlerts(security.ui)}}
			ui := uiRulesetBase()
			ui.Rules.RequireCodeScanningResults = &uimodel.CodeScanningRule{CodeScanningTools: &tools}
			got, err := RulesetDeclaration(ui)
			want := []settings.CodeScanningTool{{Tool: "CodeQL", AlertsThreshold: alerts.api, SecurityAlertsThreshold: security.api}}
			if err != nil || (*got.Rules)[0].Type != "code_scanning" || !reflect.DeepEqual((*got.Rules)[0].Parameters.CodeScanningTools, &want) {
				t.Fatalf("rules=%+v err=%v want %+v", got.Rules, err, want)
			}
			source := apiRulesetBase()
			source.Rules = rulesetPtr([]settings.RulesetRule{{Type: "code_scanning", Parameters: &settings.RuleParameters{CodeScanningTools: &want}}})
			observed, err := RulesetObservation(source)
			if err != nil || !reflect.DeepEqual(observed.Rules.RequireCodeScanningResults, ui.Rules.RequireCodeScanningResults) {
				t.Fatalf("scanning=%+v err=%v", observed.Rules, err)
			}
		}
	}
}

func TestUIRulesetQualityChoices(t *testing.T) {
	for _, test := range []struct{ ui, api string }{{"errors", "errors"}, {"warnings_and_higher", "warnings"}, {"notes_and_higher", "notes"}, {"all", "all"}} {
		ui := uiRulesetBase()
		ui.Rules.RequireCodeQualityResults = &uimodel.CodeQualityRule{Severity: rulesetPtr(uimodel.CodeQualitySeverity(test.ui))}
		got, err := RulesetDeclaration(ui)
		if err != nil || (*got.Rules)[0].Type != "code_quality" || !reflect.DeepEqual((*got.Rules)[0].Parameters.Severity, &test.api) {
			t.Fatalf("rules=%+v err=%v", got.Rules, err)
		}
		source := apiRulesetBase()
		source.Rules = rulesetPtr([]settings.RulesetRule{{Type: "code_quality", Parameters: &settings.RuleParameters{Severity: &test.api}}})
		observed, err := RulesetObservation(source)
		if err != nil || !reflect.DeepEqual(observed.Rules.RequireCodeQualityResults, ui.Rules.RequireCodeQualityResults) {
			t.Fatalf("quality=%+v err=%v", observed.Rules, err)
		}
	}
}

func TestUIRulesetAnalysisRequiresExplicitChoices(t *testing.T) {
	for _, rules := range []uimodel.RulesetRules{
		{RequireCodeQualityResults: &uimodel.CodeQualityRule{}},
		{RequireCodeQualityResults: &uimodel.CodeQualityRule{Severity: rulesetPtr(uimodel.CodeQualitySeverity("warnings"))}},
		{RequireCodeScanningResults: &uimodel.CodeScanningRule{}},
		{RequireCodeScanningResults: &uimodel.CodeScanningRule{CodeScanningTools: rulesetPtr([]uimodel.CodeScanningTool{{ToolName: "CodeQL", Alerts: "UNKNOWN", SecurityAlerts: "high_or_higher"}})}},
	} {
		ui := uiRulesetBase()
		ui.Rules = rules
		if _, err := RulesetDeclaration(ui); err == nil {
			t.Fatal("analysis desired choice defaulted or API-only token accepted")
		}
	}
}
