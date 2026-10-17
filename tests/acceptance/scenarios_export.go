package acceptance

func exportScenarios() []scenario {
	ordinary := map[string]any{"schemaVersion": 1, "settings": map[string]any{"labels": []any{}, "topics": map[string]any{"names": []any{}}, "repository": map[string]any{"description": "Example repository", "homepage": "https://example.com", "has_issues": true, "has_projects": false, "has_wiki": true}}}
	empty := map[string]any{"schemaVersion": 1, "settings": map[string]any{"labels": []any{}, "topics": map[string]any{"names": []any{}}, "repository": map[string]any{"description": "", "homepage": "", "has_issues": false, "has_projects": false, "has_wiki": false}}}
	return []scenario{
		{name: "five-settings", body: `{"description":"Example repository","homepage":"https://example.com","has_issues":true,"has_projects":false,"has_wiki":true,"id":123}`, expected: ordinary, gitSetup: setupOrdinaryOrigin},
		{name: "zero-values", body: `{"description":"","homepage":"","has_issues":false,"has_projects":false,"has_wiki":false}`, expected: empty, gitSetup: setupOrdinaryOrigin},
		{name: "unset-strings", body: `{"description":null,"homepage":null,"has_issues":false,"has_projects":false,"has_wiki":false}`, expected: empty, gitSetup: setupOrdinaryOrigin},
		{name: "origin-override", body: `{"description":"Example repository","homepage":"https://example.com","has_issues":true,"has_projects":false,"has_wiki":true}`, expected: ordinary, gitSetup: setupOrdinaryOrigin},
		{name: "origin-scp-without-user", noRead: true, errorText: "origin", gitSetup: setupUserlessSCPOrigin},
		{name: "origin-insteadof", body: `{"description":"Example repository","homepage":"https://example.com","has_issues":true,"has_projects":false,"has_wiki":true}`, expected: ordinary, gitSetup: setupRewrittenOrigin},
		{name: "origin-alias", noRead: true, errorText: "origin", gitSetup: setupAliasedOrigin},
		{name: "origin-worktree", body: `{"description":"Example repository","homepage":"https://example.com","has_issues":true,"has_projects":false,"has_wiki":true}`, expected: ordinary, gitSetup: setupWorktreeOrigin},
		{name: "read-failure", body: `{"message":"Fixture access denied"}`, status: 403, errorText: "403", gitSetup: setupOrdinaryOrigin},
		{name: "repository-failure", noRead: true, errorText: "origin"},
		{name: "incomplete-response", body: `{"homepage":null,"has_issues":false,"has_projects":false,"has_wiki":false}`, errorText: "missing description", gitSetup: setupOrdinaryOrigin},
		{name: "unsupported-command", args: []string{"unknown"}, noRead: true, errorText: "unsupported", gitSetup: setupOrdinaryOrigin},
	}
}
