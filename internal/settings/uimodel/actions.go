// Package uimodel prepares UI-oriented values without activating public YAML.
package uimodel

type ActionsPermissionChoice string

type ForkApprovalChoice string

const (
	RequireApprovalForFirstTimeContributorsWhoAreNewToGitHub ForkApprovalChoice = "require_approval_for_first_time_contributors_who_are_new_to_github"
	RequireApprovalForFirstTimeContributors                  ForkApprovalChoice = "require_approval_for_first_time_contributors"
	RequireApprovalForAllExternalContributors                ForkApprovalChoice = "require_approval_for_all_external_contributors"
	DefaultActionsForkApproval                                                  = RequireApprovalForFirstTimeContributors
)

type WorkflowPermissionChoice string

const (
	ReadAndWritePermissions                      WorkflowPermissionChoice = "read_and_write_permissions"
	ReadRepositoryContentsAndPackagesPermissions WorkflowPermissionChoice = "read_repository_contents_and_packages_permissions"
	DefaultActionsWorkflowPermissions                                     = ReadRepositoryContentsAndPackagesPermissions
)

const (
	AllowAllActionsAndReusableWorkflows                    ActionsPermissionChoice = "allow_all_actions_and_reusable_workflows"
	DisableActions                                         ActionsPermissionChoice = "disable_actions"
	AllowOwnerActionsAndReusableWorkflows                  ActionsPermissionChoice = "allow_owner_actions_and_reusable_workflows"
	AllowOwnerAndSelectNonOwnerActionsAndReusableWorkflows ActionsPermissionChoice = "allow_owner_and_select_non_owner_actions_and_reusable_workflows"
	DefaultActionsPermissions                                                      = AllowAllActionsAndReusableWorkflows
)

// Owner in a choice represents the repository owner shown in the live UI label.
type ForkPullRequestWorkflows struct {
	RunWorkflowsFromForkPullRequests                       *bool
	SendWriteTokensToWorkflowsFromForkPullRequests         *bool
	SendSecretsAndVariablesToWorkflowsFromForkPullRequests *bool
	RequireApprovalForForkPullRequestWorkflows             *bool
}

type ActionsGeneral struct {
	ForkPullRequestWorkflows                                   *ForkPullRequestWorkflows
	ApprovalForRunningForkPullRequestWorkflowsFromContributors *ForkApprovalChoice
	CheckWorkflowRunStatusArtifactAndLogRetention              *int
	ActionsPermissions                                         *ActionsPermissionChoice
	RequireActionsToBePinnedToAFullLengthCommitSHA             *bool
	AllowActionsCreatedByGitHub                                *bool
	AllowActionsByMarketplaceVerifiedCreators                  *bool
	AllowOrBlockSpecifiedActionsAndReusableWorkflows           *[]string
	WorkflowPermissions                                        *WorkflowPermissionChoice
	AllowGitHubActionsToCreateAndApprovePullRequests           *bool
}
