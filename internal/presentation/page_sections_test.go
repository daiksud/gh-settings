package presentation

import (
	"testing"

	"github.com/daiksud/gh-settings/internal/settings"
)

func TestPageSectionPathsMatchPublicControls(t *testing.T) {
	for field, want := range map[string]string{
		"security.secret_scanning":                      "repository.settings.advanced_security.secret_protection.enabled",
		"security.secret_scanning_push_protection":      "repository.settings.advanced_security.secret_protection.push_protection",
		"security.private_vulnerability_reporting":      "repository.settings.advanced_security.private_vulnerability_reporting",
		"security.immutable_releases":                   "repository.settings.general.releases.immutable_release",
		"actions.default_workflow_permissions":          "repository.settings.actions.general.workflow_permissions.default_permissions",
		"actions.can_approve_pull_request_reviews":      "repository.settings.actions.general.workflow_permissions.allow_github_actions_to_create_and_approve_pull_requests",
		"actions.sha_pinning_required":                  "repository.settings.actions.general.actions_permissions.require_actions_to_be_pinned_to_a_full_length_commit_sha",
		"actions.selected_actions.github_owned_allowed": "repository.settings.actions.general.actions_permissions.allow_actions_created_by_github",
		"actions.selected_actions.verified_allowed":     "repository.settings.actions.general.actions_permissions.allow_actions_by_marketplace_verified_creators",
		"actions.selected_actions.patterns_allowed":     "repository.settings.actions.general.actions_permissions.allow_or_block_specified_actions_and_reusable_workflows",
	} {
		if got := SettingPath(field); got != want {
			t.Errorf("SettingPath(%q) = %q, want %q", field, got, want)
		}
	}
	if settings.ActionsPermissionsPath != "repository.settings.actions.general.actions_permissions.policy" {
		t.Fatal("execution normalization path differs from the public policy control")
	}
}
