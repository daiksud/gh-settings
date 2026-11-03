package githubactions

import (
	"strings"
	"testing"
)

func TestSettingsEnvironmentScopeTrustedPRFixtures(t *testing.T) {
	const approved = settingsValidationFixture
	for _, tc := range []struct {
		name, content, path string
		valid               bool
	}{
		{"approved", approved, settingsPRWorkflowPath, true},
		{"missing activities", strings.Replace(approved, "    types: [opened, synchronize, reopened, edited]\n", "", 1), settingsPRWorkflowPath, false},
		{"ordinary PR", strings.Replace(approved, "  pull_request_target:", "  pull_request:", 1), settingsPRWorkflowPath, false},
		{"other base", strings.Replace(approved, "branches: [main]", "branches: [develop]", 1), settingsPRWorkflowPath, false},
		{"extra branch", strings.Replace(approved, "branches: [main]", "branches: [main, develop]", 1), settingsPRWorkflowPath, false},
		{"issue-comment trigger", strings.Replace(approved, "  pull_request_target:", "  issue_comment:", 1), settingsPRWorkflowPath, false},
		{"extra trigger", strings.Replace(approved, "on:\n", "on:\n  workflow_dispatch: null\n", 1), settingsPRWorkflowPath, false},
		{"closed activity", strings.Replace(approved, "opened, synchronize, reopened", "opened, synchronize, closed", 1), settingsPRWorkflowPath, false},
		{"other workflow", approved, ".github/workflows/other.yml", false},
		{"extra Environment job", approved + "  extra:\n    environment: gh-settings\n", settingsPRWorkflowPath, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if !tc.valid && tc.path == settingsPRWorkflowPath && tc.content == approved {
				t.Fatal("fixture mutation did not match")
			}
			files := settingsValidationFixtureWorkflows()
			files[tc.path] = []byte(tc.content)
			if err := checkSettingsEnvironmentScope(files); (err == nil) != tc.valid {
				t.Fatal(tc.name, err)
			}
		})
	}
}
