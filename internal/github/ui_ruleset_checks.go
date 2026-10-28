package github

import (
	"fmt"
	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

func declareUIRulesetChecks(ui uimodel.StatusChecksRule) (*settings.RuleParameters, error) {
	result := &settings.RuleParameters{StrictRequiredStatusChecksPolicy: copyUIRulesetValue(ui.RequireBranchesToBeUpToDateBeforeMerging), DoNotEnforceOnCreate: copyUIRulesetValue(ui.DoNotRequireStatusChecksOnCreation)}
	if ui.RequiredStatusChecks == nil {
		return result, nil
	}
	checks := make([]settings.RequiredStatusCheck, 0, len(*ui.RequiredStatusChecks))
	for _, check := range *ui.RequiredStatusChecks {
		if check.Source.AnySource != nil && (!*check.Source.AnySource || check.Source.App != nil) {
			return nil, fmt.Errorf("check source requires one selected Any source or App")
		}
		if check.Source.App != nil && *check.Source.App <= 0 {
			return nil, fmt.Errorf("check source App requires positive identity")
		}
		checks = append(checks, settings.RequiredStatusCheck{Context: check.Check, IntegrationID: copyUIRulesetValue(check.Source.App)})
	}
	result.RequiredStatusChecks = &checks
	return result, nil
}

// The enclosing observation converter has validated complete rule parameters.
func observeUIRulesetChecks(p *settings.RuleParameters) *uimodel.StatusChecksRule {
	checks := make([]uimodel.StatusCheck, 0, len(*p.RequiredStatusChecks))
	for _, check := range *p.RequiredStatusChecks {
		source := uimodel.CheckSource{App: copyUIRulesetValue(check.IntegrationID)}
		if source.App == nil {
			selected := true
			source.AnySource = &selected
		}
		checks = append(checks, uimodel.StatusCheck{Check: check.Context, Source: source})
	}
	return &uimodel.StatusChecksRule{RequiredStatusChecks: &checks, RequireBranchesToBeUpToDateBeforeMerging: copyUIRulesetValue(p.StrictRequiredStatusChecksPolicy), DoNotRequireStatusChecksOnCreation: copyUIRulesetValue(p.DoNotEnforceOnCreate)}
}
