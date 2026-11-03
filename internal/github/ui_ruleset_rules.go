package github

import (
	"fmt"
	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

func declareUIRulesetRules(ui uimodel.RulesetRules) (*[]settings.RulesetRule, error) {
	result := []settings.RulesetRule{}
	if ui.AutomaticallyRequestCopilotCodeReview != nil {
		rule := ui.AutomaticallyRequestCopilotCodeReview
		result = append(result, settings.RulesetRule{Type: "copilot_code_review", Parameters: &settings.RuleParameters{ReviewOnPush: copyUIRulesetValue(rule.ReviewNewPushes), ReviewDraftPullRequests: copyUIRulesetValue(rule.ReviewDraftPullRequests)}})
	}
	for _, control := range []struct {
		selected *bool
		kind     string
	}{
		{ui.RestrictCreations, "creation"}, {ui.RestrictDeletions, "deletion"}, {ui.BlockForcePushes, "non_fast_forward"}, {ui.RequireLinearHistory, "required_linear_history"}, {ui.RequireSignedCommits, "required_signatures"},
	} {
		if control.selected != nil && *control.selected {
			result = append(result, settings.RulesetRule{Type: control.kind})
		}
	}
	if ui.RestrictUpdates != nil {
		result = append(result, settings.RulesetRule{Type: "update", Parameters: &settings.RuleParameters{UpdateAllowsFetchAndMerge: copyUIRulesetValue(ui.RestrictUpdates.AllowFetchAndMerge)}})
	}
	if ui.RequireDeploymentsToSucceed != nil {
		if ui.RequireDeploymentsToSucceed.RequiredDeploymentEnvironments == nil {
			return nil, fmt.Errorf("require_deployments_to_succeed requires required_deployment_environments")
		}
		environments := append([]string{}, (*ui.RequireDeploymentsToSucceed.RequiredDeploymentEnvironments)...)
		result = append(result, settings.RulesetRule{Type: "required_deployments", Parameters: &settings.RuleParameters{RequiredDeploymentEnvironments: &environments}})
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
	if ui.RestrictCodeCoverage != nil {
		p := ui.RestrictCodeCoverage
		result = append(result, settings.RulesetRule{Type: "code_coverage", Parameters: &settings.RuleParameters{MinimumCoverage: copyUIRulesetValue(p.MinimumLineCoveragePercentage), MaxCoverageDrop: copyUIRulesetValue(p.MaximumLineCoverageDrop)}})
	}
	return &result, nil
}

func observeUIRulesetRules(rules []settings.RulesetRule) (uimodel.RulesetRules, error) {
	no := false
	result := uimodel.RulesetRules{RestrictCreations: copyUIRulesetValue(&no), RestrictDeletions: copyUIRulesetValue(&no), BlockForcePushes: copyUIRulesetValue(&no), RequireLinearHistory: copyUIRulesetValue(&no), RequireSignedCommits: copyUIRulesetValue(&no)}
	for _, rule := range rules {
		switch rule.Type {
		case "creation":
			*result.RestrictCreations = true
		case "update":
			if rule.Parameters == nil || rule.Parameters.UpdateAllowsFetchAndMerge == nil {
				return uimodel.RulesetRules{}, fmt.Errorf("update requires complete parameters")
			}
			result.RestrictUpdates = &uimodel.UpdateRule{AllowFetchAndMerge: copyUIRulesetValue(rule.Parameters.UpdateAllowsFetchAndMerge)}
		case "copilot_code_review":
			result.AutomaticallyRequestCopilotCodeReview = &uimodel.CopilotCodeReviewRule{ReviewNewPushes: copyUIRulesetValue(rule.Parameters.ReviewOnPush), ReviewDraftPullRequests: copyUIRulesetValue(rule.Parameters.ReviewDraftPullRequests)}
		case "deletion":
			*result.RestrictDeletions = true
		case "non_fast_forward":
			*result.BlockForcePushes = true
		case "required_linear_history":
			*result.RequireLinearHistory = true
		case "required_signatures":
			*result.RequireSignedCommits = true
		case "required_deployments":
			if rule.Parameters == nil || rule.Parameters.RequiredDeploymentEnvironments == nil {
				return uimodel.RulesetRules{}, fmt.Errorf("required_deployments missing environments")
			}
			environments := append([]string{}, (*rule.Parameters.RequiredDeploymentEnvironments)...)
			result.RequireDeploymentsToSucceed = &uimodel.DeploymentsRule{RequiredDeploymentEnvironments: &environments}
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
		case "code_coverage":
			p := rule.Parameters
			result.RestrictCodeCoverage = &uimodel.CodeCoverageRule{MinimumLineCoveragePercentage: copyUIRulesetValue(p.MinimumCoverage), MaximumLineCoverageDrop: copyUIRulesetValue(p.MaxCoverageDrop)}
		default:
			return uimodel.RulesetRules{}, fmt.Errorf("ruleset rule has no prepared UI mapping: %s", rule.Type)
		}
	}
	return result, nil
}
