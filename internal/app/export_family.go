package app

import (
	"fmt"

	"github.com/daiksud/gh-settings/internal/settings"
)

// exportAcquiredFamily isolates projection validation from retained declarations
// and from other reads, including unavailable core About values.
func exportAcquiredFamily(path string, d settings.Declaration) (settings.Declaration, error) {
	var family settings.Declaration
	switch path {
	case "repository.about.topics":
		family.Topics = d.Topics
	case "repository.settings.collaborators":
		family.Collaborators = d.Collaborators
	case "repository.labels":
		family.Labels = d.Labels
	case "repository.settings.actions.policies":
		family.ActionsPolicies = d.ActionsPolicies
	case settings.ClassicProtectionPath:
		family.BranchProtectionRules = d.BranchProtectionRules
	case "repository.settings.rulesets":
		family.Rulesets = d.Rulesets
	case "repository.settings.environments":
		family.Environments = d.Environments
	case "repository.settings.actions.general.actions_permissions":
		family.Actions.Enabled, family.Actions.AllowedActions, family.Actions.SHAPinningRequired, family.Actions.SelectedActions = d.Actions.Enabled, d.Actions.AllowedActions, d.Actions.SHAPinningRequired, d.Actions.SelectedActions
	case "repository.settings.actions.general.workflow_permissions":
		family.Actions.DefaultWorkflowPermissions, family.Actions.CanApprovePullRequestReviews = d.Actions.DefaultWorkflowPermissions, d.Actions.CanApprovePullRequestReviews
	case "repository.settings.actions.general.artifact_and_log_retention":
		family.Actions.RetentionDays = d.Actions.RetentionDays
	case "repository.settings.actions.general.cache_retention":
		family.Actions.CacheRetentionDays = d.Actions.CacheRetentionDays
	case "repository.settings.actions.general.cache_storage":
		family.Actions.CacheSizeGB = d.Actions.CacheSizeGB
	case "repository.settings.actions.general.approval_for_running_fork_pull_request_workflows_from_contributors":
		family.Actions.ForkApprovalPolicy = d.Actions.ForkApprovalPolicy
	case "repository.settings.copilot.code_review.use_custom_instructions":
		family.CopilotInstructions = d.CopilotInstructions
	case "repository.settings.copilot.code_review.allow_copilot_to_use_mcp_tools_when_reviewing_pull_requests":
		family.CopilotMCP = d.CopilotMCP
	case "repository.settings.copilot.code_review.review_effort_level":
		family.CopilotReviewEffort = d.CopilotReviewEffort
	case "repository.settings.interaction_limits.pull_request_limits.exemption_list":
		family.PullRequestExemptions = d.PullRequestExemptions
	case "repository.settings.interaction_limits.pull_request_limits":
		family.PullRequestLimits = d.PullRequestLimits
	case "repository.settings.general.danger_zone.archive_this_repository":
		family.Repository.Archived = d.Repository.Archived
	case "repository.settings.pages":
		family.Pages = d.Pages
	case "repository.settings.actions.variables":
		family.Variables = d.Variables
	case "repository.settings.agents.variables":
		family.AgentsVariables = d.AgentsVariables
	default:
		return settings.Declaration{}, fmt.Errorf("unknown export family")
	}
	return family, nil
}
