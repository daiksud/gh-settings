package acceptance

func exportScenarios() []scenario {
	ordinary := map[string]any{
		"version": "daiksud/gh-settings@v0",
		"repository": map[string]any{
			"about":  map[string]any{"description": "Example repository", "website": "https://example.com", "topics": []any{}},
			"labels": []any{},
			"settings": map[string]any{"general": map[string]any{
				"default_branch": "main", "template_repository": false,
				"features":      map[string]any{"issues": true, "projects": false, "wikis": true, "discussions": false, "sponsorships": false, "pull_requests": false, "issue_permissions": map[string]any{"creation_allowed_by": "all_users"}},
				"pull_requests": map[string]any{"allow_merge_commits": false, "allow_squash_merging": false, "allow_rebase_merging": false, "automatically_delete_head_branches": false, "allow_auto_merge": false, "always_suggest_updating_pull_request_branches": false},
				"commits":       map[string]any{"require_contributors_to_sign_off_on_web_based_commits": false},
			}},
		},
	}
	empty := map[string]any{
		"version": "daiksud/gh-settings@v0",
		"repository": map[string]any{
			"about":  map[string]any{"description": "", "website": "", "topics": []any{}},
			"labels": []any{},
			"settings": map[string]any{"general": map[string]any{
				"default_branch": "main", "template_repository": false,
				"features":      map[string]any{"issues": false, "projects": false, "wikis": false, "discussions": false, "sponsorships": false, "pull_requests": false},
				"pull_requests": map[string]any{"allow_merge_commits": false, "allow_squash_merging": false, "allow_rebase_merging": false, "automatically_delete_head_branches": false, "allow_auto_merge": false, "always_suggest_updating_pull_request_branches": false},
				"commits":       map[string]any{"require_contributors_to_sign_off_on_web_based_commits": false},
			}},
		},
	}
	return []scenario{
		{name: "five-settings", body: `{"default_branch":"main","description":"Example repository","homepage":"https://example.com","has_issues":true,"has_projects":false,"has_wiki":true,"has_discussions":false,"web_commit_signoff_required":false,"is_template":false,"id":123}`, expected: ordinary, gitSetup: setupOrdinaryOrigin},
		{name: "zero-values", body: `{"default_branch":"main","description":"","homepage":"","has_issues":false,"has_projects":false,"has_wiki":false,"has_discussions":false,"web_commit_signoff_required":false,"is_template":false}`, expected: empty, gitSetup: setupOrdinaryOrigin},
		{name: "unset-strings", body: `{"default_branch":"main","description":null,"homepage":null,"has_issues":false,"has_projects":false,"has_wiki":false,"has_discussions":false,"web_commit_signoff_required":false,"is_template":false}`, expected: empty, gitSetup: setupOrdinaryOrigin},
		{name: "origin-override", body: `{"default_branch":"main","description":"Example repository","homepage":"https://example.com","has_issues":true,"has_projects":false,"has_wiki":true,"has_discussions":false,"web_commit_signoff_required":false,"is_template":false}`, expected: ordinary, gitSetup: setupOrdinaryOrigin},
		{name: "origin-scp-without-user", noRead: true, errorText: "origin", gitSetup: setupUserlessSCPOrigin},
		{name: "origin-insteadof", body: `{"default_branch":"main","description":"Example repository","homepage":"https://example.com","has_issues":true,"has_projects":false,"has_wiki":true,"has_discussions":false,"web_commit_signoff_required":false,"is_template":false}`, expected: ordinary, gitSetup: setupRewrittenOrigin},
		{name: "origin-alias", noRead: true, errorText: "origin", gitSetup: setupAliasedOrigin},
		{name: "origin-worktree", body: `{"default_branch":"main","description":"Example repository","homepage":"https://example.com","has_issues":true,"has_projects":false,"has_wiki":true,"has_discussions":false,"web_commit_signoff_required":false,"is_template":false}`, expected: ordinary, gitSetup: setupWorktreeOrigin},
		{name: "read-failure", body: `{"message":"Fixture access denied"}`, status: 403, errorText: "403", gitSetup: setupOrdinaryOrigin},
		{name: "repository-failure", noRead: true, errorText: "origin"},
		{name: "incomplete-response", body: `{"homepage":null,"has_issues":false,"has_projects":false,"has_wiki":false,"has_discussions":false}`, errorText: "missing description", gitSetup: setupOrdinaryOrigin},
		{name: "unsupported-command", args: []string{"unknown"}, noRead: true, errorText: "unsupported", gitSetup: setupOrdinaryOrigin},
	}
}
