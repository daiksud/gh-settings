package main

func exportScenarios() []scenario {
	ordinary := map[string]any{"schemaVersion": 1, "settings": map[string]any{"repository": map[string]any{"description": "Example repository", "homepage": "https://example.com", "has_issues": true, "has_projects": false, "has_wiki": true}}}
	empty := map[string]any{"schemaVersion": 1, "settings": map[string]any{"repository": map[string]any{"description": "", "homepage": "", "has_issues": false, "has_projects": false, "has_wiki": false}}}
	return []scenario{
		{name: "five-settings", body: `{"description":"Example repository","homepage":"https://example.com","has_issues":true,"has_projects":false,"has_wiki":true,"id":123}`, expected: ordinary},
		{name: "zero-values", body: `{"description":"","homepage":"","has_issues":false,"has_projects":false,"has_wiki":false}`, expected: empty},
		{name: "unset-strings", body: `{"description":null,"homepage":null,"has_issues":false,"has_projects":false,"has_wiki":false}`, expected: empty},
		{name: "origin-override", body: `{"description":"Example repository","homepage":"https://example.com","has_issues":true,"has_projects":false,"has_wiki":true}`, expected: ordinary},
		{name: "origin-scp-without-user", body: `{"description":"Example repository","homepage":"https://example.com","has_issues":true,"has_projects":false,"has_wiki":true}`, expected: ordinary, userlessOrigin: true},
		{name: "origin-insteadof", body: `{"description":"Example repository","homepage":"https://example.com","has_issues":true,"has_projects":false,"has_wiki":true}`, expected: ordinary, rewriteOrigin: true},
		{name: "origin-alias", body: `{"description":"Example repository","homepage":"https://example.com","has_issues":true,"has_projects":false,"has_wiki":true}`, expected: ordinary, aliasOrigin: true},
		{name: "origin-worktree", body: `{"description":"Example repository","homepage":"https://example.com","has_issues":true,"has_projects":false,"has_wiki":true}`, expected: ordinary, worktreeOrigin: true},
		{name: "read-failure", body: `{"message":"Fixture access denied"}`, status: 403, errorText: "403"},
		{name: "repository-failure", noRepo: true, noRead: true, errorText: "origin"},
		{name: "incomplete-response", body: `{"homepage":null,"has_issues":false,"has_projects":false,"has_wiki":false}`, errorText: "missing description"},
		{name: "output-failure", body: `{"description":"Example repository","homepage":"https://example.com","has_issues":true,"has_projects":false,"has_wiki":true}`, outputFailure: true, errorText: "write output"},
		{name: "unsupported-command", args: []string{"apply"}, noRead: true, errorText: "unsupported"},
	}
}
