package github

import (
	"fmt"
	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

func declareUIRulesetRules(ui uimodel.RulesetRules) (*[]settings.RulesetRule, error) {
	result := []settings.RulesetRule{}
	for _, control := range []struct {
		selected *bool
		kind     string
	}{
		{ui.RestrictDeletions, "deletion"}, {ui.BlockForcePushes, "non_fast_forward"}, {ui.RequireLinearHistory, "required_linear_history"}, {ui.RequireSignedCommits, "required_signatures"},
	} {
		if control.selected != nil && *control.selected {
			result = append(result, settings.RulesetRule{Type: control.kind})
		}
	}
	if ui.RequireAPullRequestBeforeMerging != nil {
		parameters, err := declareUIRulesetPR(*ui.RequireAPullRequestBeforeMerging)
		if err != nil {
			return nil, err
		}
		result = append(result, settings.RulesetRule{Type: "pull_request", Parameters: parameters})
	}
	if ui.RequireStatusChecksToPass != nil {
		parameters, err := declareUIRulesetChecks(*ui.RequireStatusChecksToPass)
		if err != nil {
			return nil, err
		}
		result = append(result, settings.RulesetRule{Type: "required_status_checks", Parameters: parameters})
	}
	if ui.RequireCodeScanningResults != nil {
		parameters, err := declareUIRulesetScanning(*ui.RequireCodeScanningResults)
		if err != nil {
			return nil, err
		}
		result = append(result, settings.RulesetRule{Type: "code_scanning", Parameters: parameters})
	}
	if ui.RequireCodeQualityResults != nil {
		parameters, err := declareUIRulesetQuality(*ui.RequireCodeQualityResults)
		if err != nil {
			return nil, err
		}
		result = append(result, settings.RulesetRule{Type: "code_quality", Parameters: parameters})
	}
	return &result, nil
}

func observeUIRulesetRules(rules []settings.RulesetRule) (uimodel.RulesetRules, error) {
	no := false
	result := uimodel.RulesetRules{RestrictDeletions: copyUIRulesetValue(&no), BlockForcePushes: copyUIRulesetValue(&no), RequireLinearHistory: copyUIRulesetValue(&no), RequireSignedCommits: copyUIRulesetValue(&no)}
	for _, rule := range rules {
		switch rule.Type {
		case "deletion":
			*result.RestrictDeletions = true
		case "non_fast_forward":
			*result.BlockForcePushes = true
		case "required_linear_history":
			*result.RequireLinearHistory = true
		case "required_signatures":
			*result.RequireSignedCommits = true
		case "pull_request":
			var err error
			result.RequireAPullRequestBeforeMerging, err = observeUIRulesetPR(rule.Parameters)
			if err != nil {
				return uimodel.RulesetRules{}, err
			}
		case "required_status_checks":
			result.RequireStatusChecksToPass = observeUIRulesetChecks(rule.Parameters)
		case "code_scanning":
			var err error
			result.RequireCodeScanningResults, err = observeUIRulesetScanning(rule.Parameters)
			if err != nil {
				return uimodel.RulesetRules{}, err
			}
		case "code_quality":
			var err error
			result.RequireCodeQualityResults, err = observeUIRulesetQuality(rule.Parameters)
			if err != nil {
				return uimodel.RulesetRules{}, err
			}
		default:
			return uimodel.RulesetRules{}, fmt.Errorf("ruleset rule has no prepared UI mapping: %s", rule.Type)
		}
	}
	return result, nil
}
