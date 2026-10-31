package presentation

import "github.com/daiksud/gh-settings/internal/settings"

// SettingPath renders unambiguous UI paths without changing comparison identity.
// Compound controls and the complete reporting contract are owned by Issue #481.
func SettingPath(path string) string {
	switch path {
	case "repository.description":
		return "repository.about.description"
	case "repository.homepage":
		return "repository.about.website"
	case "topics.names":
		return "repository.about.topics"
	case "repository.is_template":
		return "repository.settings.general.template_repository"
	case "repository.default_branch":
		return "repository.settings.general.default_branch"
	case "repository.has_wiki":
		return "repository.settings.general.features.wikis"
	case "repository.has_issues":
		return "repository.settings.general.features.issues"
	case "repository.has_sponsorships":
		return "repository.settings.general.features.sponsorships"
	case "repository.has_discussions":
		return "repository.settings.general.features.discussions"
	case "repository.has_projects":
		return "repository.settings.general.features.projects"
	case "repository.issue_creation_policy":
		return settings.IssuePermissionsControl.Path()
	case "repository.pull_request_creation_policy":
		return settings.PullRequestPermissionsControl.Path()
	case "repository.has_pull_requests":
		return "repository.settings.general.features.pull_requests"
	case "repository.allow_merge_commit":
		return "repository.settings.general.pull_requests.allow_merge_commits"
	case "repository.allow_squash_merge":
		return "repository.settings.general.pull_requests.allow_squash_merging"
	case "repository.allow_rebase_merge":
		return "repository.settings.general.pull_requests.allow_rebase_merging"
	case "repository.allow_update_branch":
		return "repository.settings.general.pull_requests.always_suggest_updating_pull_request_branches"
	case "repository.allow_auto_merge":
		return "repository.settings.general.pull_requests.allow_auto_merge"
	case "repository.delete_branch_on_merge":
		return "repository.settings.general.pull_requests.automatically_delete_head_branches"
	case "repository.web_commit_signoff_required":
		return "repository.settings.general.commits.require_contributors_to_sign_off_on_web_based_commits"
	case "repository.archived":
		return "repository.settings.general.danger_zone.archive_this_repository"
	case "security.secret_scanning":
		return settings.SecretProtectionControl.Path()
	case "security.secret_scanning_push_protection":
		return settings.PushProtectionControl.Path()
	case "security.private_vulnerability_reporting":
		return "repository.settings.advanced_security.private_vulnerability_reporting"
	case "security.immutable_releases":
		return "repository.settings.general.releases.immutable_release"
	case "pages.build_type":
		return "repository.settings.pages.build_and_deployment.source"
	case "pages.source.branch":
		return "repository.settings.pages.build_and_deployment.branch"
	case "pages.source.path":
		return "repository.settings.pages.build_and_deployment.select_folder"
	case "pages.cname":
		return "repository.settings.pages.custom_domain"
	case "pages.https_enforced":
		return "repository.settings.pages.enforce_https"
	case "actions.fork_pr_approval_policy":
		return "repository.settings.actions.general.approval_for_running_fork_pull_request_workflows_from_contributors"
	case "actions.artifact_and_log_retention_days":
		return "repository.settings.actions.general.check_workflow_run_status_artifact_and_log_retention"
	case "actions.default_workflow_permissions":
		return settings.WorkflowPermissionsPath
	case "actions.can_approve_pull_request_reviews":
		return "repository.settings.actions.general.workflow_permissions.allow_github_actions_to_create_and_approve_pull_requests"
	case "actions.sha_pinning_required":
		return "repository.settings.actions.general.actions_permissions.require_actions_to_be_pinned_to_a_full_length_commit_sha"
	case "actions.selected_actions.github_owned_allowed":
		return "repository.settings.actions.general.actions_permissions.allow_actions_created_by_github"
	case "actions.selected_actions.verified_allowed":
		return "repository.settings.actions.general.actions_permissions.allow_actions_by_marketplace_verified_creators"
	case "actions.selected_actions.patterns_allowed":
		return "repository.settings.actions.general.actions_permissions.allow_or_block_specified_actions_and_reusable_workflows"
	case "actions.private_fork_workflows.run_workflows_from_fork_pull_requests":
		return "repository.settings.actions.general.fork_pull_request_workflows.run_workflows_from_fork_pull_requests"
	case "actions.private_fork_workflows.send_write_tokens_to_workflows":
		return "repository.settings.actions.general.fork_pull_request_workflows.send_write_tokens_to_workflows_from_fork_pull_requests"
	case "actions.private_fork_workflows.send_secrets_and_variables":
		return "repository.settings.actions.general.fork_pull_request_workflows.send_secrets_and_variables_to_workflows_from_fork_pull_requests"
	case "actions.private_fork_workflows.require_approval_for_fork_pr_workflows":
		return "repository.settings.actions.general.fork_pull_request_workflows.require_approval_for_fork_pull_request_workflows"
	default:
		return "settings." + path
	}
}
