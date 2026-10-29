package presentation

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
	default:
		return "settings." + path
	}
}
