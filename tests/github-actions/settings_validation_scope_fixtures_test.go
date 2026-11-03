package githubactions

import (
	"strings"
	"testing"
)

// The inline body is a placeholder: this guard owns the event/permission/input
// boundary, while the workflow's behavior fixtures and review own its Python.
const settingsValidationFixture = `name: Settings PR
on:
  pull_request_target:
    types: [opened, synchronize, reopened, edited]
    branches: [main]
permissions: {}
defaults:
  run: {shell: bash}
jobs:
  settings-validate:
    permissions: {contents: read}
    runs-on: ubuntu-24.04
    concurrency:
      group: settings-pr-validation-${{ github.repository }}-${{ github.event.pull_request.number }}
      cancel-in-progress: false
    steps:
      - uses: actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1
        with:
          ref: ${{ format('refs/pull/{0}/merge', github.event.pull_request.number) }}
          fetch-depth: 2
          persist-credentials: false
          allow-unsafe-pr-checkout: true
      - id: candidate
        env:
          EXPECTED_HEAD: ${{ github.event.pull_request.head.sha }}
          EXPECTED_BASE: ${{ github.event.pull_request.base.sha }}
        run: python3 -I -c 'pass'
      - uses: daiksud/gh-settings@6dbdbb3ab11cbc0ef257fef089d6915d31d003d6
        with: {command: validate}
`

func settingsValidationFixtureWorkflows() map[string][]byte {
	return map[string][]byte{
		settingsWorkflowPath:   []byte("on: {workflow_dispatch: null}\njobs:\n  apply:\n    environment: gh-settings\n"),
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
		t.Fatal("the required PR validation workflow must exist")
	}
	files = settingsValidationFixtureWorkflows()
	start := strings.Index(settingsValidationFixture, "  settings-validate:\n")
	files[settingsPRWorkflowPath] = []byte(settingsValidationFixture[:start] + "  other: {}\n")
	if err := checkSettingsEnvironmentScope(files); err == nil {
		t.Fatal("a workflow without the required validator must fail")
	}
	files = settingsValidationFixtureWorkflows()
	files[retiredSettingsPlanPath] = []byte("jobs: {}\n")
	if err := checkSettingsEnvironmentScope(files); err == nil {
		t.Fatal("the retired validation workflow must not return")
	}
	files = settingsValidationFixtureWorkflows()
	files[settingsPRWorkflowPath] = []byte(strings.ReplaceAll(settingsValidationFixture, "6dbdbb3ab11cbc0ef257fef089d6915d31d003d6", strings.Repeat("a", 40)))
	if err := checkSettingsEnvironmentScope(files); err != nil {
		t.Fatalf("immutable validation pins are not tied to a particular release: %v", err)
	}
}

func TestSettingsEnvironmentScopeRequiredValidationBoundaries(t *testing.T) {
	for _, change := range []struct{ name, before, after string }{
		{"missing default permissions", "permissions: {}\n", ""},
		{"default write grant", "permissions: {}", "permissions: {contents: write}"},
		{"global secret", "permissions: {}", "permissions: {}\nenv: {GH_TOKEN: '${{ secrets.TOKEN }}'}"},
		{"job write grant", "permissions: {contents: read}", "permissions: {contents: write}"},
		{"extra grant", "permissions: {contents: read}", "permissions: {contents: read, issues: write}"},
		{"Environment", "    runs-on:", "    environment: gh-settings\n    runs-on:"},
		{"other Environment", "    runs-on:", "    environment: preview\n    runs-on:"},
		{"job secret", "    runs-on:", "    env: {GH_TOKEN: '${{ secrets.TOKEN }}'}\n    runs-on:"},
		{"ignored failure", "    runs-on:", "    continue-on-error: true\n    runs-on:"},
		{"skipped dependency", "    runs-on:", "    needs: missing\n    runs-on:"},
		{"delegated validation", "    runs-on:", "    uses: owner/repo/.github/workflows/check.yml@main\n    runs-on:"},
		{"matrix substitute", "    runs-on:", "    strategy: {matrix: {include: []}}\n    runs-on:"},
		{"wrong check name", "  settings-validate:\n", "  settings-validate:\n    name: other-check\n"},
		{"mutable validator", "daiksud/gh-settings@6dbdbb3ab11cbc0ef257fef089d6915d31d003d6", "daiksud/gh-settings@main"},
		{"different action", "daiksud/gh-settings@", "other/validator@"},
		{"remote command", "with: {command: validate}", "with: {command: plan}"},
		{"token input", "with: {command: validate}", "with: {command: validate, token: '${{ github.token }}'}"},
		{"conditional validator", "        with: {command: validate}", "        if: false\n        with: {command: validate}"},
		{"ignored validator failure", "        with: {command: validate}", "        continue-on-error: true\n        with: {command: validate}"},
		{"persisted credentials", "persist-credentials: false", "persist-credentials: true"},
		{"extra step", "    steps:\n", "    steps:\n      - run: echo substitute\n"},
	} {
		t.Run(change.name, func(t *testing.T) {
			if !strings.Contains(settingsValidationFixture, change.before) {
				t.Fatal("fixture mutation did not match")
			}
			files := settingsValidationFixtureWorkflows()
			files[settingsPRWorkflowPath] = []byte(strings.Replace(settingsValidationFixture, change.before, change.after, 1))
			if err := checkSettingsEnvironmentScope(files); err == nil {
				t.Fatal("invalid validation boundary accepted")
			}
		})
	}
}

func TestSettingsEnvironmentScopeValidationBoundaries(t *testing.T) {
	for _, change := range []struct{ name, before, after string }{
		{"path-filtered required check", "    branches: [main]", "    branches: [main]\n    paths: ['.github/settings.yml']"},
		{"missing edit activity", ", edited", ""},
		{"mixed event", "on:\n", "on:\n  pull_request: {}\n"},
		{"conditional required check", "  settings-validate:\n", "  settings-validate:\n    if: false\n"},
		{"main instead of merge", "ref: ${{ format('refs/pull/{0}/merge', github.event.pull_request.number) }}", "ref: ${{ github.sha }}"},
		{"head instead of merge", "refs/pull/{0}/merge", "refs/pull/{0}/head"},
		{"missing parents", "fetch-depth: 2", "fetch-depth: 1"},
		{"missing base identity", "          EXPECTED_BASE: ${{ github.event.pull_request.base.sha }}\n", ""},
		{"wrong head identity", "EXPECTED_HEAD: ${{ github.event.pull_request.head.sha }}", "EXPECTED_HEAD: ${{ github.sha }}"},
		{"conditional identity", "      - id: candidate\n", "      - id: candidate\n        if: false\n"},
		{"injected expression", "run: python3 -I -c 'pass'", "run: echo '${{ github.event.pull_request.title }}'"},
		{"missing metadata body", "        run: python3 -I -c 'pass'\n", ""},
		{"extra plan job", "jobs:\n", "jobs:\n  plan: {environment: gh-settings}\n"},
		{"unexpected output", "    steps:\n", "    outputs: {plan-required: true}\n    steps:\n"},
		{"shared workflow concurrency", "permissions: {}", "permissions: {}\nconcurrency: shared"},
		{"cancelling validation", "cancel-in-progress: false", "cancel-in-progress: true"},
		{"shared validation group", "settings-pr-validation-${{ github.repository }}", "settings-pr-${{ github.repository }}"},
	} {
		t.Run(change.name, func(t *testing.T) {
			if !strings.Contains(settingsValidationFixture, change.before) {
				t.Fatal("fixture mutation did not match")
			}
			files := settingsValidationFixtureWorkflows()
			files[settingsPRWorkflowPath] = []byte(strings.Replace(settingsValidationFixture, change.before, change.after, 1))
			if err := checkSettingsEnvironmentScope(files); err == nil {
				t.Fatal("invalid validation boundary accepted")
			}
		})
	}
}
