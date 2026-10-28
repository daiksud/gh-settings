// Package uimodel contains internal preparation for UI-oriented settings.
// The current public YAML codec does not consume these types.
package uimodel

type EnforcementStatus string

const (
	Active                          EnforcementStatus = "active"
	Disabled                        EnforcementStatus = "disabled"
	DefaultRulesetEnforcementStatus                   = Disabled
)

// Ruleset retains selected-ID ownership while naming controls after the UI.
type Ruleset struct {
	ID                *int64
	RulesetName       string
	EnforcementStatus *EnforcementStatus
	TargetBranches    *BranchTargets
	TargetTags        *TagTargets
	BypassList        *[]BypassEntry
	Rules             RulesetRules
}

// RulesetRules describes checkbox states inside a whole owned Ruleset.
// Omission does not retain a remote rule; false and nil both leave it unselected.
type RulesetRules struct {
	RestrictDeletions, BlockForcePushes, RequireLinearHistory, RequireSignedCommits *bool
	RequireAPullRequestBeforeMerging                                                *PullRequestRule
	RequireStatusChecksToPass                                                       *StatusChecksRule
	RequireCodeScanningResults                                                      *CodeScanningRule
	RequireCodeQualityResults                                                       *CodeQualityRule
}

type MergeMethod string

type CodeScanningAlerts string
type CodeScanningSecurityAlerts string
type CodeQualitySeverity string

const (
	DefaultCodeScanningAlerts         CodeScanningAlerts         = "errors"
	DefaultCodeScanningSecurityAlerts CodeScanningSecurityAlerts = "high_or_higher"
	DefaultCodeQualitySeverity        CodeQualitySeverity        = "errors"
)

type CodeScanningTool struct {
	ToolName       string
	SecurityAlerts CodeScanningSecurityAlerts
	Alerts         CodeScanningAlerts
}

type CodeScanningRule struct{ CodeScanningTools *[]CodeScanningTool }
type CodeQualityRule struct{ Severity *CodeQualitySeverity }

type StatusChecksRule struct {
	RequiredStatusChecks                     *[]StatusCheck
	RequireBranchesToBeUpToDateBeforeMerging *bool
	DoNotRequireStatusChecksOnCreation       *bool
}

type StatusCheck struct {
	Check  string
	Source CheckSource
}

// CheckSource retains Any source separately from a selected dynamic App ID.
// Its zero value keeps the existing any-source declaration default.
type CheckSource struct {
	AnySource *bool
	App       *int64
}

const (
	Merge                    MergeMethod = "merge"
	Squash                   MergeMethod = "squash"
	Rebase                   MergeMethod = "rebase"
	DefaultRequiredApprovals             = 0
)

func DefaultRulesetMergeMethods() []MergeMethod { return []MergeMethod{Merge, Squash, Rebase} }

type PullRequestRule struct {
	RequiredApprovals                                             *int
	AllowedMergeMethods                                           *[]MergeMethod
	DismissStalePullRequestApprovalsWhenNewCommitsArePushed       *bool
	RequireReviewFromCodeOwners                                   *bool
	RequireApprovalOfTheMostRecentReviewablePush                  *bool
	RequireConversationResolutionBeforeMerging                    *bool
	RequireAnAdditionalApprovalForUnattributedCopilotPullRequests *bool
	RequireReviewFromSpecificTeams                                *[]TeamReview
}

type TeamReview struct {
	Reviewer     int64
	Approvals    *int
	FilePatterns *[]string
}

const DefaultTeamReviewApprovals = 0

type BypassRole string
type BypassMode string

const (
	RepositoryAdmin     BypassRole = "repository_admin"
	Maintain            BypassRole = "maintain"
	Write               BypassRole = "write"
	Always              BypassMode = "always"
	Exempt              BypassMode = "exempt"
	ForPullRequestsOnly BypassMode = "for_pull_requests_only"
	DefaultBypassMode              = Always
)

// BypassEntry selects exactly one actor. Dynamic actors retain numeric identity.
// DeployKeys is a selected marker and must be true when present.
type BypassEntry struct {
	Role            *BypassRole
	Team, App, User *int64
	DeployKeys      *bool
	BypassMode      *BypassMode
}

type BranchTargets struct {
	IncludeDefaultBranch, IncludeAllBranches *bool
	IncludeByPattern, ExcludeByPattern       *[]string
}

type TagTargets struct {
	IncludeAllTags                     *bool
	IncludeByPattern, ExcludeByPattern *[]string
}
