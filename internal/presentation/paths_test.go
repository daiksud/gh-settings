package presentation

import "testing"

func TestUnambiguousAboutGeneralAndTopicsDisplayPathInventory(t *testing.T) {
	for _, test := range []struct{ raw, ui string }{
		{"repository.description", "repository.about.description"},
		{"repository.homepage", "repository.about.website"},
		{"topics.names", "repository.about.topics"},
		{"repository.is_template", "repository.settings.general.template_repository"},
		{"repository.default_branch", "repository.settings.general.default_branch"},
		{"repository.has_wiki", "repository.settings.general.features.wikis"},
		{"repository.has_issues", "repository.settings.general.features.issues"},
		{"repository.has_sponsorships", "repository.settings.general.features.sponsorships"},
		{"repository.has_discussions", "repository.settings.general.features.discussions"},
		{"repository.has_projects", "repository.settings.general.features.projects"},
		{"repository.has_pull_requests", "repository.settings.general.features.pull_requests"},
		{"repository.allow_merge_commit", "repository.settings.general.pull_requests.allow_merge_commits"},
		{"repository.allow_squash_merge", "repository.settings.general.pull_requests.allow_squash_merging"},
		{"repository.allow_rebase_merge", "repository.settings.general.pull_requests.allow_rebase_merging"},
		{"repository.allow_update_branch", "repository.settings.general.pull_requests.always_suggest_updating_pull_request_branches"},
		{"repository.allow_auto_merge", "repository.settings.general.pull_requests.allow_auto_merge"},
		{"repository.delete_branch_on_merge", "repository.settings.general.pull_requests.automatically_delete_head_branches"},
		{"repository.web_commit_signoff_required", "repository.settings.general.commits.require_contributors_to_sign_off_on_web_based_commits"},
		{"repository.archived", "repository.settings.general.danger_zone.archive_this_repository"},
	} {
		t.Run(test.raw, func(t *testing.T) {
			if got := SettingPath(test.raw); got != test.ui {
				t.Fatalf("path %q; want %q", got, test.ui)
			}
		})
	}
}

func TestCompoundAndEnumReportAuditRemainsExplicitlySeparate(t *testing.T) {
	// A changed API component cannot independently establish the complete UI
	// menu/visibility state. Issue #481 owns that reporting context and schema.
	for _, raw := range []string{
		"repository.merge_commit_title", "repository.merge_commit_message",
		"repository.squash_merge_commit_title", "repository.squash_merge_commit_message",
		"repository.issue_creation_policy", "repository.pull_request_creation_policy",
	} {
		if got := SettingPath(raw); got != "settings."+raw {
			t.Fatalf("invented compound UI path %q for %s", got, raw)
		}
	}
}
