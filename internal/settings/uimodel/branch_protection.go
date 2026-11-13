package uimodel

// BranchProtectionRule owns only the selected classic controls, independently of Rulesets.
type BranchProtectionRule struct {
	RequireAPullRequestBeforeMerging           *ClassicPullRequestReviews `json:"require_a_pull_request_before_merging,omitempty"`
	ID                                         *int64                     `json:"id,omitempty"`
	BranchNamePattern                          string                     `json:"branch_name_pattern"`
	RequireConversationResolutionBeforeMerging *bool                      `json:"require_conversation_resolution_before_merging,omitempty"`
	RequireSignedCommits                       *bool                      `json:"require_signed_commits,omitempty"`
	RequireLinearHistory                       *bool                      `json:"require_linear_history,omitempty"`
	DoNotAllowBypassingTheAboveSettings        *bool                      `json:"do_not_allow_bypassing_the_above_settings,omitempty"`
	LockBranch                                 *bool                      `json:"lock_branch,omitempty"`
	AllowDeletions                             *bool                      `json:"allow_deletions,omitempty"`
	RequireDeploymentsToSucceedBeforeMerging   *ClassicDeployments        `json:"require_deployments_to_succeed_before_merging,omitempty"`
}

// ClassicDeployments is a complete active configuration, independent of latent names.
type ClassicDeployments struct {
	Enabled      *bool     `json:"enabled,omitempty"`
	Environments *[]string `json:"environments,omitempty"`
}
