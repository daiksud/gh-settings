package workspace

import (
	"strings"
	"testing"
)

const uiGeneralConfig = `version: daiksud/gh-settings@v0
repository:
    settings:
        general:
            template_repository: false
            default_branch: main
            features:
                wikis: false
                issues: true
                sponsorships: false
                discussions: false
                projects: false
                pull_requests: true
                issue_permissions:
                    creation_allowed_by: collaborators_only
                pull_request_permissions:
                    creation_allowed_by: all_users
            pull_requests:
                allow_merge_commits: true
                allow_squash_merging: true
                allow_rebase_merging: false
                always_suggest_updating_pull_request_branches: false
                allow_auto_merge: false
                automatically_delete_head_branches: false
                merge_commit:
                    default_commit_message: pull_request_title_and_description
                squash_merge:
                    default_commit_message: pull_request_title_and_commit_details
            commits:
                require_contributors_to_sign_off_on_web_based_commits: false
            danger_zone:
                archive_this_repository: false
`

func TestUIConfigGeneralRoundTrip(t *testing.T) {
	got, err := DecodeConfig([]byte(uiGeneralConfig))
	if err != nil {
		t.Fatal(err)
	}
	if got.Settings == nil || got.Settings.General == nil {
		t.Fatal("General settings were omitted")
	}
	g := got.Settings.General
	if g.TemplateRepository == nil || *g.TemplateRepository || g.Features.Wikis == nil || *g.Features.Wikis || g.PullRequests.AllowAutoMerge == nil || *g.PullRequests.AllowAutoMerge || g.PullRequests.MergeCommit.DefaultCommitMessage == nil || *g.PullRequests.MergeCommit.DefaultCommitMessage != "pull_request_title_and_description" {
		t.Fatalf("lost nested General intent: %#v", g)
	}
	encoded, err := EncodeConfig(got)
	if err != nil || string(encoded) != uiGeneralConfig {
		t.Fatalf("encoded %q, error %v; want %q", encoded, err, uiGeneralConfig)
	}
}

func TestUIConfigGeneralOmission(t *testing.T) {
	input := "version: daiksud/gh-settings@v0\nrepository: {settings: {general: {features: {issues: false}}}}\n"
	got, err := DecodeConfig([]byte(input))
	if err != nil || got.Settings == nil || got.Settings.General == nil {
		t.Fatalf("missing General: %#v, %v", got, err)
	}
	g := got.Settings.General
	if g.Features.Issues == nil || *g.Features.Issues || g.DefaultBranch != nil || g.Features.PullRequests != nil || g.Features.IssuePermissions.CreationAllowedBy != nil {
		t.Fatalf("omitted fields supplied intent: %#v", g)
	}
}

func TestUIConfigGeneralRejectsUnknownAndWrongTypes(t *testing.T) {
	for _, fields := range []string{
		"allow_merge_commit: true", "future: true", "features: {has_issues: false}",
		"features: {issues: 'false'}", "features: {issues: yes}",
		"features: {issues: null}", "features: null", "default_branch: 3",
		"pull_requests: {merge_commit: {default_commit_message: false}}",
		"pull_requests: {allow_auto_merge: [false]}",
	} {
		t.Run(fields, func(t *testing.T) {
			input := "version: daiksud/gh-settings@v0\nrepository: {settings: {general: {" + fields + "}}}\n"
			if _, err := DecodeConfig([]byte(input)); err == nil {
				t.Fatal("accepted unknown or mistyped General control")
			}
		})
	}
	if _, err := DecodeConfig([]byte(strings.Replace(uiGeneralConfig, "issues: true", "issues: true\n                issues: false", 1))); err == nil {
		t.Fatal("accepted duplicate nested General key")
	}
}
