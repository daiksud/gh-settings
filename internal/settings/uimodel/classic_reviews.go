package uimodel

// ClassicPullRequestReviews is one complete active review policy. Empty groups
// remain unmanaged; disabling owns only the outer requirement.
type ClassicPullRequestReviews struct {
	Enabled                                                 *bool `json:"enabled,omitempty"`
	RequireApprovals                                        *bool `json:"require_approvals,omitempty"`
	RequiredNumberOfApprovalsBeforeMerging                  *int  `json:"required_number_of_approvals_before_merging,omitempty"`
	DismissStalePullRequestApprovalsWhenNewCommitsArePushed *bool `json:"dismiss_stale_pull_request_approvals_when_new_commits_are_pushed,omitempty"`
	RequireReviewFromCodeOwners                             *bool `json:"require_review_from_code_owners,omitempty"`
	RequireApprovalOfTheMostRecentReviewablePush            *bool `json:"require_approval_of_the_most_recent_reviewable_push,omitempty"`
}
