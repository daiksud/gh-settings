package github

import "github.com/daiksud/gh-settings/internal/settings"

func classicReviewInputs(input map[string]any, before *settings.ClassicState, desired *settings.ClassicPullRequestReviews) {
	if !settings.ClassicReviewsManaged(desired) {
		return
	}
	if !*desired.Enabled {
		if before == nil || before.Companions.RequiresApprovingReviews {
			input["requiresApprovingReviews"] = false
		}
		return
	}
	count := 0
	if *desired.RequireApprovals {
		count = *desired.RequiredNumberOfApprovalsBeforeMerging
	}
	all := before == nil || !before.Companions.RequiresApprovingReviews
	values := map[string]any{
		"requiresApprovingReviews":     true,
		"requiredApprovingReviewCount": count,
		"dismissesStaleReviews":        *desired.DismissStalePullRequestApprovalsWhenNewCommitsArePushed,
		"requiresCodeOwnerReviews":     *desired.RequireReviewFromCodeOwners,
		"requireLastPushApproval":      *desired.RequireApprovalOfTheMostRecentReviewablePush,
	}
	if all {
		for k, v := range values {
			input[k] = v
		}
		return
	}
	current := map[string]any{
		"requiresApprovingReviews":     before.Companions.RequiresApprovingReviews,
		"requiredApprovingReviewCount": *before.Companions.ReviewCount.Value,
		"dismissesStaleReviews":        before.Companions.DismissesStaleReviews,
		"requiresCodeOwnerReviews":     before.Companions.RequiresCodeOwnerReviews,
		"requireLastPushApproval":      before.Companions.RequireLastPushApproval,
	}
	for k, v := range values {
		if v != current[k] {
			input[k] = v
		}
	}
}
