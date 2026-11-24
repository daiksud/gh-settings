package github

import (
	"fmt"
	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

func declareUIRulesetPR(ui uimodel.PullRequestRule) (*settings.RuleParameters, error) {
	if ui.RequiredApprovals != nil && !settings.RulesetRequiredApprovalsSupported(*ui.RequiredApprovals) {
		return nil, fmt.Errorf("required_approvals must be a UI choice from 0 to 10")
	}
	result := &settings.RuleParameters{
		RequireExtraApprovalForUnattributedChanges: copyUIValue(ui.RequireAnAdditionalApprovalForUnattributedCopilotPullRequests),
		RequiredApprovingReviewCount:               copyUIValue(ui.RequiredApprovals),
		DismissStaleReviewsOnPush:                  copyUIValue(ui.DismissStalePullRequestApprovalsWhenNewCommitsArePushed),
		RequireCodeOwnerReview:                     copyUIValue(ui.RequireReviewFromCodeOwners),
		RequireLastPushApproval:                    copyUIValue(ui.RequireApprovalOfTheMostRecentReviewablePush),
		RequiredReviewThreadResolution:             copyUIValue(ui.RequireConversationResolutionBeforeMerging),
	}
	if ui.AllowedMergeMethods != nil {
		methods := make([]string, 0, len(*ui.AllowedMergeMethods))
		for _, choice := range *ui.AllowedMergeMethods {
			method, supported := settings.RulesetMergeMethodValueForChoice(choice)
			if !supported {
				return nil, fmt.Errorf("unsupported allowed_merge_methods UI choice")
			}
			methods = append(methods, method)
		}
		result.AllowedMergeMethods = &methods
	}
	return result, nil
}

func observeUIRulesetPR(p *settings.RuleParameters) (*uimodel.PullRequestRule, error) {
	if p == nil || p.RequiredApprovingReviewCount == nil || !settings.RulesetRequiredApprovalsSupported(*p.RequiredApprovingReviewCount) {
		return nil, fmt.Errorf("required_approvals is missing or outside UI choices")
	}
	result := &uimodel.PullRequestRule{
		RequireAnAdditionalApprovalForUnattributedCopilotPullRequests: copyUIValue(p.RequireExtraApprovalForUnattributedChanges),
		RequiredApprovals: copyUIValue(p.RequiredApprovingReviewCount),
		DismissStalePullRequestApprovalsWhenNewCommitsArePushed: copyUIValue(p.DismissStaleReviewsOnPush),
		RequireReviewFromCodeOwners:                             copyUIValue(p.RequireCodeOwnerReview),
		RequireApprovalOfTheMostRecentReviewablePush:            copyUIValue(p.RequireLastPushApproval),
		RequireConversationResolutionBeforeMerging:              copyUIValue(p.RequiredReviewThreadResolution),
	}
	if p.AllowedMergeMethods == nil {
		return nil, fmt.Errorf("missing acquired allowed merge methods")
	}
	methods := make([]uimodel.MergeMethod, 0, len(*p.AllowedMergeMethods))
	for _, method := range *p.AllowedMergeMethods {
		choice, supported := settings.RulesetMergeMethodChoiceForValue(method)
		if !supported {
			return nil, fmt.Errorf("allowed merge methods outside UI choices")
		}
		methods = append(methods, choice)
	}
	result.AllowedMergeMethods = &methods
	return result, nil
}
