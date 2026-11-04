package githubactions

import "testing"

func TestSettingsEnvironmentScopeReservedNames(t *testing.T) {
	for _, source := range []string{
		"jobs:\n  other:\n    environment: gh-settings-plan\n",
		"jobs:\n  other:\n    environment: {name: ' GH-SETTINGS-PLAN '}\n",
		"jobs:\n  settings-pr:\n    runs-on: ubuntu-slim\n",
		"jobs:\n  other:\n    name: settings-pr\n    runs-on: ubuntu-slim\n",
		"jobs:\n  settings-plan:\n    runs-on: ubuntu-slim\n",
		"jobs:\n  other:\n    name: settings-plan\n    runs-on: ubuntu-slim\n",
	} {
		files := settingsValidationFixtureWorkflows()
		files[".github/workflows/other.yml"] = []byte(source)
		if err := checkSettingsEnvironmentScope(files); err == nil {
			t.Fatalf("retired Environment or reserved check name accepted in another workflow: %s", source)
		}
	}
}
