package main

func validateScenarios() []scenario {
	return []scenario{
		{name: "validate-all-fields", args: []string{"validate"}, input: repositoryInput("description: Example\nhomepage: https://example.com\nhas_issues: true\nhas_projects: false\nhas_wiki: true\n"), noRead: true, success: true},
		{name: "validate-partial-zero-values", args: []string{"validate"}, input: repositoryInput("description: ''\nhas_issues: false\n"), noRead: true, success: true},
		{name: "validate-unknown-key", args: []string{"validate"}, input: repositoryInput("visibility: private\n"), noRead: true, errorText: "unknown setting"},
		{name: "validate-duplicate-key", args: []string{"validate"}, input: repositoryInput("has_issues: true\nhas_issues: false\n"), noRead: true, errorText: "duplicate key"},
		{name: "validate-alias", args: []string{"validate"}, input: repositoryInput("description: &text Example\nhomepage: *text\n"), noRead: true, success: true},
		{name: "validate-merge", args: []string{"validate"}, input: repositoryInput("<<: {has_issues: false}\n"), noRead: true, success: true},
		{name: "validate-quoted-null-string", args: []string{"validate"}, input: repositoryInput("description: 'null'\n"), noRead: true, success: true},
		{name: "validate-wrong-type", args: []string{"validate"}, input: repositoryInput("has_issues: 'false'\n"), noRead: true, errorText: "must be a boolean"},
		{name: "validate-null-string", args: []string{"validate"}, input: repositoryInput("description: null\n"), noRead: true, errorText: "must be a string"},
		{name: "validate-null-boolean", args: []string{"validate"}, input: repositoryInput("has_issues: null\n"), noRead: true, errorText: "must be a boolean"},
		{name: "validate-malformed", args: []string{"validate"}, input: repositoryInput("has_issues: [\n"), noRead: true, errorText: "decode YAML"},
		{name: "validate-multiple-documents", args: []string{"validate"}, input: "has_issues: true\n---\nhas_wiki: false\n", noRead: true, errorText: "expected one YAML document"},
		{name: "validate-old-flat-format", args: []string{"validate"}, input: "schemaVersion: 1\nhas_issues: false\n", noRead: true, errorText: "unknown root key"},
		{name: "validate-missing-schema-version", args: []string{"validate"}, input: "settings: {}\n", noRead: true, errorText: "missing schemaVersion"},
		{name: "validate-unsupported-schema-version", args: []string{"validate"}, input: "schemaVersion: 2\n", noRead: true, errorText: "unsupported schemaVersion"},
		{name: "validate-null-key", args: []string{"validate"}, input: "schemaVersion: 1\nnull: ignored\n", noRead: true, errorText: "mapping keys must be strings"},
		{name: "validate-alias-duplicate-key", args: []string{"validate"}, input: repositoryInput("&field has_issues: true\n*field: false\n"), noRead: true, errorText: "duplicate key"},
		{name: "validate-alias-merge-key", args: []string{"validate"}, input: repositoryInput("&merge <<: {has_issues: true}\n*merge: {visibility: private}\n"), noRead: true, errorText: "merge key cannot be an alias"},
		{name: "validate-binary-string", args: []string{"validate"}, input: repositoryInput("description: !!binary /w==\n"), noRead: true, errorText: "must be a string"},
		{name: "validate-legacy-tagged-boolean", args: []string{"validate"}, input: repositoryInput("has_issues: !!bool yes\n"), noRead: true, errorText: "spelled true or false"},
		{name: "validate-mistagged-mapping", args: []string{"validate"}, input: "schemaVersion: 1\nsettings: !!str {}\n", noRead: true, errorText: "mapping must have YAML map tag"},
		{name: "validate-mistagged-merge-source", args: []string{"validate"}, input: repositoryInput("<<: !!str {has_issues: true}\n"), noRead: true, errorText: "mapping must have YAML map tag"},
		{name: "validate-missing-file", args: []string{"validate"}, noRead: true, errorText: "settings.yml"},
	}
}
