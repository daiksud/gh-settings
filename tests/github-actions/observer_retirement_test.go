package githubactions

import (
	"strings"
	"testing"
)

func TestObserverRetirementAllowsOnlyUnprivilegedPRValidation(t *testing.T) {
	const validation = `name: settings-plan
on:
  pull_request:
    types: [opened, synchronize, reopened, edited]
    branches: [main]
permissions: {}
jobs:
  settings-plan:
    if: github.event_name == 'pull_request'
    permissions: {contents: read}
    runs-on: ubuntu-slim
    steps:
      - run: go run . validate
`
	for _, test := range []struct {
		name, source string
		valid        bool
	}{
		{"credential-free replacement", validation, true},
		{"default write grant", strings.Replace(validation, "permissions: {}", "permissions: {contents: write}", 1), false},
		{"job write grant", strings.Replace(validation, "permissions: {contents: read}", "permissions: {contents: write}", 1), false},
		{"extra grant", strings.Replace(validation, "permissions: {contents: read}", "permissions: {contents: read, issues: write}", 1), false},
		{"retired Environment", strings.Replace(validation, "    runs-on:", "    environment: gh-settings-plan\n    runs-on:", 1), false},
		{"other Environment", strings.Replace(validation, "    runs-on:", "    environment: preview\n    runs-on:", 1), false},
		{"missing PR condition", strings.Replace(validation, "    if: github.event_name == 'pull_request'\n", "", 1), false},
		{"extra job", validation + "  other:\n    runs-on: ubuntu-slim\n", false},
		{"reusable job", strings.Replace(validation, "    runs-on:", "    uses: owner/repo/.github/workflows/other.yml@main\n    runs-on:", 1), false},
		{"PR target", strings.Replace(validation, "  pull_request:", "  pull_request_target:", 1), false},
		{"manual dispatch", strings.Replace(validation, "on:\n", "on:\n  workflow_dispatch: {}\n", 1), false},
		{"wrong base", strings.Replace(validation, "branches: [main]", "branches: [dev]", 1), false},
		{"missing activity", strings.Replace(validation, ", edited", "", 1), false},
	} {
		t.Run(test.name, func(t *testing.T) {
			files := map[string][]byte{
				settingsWorkflowPath:      []byte("on: {push: {branches: [main]}}\njobs:\n  apply:\n    environment: gh-settings\n"),
				settingsGuardWorkflowPath: []byte("jobs:\n  settings-env:\n    name: settings-env\n"),
				settingsPlanWorkflowPath:  []byte(test.source),
			}
			if err := checkSettingsEnvironmentScope(files); (err == nil) != test.valid {
				t.Fatalf("valid=%v, guard error=%v", test.valid, err)
			}
		})
	}
	for _, source := range []string{
		"jobs:\n  other:\n    environment: gh-settings-plan\n",
		"jobs:\n  other:\n    environment: {name: ' GH-SETTINGS-PLAN '}\n",
		"jobs:\n  settings-plan:\n    runs-on: ubuntu-slim\n",
		"jobs:\n  other:\n    name: settings-plan\n    runs-on: ubuntu-slim\n",
	} {
		files := map[string][]byte{
			settingsWorkflowPath:          []byte("on: {push: {branches: [main]}}\njobs:\n  apply:\n    environment: gh-settings\n"),
			settingsGuardWorkflowPath:     []byte("jobs:\n  settings-env:\n    name: settings-env\n"),
			settingsPlanWorkflowPath:      []byte(validation),
			".github/workflows/other.yml": []byte(source),
		}
		if err := checkSettingsEnvironmentScope(files); err == nil {
			t.Fatalf("retired Environment or reserved check name accepted in another workflow: %s", source)
		}
	}
}
