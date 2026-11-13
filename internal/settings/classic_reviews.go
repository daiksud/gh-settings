package settings

import (
	"fmt"
	"reflect"

	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

type ClassicPullRequestReviews = uimodel.ClassicPullRequestReviews

func CloneClassicReviews(v *ClassicPullRequestReviews) *ClassicPullRequestReviews {
	if v == nil {
		return nil
	}
	r := *v
	r.Enabled = cloneClassicValue(v.Enabled)
	r.RequireApprovals = cloneClassicValue(v.RequireApprovals)
	r.RequiredNumberOfApprovalsBeforeMerging = cloneClassicValue(v.RequiredNumberOfApprovalsBeforeMerging)
	r.DismissStalePullRequestApprovalsWhenNewCommitsArePushed = cloneClassicValue(v.DismissStalePullRequestApprovalsWhenNewCommitsArePushed)
	r.RequireReviewFromCodeOwners = cloneClassicValue(v.RequireReviewFromCodeOwners)
	r.RequireApprovalOfTheMostRecentReviewablePush = cloneClassicValue(v.RequireApprovalOfTheMostRecentReviewablePush)
	return &r
}
func ClassicReviewsManaged(v *ClassicPullRequestReviews) bool { return v != nil && v.Enabled != nil }
func ValidateClassicReviews(v *ClassicPullRequestReviews) error {
	if v == nil || reflect.DeepEqual(*v, ClassicPullRequestReviews{}) {
		return nil
	}
	children := v.RequireApprovals != nil || v.RequiredNumberOfApprovalsBeforeMerging != nil || v.DismissStalePullRequestApprovalsWhenNewCommitsArePushed != nil || v.RequireReviewFromCodeOwners != nil || v.RequireApprovalOfTheMostRecentReviewablePush != nil
	if v.Enabled == nil || (!*v.Enabled && children) {
		return fmt.Errorf("classic PR reviews require explicit enabled and no inactive children")
	}
	if !*v.Enabled {
		return nil
	}
	if v.RequireApprovals == nil || v.DismissStalePullRequestApprovalsWhenNewCommitsArePushed == nil || v.RequireReviewFromCodeOwners == nil || v.RequireApprovalOfTheMostRecentReviewablePush == nil {
		return fmt.Errorf("enabled classic PR reviews require every active checkbox")
	}
	count := v.RequiredNumberOfApprovalsBeforeMerging
	if *v.RequireApprovals {
		if count == nil || *count < 1 || *count > 6 {
			return fmt.Errorf("classic approvals require an explicit count from 1 through 6")
		}
	} else if count != nil {
		return fmt.Errorf("disabled classic approvals forbid a count")
	}
	return nil
}

// ProjectClassicReviews never invents an active count or leaks inactive members.
// The caller retains c unchanged for omission preservation and pruning.
func ProjectClassicReviews(c ClassicCompanions) (*ClassicPullRequestReviews, error) {
	if !c.ReviewCount.Present {
		return nil, fmt.Errorf("missing acquired classic review count")
	}
	enabled := c.RequiresApprovingReviews
	r := &ClassicPullRequestReviews{Enabled: &enabled}
	if !enabled {
		return r, nil
	}
	if c.ReviewCount.Value == nil || *c.ReviewCount.Value < 0 || *c.ReviewCount.Value > 6 {
		return nil, fmt.Errorf("unavailable active classic review count")
	}
	approvals := *c.ReviewCount.Value > 0
	r.RequireApprovals = &approvals
	if approvals {
		r.RequiredNumberOfApprovalsBeforeMerging = cloneClassicValue(c.ReviewCount.Value)
	}
	r.DismissStalePullRequestApprovalsWhenNewCommitsArePushed = cloneClassicValue(&c.DismissesStaleReviews)
	r.RequireReviewFromCodeOwners = cloneClassicValue(&c.RequiresCodeOwnerReviews)
	r.RequireApprovalOfTheMostRecentReviewablePush = cloneClassicValue(&c.RequireLastPushApproval)
	return r, nil
}
func ClassicReviewsMatch(observed, desired *ClassicPullRequestReviews) bool {
	if !ClassicReviewsManaged(desired) {
		return true
	}
	return reflect.DeepEqual(observed, desired)
}
func ValidateClassicReviewTransition(before ClassicState, desired ClassicRule) error {
	d := desired.RequireAPullRequestBeforeMerging
	if ClassicReviewsManaged(d) && !*d.Enabled && before.Companions.RequiresApprovingReviews &&
		(before.Companions.RestrictsReviewDismissals || len(before.Companions.ReviewDismissalActors) > 0 || len(before.Companions.PullRequestActors) > 0) {
		return fmt.Errorf("cannot disable classic PR requirement with unowned bypass or dismissal protections")
	}
	return nil
}

// Equalize only selected review fields before checking every other companion.
// Disabling makes no promise about the inactive count or review booleans.
func maskSelectedClassicReviews(before *ClassicCompanions, after ClassicCompanions, desired *ClassicPullRequestReviews) {
	if !ClassicReviewsManaged(desired) {
		return
	}
	before.RequiresApprovingReviews = after.RequiresApprovingReviews
	before.ReviewCount = after.ReviewCount
	before.DismissesStaleReviews = after.DismissesStaleReviews
	before.RequiresCodeOwnerReviews = after.RequiresCodeOwnerReviews
	before.RequireLastPushApproval = after.RequireLastPushApproval
}
