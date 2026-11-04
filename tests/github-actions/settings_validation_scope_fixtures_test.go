package githubactions

import (
	"strings"
	"testing"
)

const settingsValidationFixture = `name: settings-pr
on:
  pull_request:
    branches: [main]
permissions: {}
jobs:
  settings-pr:
    runs-on: ubuntu-slim
    timeout-minutes: 10
    permissions: {contents: read}
    steps:
      - uses: actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1
      - uses: daiksud/gh-settings@6dbdbb3ab11cbc0ef257fef089d6915d31d003d6
        with: {command: validate}
`

func settingsValidationFixtureWorkflows() map[string][]byte {
	return map[string][]byte{
		settingsWorkflowPath: []byte("on: {workflow_dispatch: null}\njobs:\n  settings-apply:\n    environment: gh-settings\n"),
		settingsPRWorkflowPath: []byte(settingsValidationFixture),
	}
}

func TestSettingsEnvironmentScopeRequiresOneValidator(t *testing.T) {
	files := settingsValidationFixtureWorkflows()
	if err := checkSettingsEnvironmentScope(files); err != nil { t.Fatal(err) }
	delete(files, settingsPRWorkflowPath)
	if err := checkSettingsEnvironmentScope(files); err == nil { t.Fatal("missing validator accepted") }
	files = settingsValidationFixtureWorkflows()
	files[settingsPRWorkflowPath] = []byte(strings.Replace(settingsValidationFixture, "  settings-pr:\n", "  other:\n", 1))
	if err := checkSettingsEnvironmentScope(files); err == nil { t.Fatal("missing required job accepted") }
	files = settingsValidationFixtureWorkflows()
	files[retiredSettingsPlanPath] = []byte("jobs: {}\n")
	if err := checkSettingsEnvironmentScope(files); err == nil { t.Fatal("retired workflow accepted") }
	files = settingsValidationFixtureWorkflows()
	files[settingsPRWorkflowPath] = []byte(strings.ReplaceAll(settingsValidationFixture, "6dbdbb3ab11cbc0ef257fef089d6915d31d003d6", strings.Repeat("a", 40)))
	if err := checkSettingsEnvironmentScope(files); err != nil { t.Fatalf("another immutable pin must be allowed: %v", err) }
}

func TestSettingsValidationRequiredBoundaries(t *testing.T) {
	for _, change := range []struct{ name, before, after string }{
		{"missing permissions", "permissions: {}\n", ""},
		{"default write", "permissions: {}", "permissions: {contents: write}"},
		{"workflow secret", "permissions: {}", "permissions: {}\nenv: {GH_TOKEN: unsafe}"},
		{"wrong workflow name", "name: settings-pr", "name: other"},
		{"job write", "permissions: {contents: read}", "permissions: {contents: write}"},
		{"extra grant", "permissions: {contents: read}", "permissions: {contents: read, issues: write}"},
		{"protected Environment", "    runs-on:", "    environment: gh-settings\n    runs-on:"},
		{"other Environment", "    runs-on:", "    environment: preview\n    runs-on:"},
		{"job secret", "    runs-on:", "    env: {GH_TOKEN: unsafe}\n    runs-on:"},
		{"conditional job", "  settings-pr:\n", "  settings-pr:\n    if: false\n"},
		{"job outputs", "    steps:\n", "    outputs: {success: true}\n    steps:\n"},
		{"ignored failure", "    steps:\n", "    continue-on-error: true\n    steps:\n"},
		{"wrong runner", "runs-on: ubuntu-slim", "runs-on: ubuntu-24.04"},
		{"wrong timeout", "timeout-minutes: 10", "timeout-minutes: 60"},
		{"workflow concurrency", "permissions: {}", "permissions: {}\nconcurrency: shared"},
		{"job concurrency", "    steps:\n", "    concurrency: {group: shared}\n    steps:\n"},
		{"wrong job", "  settings-pr:\n", "  other:\n"},
		{"mutable validator", "daiksud/gh-settings@6dbdbb3ab11cbc0ef257fef089d6915d31d003d6", "daiksud/gh-settings@main"},
		{"wrong validator", "daiksud/gh-settings@", "other/validator@"},
		{"plan instead of validate", "with: {command: validate}", "with: {command: plan}"},
		{"token input", "with: {command: validate}", "with: {command: validate, token: unsafe}"},
		{"conditional validator", "        with: {command: validate}", "        if: false\n        with: {command: validate}"},
		{"ignored validator", "        with: {command: validate}", "        continue-on-error: true\n        with: {command: validate}"},
		{"checkout with inputs", "      - uses: actions/checkout@", "      - with: {ref: main}\n        uses: actions/checkout@"},
		{"extra step", "    steps:\n", "    steps:\n      - run: echo bypass\n"},
	} {
		t.Run(change.name, func(t *testing.T) {
			if !strings.Contains(settingsValidationFixture, change.before) { t.Fatal("fixture mutation did not match") }
			files := settingsValidationFixtureWorkflows()
			files[settingsPRWorkflowPath] = []byte(strings.Replace(settingsValidationFixture, change.before, change.after, 1))
			if err := checkSettingsEnvironmentScope(files); err == nil { t.Fatal("unsafe validation accepted") }
		})
	}
}

func TestSettingsValidationTriggerBoundaries(t *testing.T) {
	for _, change := range []struct{ name, before, after string }{
		{"path filter", "    branches: [main]", "    branches: [main]\n    paths: ['.github/settings.yml']"},
		{"types override", "    branches: [main]", "    branches: [main]\n    types: [opened]"},
		{"wrong event", "  pull_request:", "  pull_request_target:"},
		{"wrong branch", "branches: [main]", "branches: [develop]"},
		{"extra branch", "branches: [main]", "branches: [main, develop]"},
		{"extra event", "on:\n", "on:\n  workflow_dispatch: null\n"},
		{"missing event", "on:\n  pull_request:\n    branches: [main]\n", "on:\n"},
	} {
		t.Run(change.name, func(t *testing.T) {
			if !strings.Contains(settingsValidationFixture, change.before) { t.Fatal("fixture mutation did not match") }
			files := settingsValidationFixtureWorkflows()
			files[settingsPRWorkflowPath] = []byte(strings.Replace(settingsValidationFixture, change.before, change.after, 1))
			if err := checkSettingsEnvironmentScope(files); err == nil { t.Fatal("invalid PR trigger accepted") }
		})
	}
}
