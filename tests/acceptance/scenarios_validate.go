package main

import "strings"

type validateCase struct {
	name, input, wantError string
}

func validateScenarios() []validateCase {
	return []validateCase{
		{name: "validate-all-fields", input: repositoryInput("description: Example\nhomepage: https://example.com\nhas_issues: true\nhas_projects: false\nhas_wiki: true\n")},
		{name: "validate-partial-zero-values", input: repositoryInput("description: ''\nhas_issues: false\n")},
		{name: "validate-unknown-key", input: repositoryInput("visibility: private\n"), wantError: "unknown setting"},
		{name: "validate-duplicate-key", input: repositoryInput("has_issues: true\nhas_issues: false\n"), wantError: "duplicate key"},
		{name: "validate-alias", input: repositoryInput("description: &text Example\nhomepage: *text\n")},
		{name: "validate-merge", input: repositoryInput("<<: {has_issues: false}\n")},
		{name: "validate-wrong-type", input: repositoryInput("has_issues: 'false'\n"), wantError: "must be a boolean"},
		{name: "validate-null-boolean", input: repositoryInput("has_issues: null\n"), wantError: "must be a boolean"},
		{name: "validate-malformed", input: repositoryInput("has_issues: [\n"), wantError: "decode YAML"},
		{name: "validate-missing-schema-version", input: "settings: {}\n", wantError: "missing schemaVersion"},
		{name: "validate-unsupported-schema-version", input: "schemaVersion: 2\n", wantError: "unsupported schemaVersion"},
		{name: "validate-legacy-tagged-boolean", input: repositoryInput("has_issues: !!bool yes\n"), wantError: "spelled true or false"},
		{name: "validate-mistagged-mapping", input: "schemaVersion: 1\nsettings: !!str {}\n", wantError: "mapping must have YAML map tag"},
		{name: "validate-missing-file", wantError: "settings.yml"},
	}
}

func repositoryInput(fields string) string {
	var result strings.Builder
	result.WriteString("schemaVersion: 1\nsettings:\n  repository:\n")
	for _, line := range strings.SplitAfter(fields, "\n") {
		if line != "" {
			result.WriteString("    ")
			result.WriteString(line)
		}
	}
	return result.String()
}
