package uimodel

// BranchProtectionRule owns only the selected classic controls, independently of Rulesets.
type BranchProtectionRule struct {
	AllowForkSyncing                           *bool                      `json:"allow_fork_syncing,omitempty"`
	AllowForcePushes                           *ClassicForcePush          `json:"allow_force_pushes,omitempty"`
	RequireAPullRequestBeforeMerging           *ClassicPullRequestReviews `json:"require_a_pull_request_before_merging,omitempty"`
	RequireStatusChecksToPassBeforeMerging     *ClassicStatusChecks       `json:"require_status_checks_to_pass_before_merging,omitempty"`
	RequireDeploymentsToSucceedBeforeMerging   *ClassicDeployments        `json:"require_deployments_to_succeed_before_merging,omitempty"`
	ID                                         *int64                     `json:"id,omitempty"`
	BranchNamePattern                          string                     `json:"branch_name_pattern"`
	RequireConversationResolutionBeforeMerging *bool                      `json:"require_conversation_resolution_before_merging,omitempty"`
	RequireSignedCommits                       *bool                      `json:"require_signed_commits,omitempty"`
	RequireLinearHistory                       *bool                      `json:"require_linear_history,omitempty"`
	DoNotAllowBypassingTheAboveSettings        *bool                      `json:"do_not_allow_bypassing_the_above_settings,omitempty"`
	LockBranch                                 *bool                      `json:"lock_branch,omitempty"`
	AllowDeletions                             *bool                      `json:"allow_deletions,omitempty"`
}

// ClassicStatusChecks is either unmanaged, disabled, or a complete active policy.
type ClassicStatusChecks struct {
	Enabled                                  *bool                 `json:"enabled,omitempty"`
	RequireBranchesToBeUpToDateBeforeMerging *bool                 `json:"require_branches_to_be_up_to_date_before_merging,omitempty"`
	StatusChecks                             *[]ClassicStatusCheck `json:"status_checks,omitempty"`
}

type ClassicStatusCheck struct {
	Name              string `json:"name"`
	AcceptUpdatesFrom string `json:"accept_updates_from"`
}

// ClassicDeployments is a complete active configuration, independent of latent names.
type ClassicDeployments struct {
	Enabled      *bool     `json:"enabled,omitempty"`
	Environments *[]string `json:"environments,omitempty"`
}
