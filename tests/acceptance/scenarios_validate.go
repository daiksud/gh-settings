package acceptance

import "strings"

type validateCase struct {
	name, input, wantError string
}

func validateScenarios() []validateCase {
	return []validateCase{
		{name: "validate-all-fields", input: repositoryInput("about: {description: Example, website: https://example.com}\nsettings: {general: {features: {issues: true, projects: false, wikis: true}}}\n")},
		{name: "validate-partial-zero-values", input: repositoryInput("about: {description: ''}\nsettings: {general: {features: {issues: false}}}\n")},
		{name: "validate-unknown-key", input: generalInput("visibility: private\n"), wantError: "unknown repository.settings.general key"},
		{name: "validate-duplicate-key", input: generalInput("features: {issues: true, issues: false}\n"), wantError: "duplicate key"},
		{name: "validate-alias", input: aboutInput("description: &text Example\nwebsite: *text\n")},
		{name: "validate-merge", input: generalInput("features: {<<: {issues: false}}\n")},
		{name: "validate-wrong-type", input: generalInput("features: {issues: 'false'}\n"), wantError: "must be a boolean"},
		{name: "validate-null-boolean", input: generalInput("features: {issues: null}\n"), wantError: "must not be null"},
		{name: "validate-malformed", input: generalInput("features: {issues: [\n"), wantError: "decode YAML"},
		{name: "validate-missing-version", input: "repository: {}\n", wantError: "missing version"},
		{name: "validate-unsupported-version", input: "version: daiksud/gh-settings@v1\n", wantError: "version must be the string"},
		{name: "validate-obsolete-envelope", input: "schemaVersion: 1\nsettings: {repository: {has_issues: false}}\n", wantError: "missing version"},
		{name: "validate-obsolete-settings-alias", input: "version: daiksud/gh-settings@v0\nsettings: {}\n", wantError: "unknown root key"},
		{name: "validate-legacy-tagged-boolean", input: generalInput("features: {issues: !!bool yes}\n"), wantError: "spelled true or false"},
		{name: "validate-mistagged-mapping", input: "version: daiksud/gh-settings@v0\nrepository: {settings: !!str {}}\n", wantError: "mapping must have YAML map tag"},
		{name: "validate-version-only-noop", input: "version: daiksud/gh-settings@v0\n"},
		{name: "validate-empty-repository-noop", input: "version: daiksud/gh-settings@v0\nrepository: {}\n"},
		{name: "validate-empty-settings-noop", input: "version: daiksud/gh-settings@v0\nrepository: {settings: {}}\n"},
		{name: "validate-missing-file", wantError: "settings.yml"},
	}
}

// These helpers indent literal UI declarations; they do not translate API fixtures.
func repositoryInput(fields string) string {
	return "version: daiksud/gh-settings@v0\nrepository:\n" + indentInput(fields, 2)
}

func aboutInput(fields string) string {
	return repositoryInput("about:\n" + indentInput(fields, 2))
}

func generalInput(fields string) string {
	return repositoryInput("settings:\n  general:\n" + indentInput(fields, 4))
}

func indentInput(fields string, spaces int) string {
	var result strings.Builder
	for _, line := range strings.SplitAfter(fields, "\n") {
		if line != "" {
			result.WriteString(strings.Repeat(" ", spaces))
			result.WriteString(line)
		}
	}
	return result.String()
}
