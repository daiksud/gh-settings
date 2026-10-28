package github

import (
	"fmt"
	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

func declareUIRulesetPR(ui uimodel.PullRequestRule) (*settings.RuleParameters, error) {
	if ui.RequiredApprovals != nil && (*ui.RequiredApprovals < 0 || *ui.RequiredApprovals > 10) {
		return nil, fmt.Errorf("required_approvals must be a UI choice from 0 to 10")
	}
	result := &settings.RuleParameters{
		RequiredApprovingReviewCount:               copyUIRulesetValue(ui.RequiredApprovals),
		DismissStaleReviewsOnPush:                  copyUIRulesetValue(ui.DismissStalePullRequestApprovalsWhenNewCommitsArePushed),
		RequireCodeOwnerReview:                     copyUIRulesetValue(ui.RequireReviewFromCodeOwners),
		RequireLastPushApproval:                    copyUIRulesetValue(ui.RequireApprovalOfTheMostRecentReviewablePush),
		RequiredReviewThreadResolution:             copyUIRulesetValue(ui.RequireConversationResolutionBeforeMerging),
		RequireExtraApprovalForUnattributedChanges: copyUIRulesetValue(ui.RequireAnAdditionalApprovalForUnattributedCopilotPullRequests),
	}
	if ui.AllowedMergeMethods != nil {
		methods := make([]string, 0, len(*ui.AllowedMergeMethods))
		for _, choice := range *ui.AllowedMergeMethods {
			switch choice {
			case uimodel.Merge:
				methods = append(methods, "merge")
			case uimodel.Squash:
				methods = append(methods, "squash")
			case uimodel.Rebase:
				methods = append(methods, "rebase")
			default:
				return nil, fmt.Errorf("unsupported allowed_merge_methods UI choice")
			}
		}
		result.AllowedMergeMethods = &methods
	}
	var err error
	result.RequiredReviewers, err = declareUIRulesetTeams(ui.RequireReviewFromSpecificTeams)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func observeUIRulesetPR(p *settings.RuleParameters) (*uimodel.PullRequestRule, error) {
	if p == nil || p.RequiredApprovingReviewCount == nil || *p.RequiredApprovingReviewCount > 10 || *p.RequiredApprovingReviewCount < 0 {
		return nil, fmt.Errorf("required_approvals is missing or outside UI choices")
	}
	if p.DismissalRestriction != nil {
		return nil, fmt.Errorf("UI mapping unresolved for write-only review dismissal restrictions")
	}
	result := &uimodel.PullRequestRule{
		RequiredApprovals: copyUIRulesetValue(p.RequiredApprovingReviewCount),
		DismissStalePullRequestApprovalsWhenNewCommitsArePushed:       copyUIRulesetValue(p.DismissStaleReviewsOnPush),
		RequireReviewFromCodeOwners:                                   copyUIRulesetValue(p.RequireCodeOwnerReview),
		RequireApprovalOfTheMostRecentReviewablePush:                  copyUIRulesetValue(p.RequireLastPushApproval),
		RequireConversationResolutionBeforeMerging:                    copyUIRulesetValue(p.RequiredReviewThreadResolution),
		RequireAnAdditionalApprovalForUnattributedCopilotPullRequests: copyUIRulesetValue(p.RequireExtraApprovalForUnattributedChanges),
	}
	if p.AllowedMergeMethods == nil {
		return nil, fmt.Errorf("missing acquired allowed merge methods")
	}
	methods := make([]uimodel.MergeMethod, 0, len(*p.AllowedMergeMethods))
	for _, method := range *p.AllowedMergeMethods {
		switch method {
		case "merge":
			methods = append(methods, uimodel.Merge)
		case "squash":
			methods = append(methods, uimodel.Squash)
		case "rebase":
			methods = append(methods, uimodel.Rebase)
		default:
			return nil, fmt.Errorf("allowed merge methods outside UI choices")
		}
	}
	result.AllowedMergeMethods = &methods
	var err error
	result.RequireReviewFromSpecificTeams, err = observeUIRulesetTeams(p.RequiredReviewers)
	if err != nil {
		return nil, err
	}
	return result, nil
}
