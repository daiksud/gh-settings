package githubactions

import (
	"strings"
	"testing"
)

const settingsValidationFixture = `name: settings-pr
on:
  pull_request:
    branches: [main]
    types: [opened, synchronize, reopened, edited]
permissions: {}
jobs:
  validate:
    name: ${{ (github.event.action != 'edited' || github.event.changes.base) && 'settings-pr' || 'settings-metadata' }}
    if: github.event.action != 'edited' || github.event.changes.base
    runs-on: ubuntu-slim
    timeout-minutes: 10
    permissions: {contents: read}
    steps:
      - uses: actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1
        with: {persist-credentials: false}
      - uses: daiksud/gh-settings@998576e7326d1075406b84e4988cf728108772f2
        with: {command: validate}
`

func settingsValidationFixtureWorkflows() map[string][]byte {
	return map[string][]byte{
		settingsWorkflowPath:   []byte("on: {workflow_dispatch: null}\njobs:\n  settings-apply:\n    environment: gh-settings\n"),
		settingsPRWorkflowPath: []byte(settingsValidationFixture),
	}
}

func TestSettingsEnvironmentScopeRequiresOneValidator(t *testing.T) {
	files := settingsValidationFixtureWorkflows()
	if err := checkSettingsEnvironmentScope(files); err != nil {
		t.Fatal(err)
	}
	delete(files, settingsPRWorkflowPath)
	if err := checkSettingsEnvironmentScope(files); err == nil {
		t.Fatal("missing validator accepted")
	}
	files = settingsValidationFixtureWorkflows()
	files[settingsPRWorkflowPath] = []byte(strings.Replace(settingsValidationFixture, "  validate:\n", "  other:\n", 1))
	if err := checkSettingsEnvironmentScope(files); err == nil {
		t.Fatal("missing required job accepted")
	}
	files = settingsValidationFixtureWorkflows()
	files[retiredSettingsPlanPath] = []byte("jobs: {}\n")
	if err := checkSettingsEnvironmentScope(files); err == nil {
		t.Fatal("retired workflow accepted")
	}
	files = settingsValidationFixtureWorkflows()
	files[settingsPRWorkflowPath] = []byte(strings.ReplaceAll(settingsValidationFixture, "998576e7326d1075406b84e4988cf728108772f2", strings.Repeat("a", 40)))
	if err := checkSettingsEnvironmentScope(files); err != nil {
		t.Fatalf("another immutable pin must be allowed: %v", err)
	}
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
		{"suppressed validator", "if: " + candidateEventCondition, "if: false"},
		{"metadata claims required check", validationCheckName, "settings-pr"},
		{"job outputs", "    steps:\n", "    outputs: {success: true}\n    steps:\n"},
		{"ignored failure", "    steps:\n", "    continue-on-error: true\n    steps:\n"},
		{"wrong runner", "runs-on: ubuntu-slim", "runs-on: ubuntu-24.04"},
		{"wrong timeout", "timeout-minutes: 10", "timeout-minutes: 60"},
		{"workflow concurrency", "permissions: {}", "permissions: {}\nconcurrency: shared"},
		{"job concurrency", "    steps:\n", "    concurrency: {group: shared}\n    steps:\n"},
		{"wrong job", "  validate:\n", "  other:\n"},
		{"mutable validator", "daiksud/gh-settings@998576e7326d1075406b84e4988cf728108772f2", "daiksud/gh-settings@main"},
		{"wrong validator", "daiksud/gh-settings@", "other/validator@"},
		{"plan instead of validate", "with: {command: validate}", "with: {command: plan}"},
		{"token input", "with: {command: validate}", "with: {command: validate, token: unsafe}"},
		{"conditional validator", "        with: {command: validate}", "        if: false\n        with: {command: validate}"},
		{"ignored validator", "        with: {command: validate}", "        continue-on-error: true\n        with: {command: validate}"},
		{"base checkout", "persist-credentials: false}", "persist-credentials: false, ref: main}"},
		{"persisted credentials", "persist-credentials: false", "persist-credentials: true"},
		{"extra step", "    steps:\n", "    steps:\n      - run: echo bypass\n"},
	} {
		t.Run(change.name, func(t *testing.T) {
			if !strings.Contains(settingsValidationFixture, change.before) {
				t.Fatal("fixture mutation did not match")
			}
			files := settingsValidationFixtureWorkflows()
			files[settingsPRWorkflowPath] = []byte(strings.Replace(settingsValidationFixture, change.before, change.after, 1))
			if err := checkSettingsEnvironmentScope(files); err == nil {
				t.Fatal("unsafe validation accepted")
			}
		})
	}
}

func TestSettingsValidationTriggerBoundaries(t *testing.T) {
	for _, change := range []struct{ name, before, after string }{
		{"path filter", "    branches: [main]", "    branches: [main]\n    paths: ['.github/settings.yml']"},
		{"missing retarget event", "types: [opened, synchronize, reopened, edited]", "types: [opened, synchronize, reopened]"},
		{"wrong event", "  pull_request:", "  pull_request_target:"},
		{"wrong branch", "branches: [main]", "branches: [develop]"},
		{"extra branch", "branches: [main]", "branches: [main, develop]"},
		{"extra event", "on:\n", "on:\n  workflow_dispatch: null\n"},
		{"missing event", "on:\n  pull_request:\n    branches: [main]\n", "on:\n"},
	} {
		t.Run(change.name, func(t *testing.T) {
			if !strings.Contains(settingsValidationFixture, change.before) {
				t.Fatal("fixture mutation did not match")
			}
			files := settingsValidationFixtureWorkflows()
			files[settingsPRWorkflowPath] = []byte(strings.Replace(settingsValidationFixture, change.before, change.after, 1))
			if err := checkSettingsEnvironmentScope(files); err == nil {
				t.Fatal("invalid PR trigger accepted")
			}
		})
	}
}
