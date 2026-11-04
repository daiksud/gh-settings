package githubactions

import (
	"strings"
	"testing"
)

func TestSettingsEnvironmentScopePRFixtures(t *testing.T) {
	const approved = settingsValidationFixture
	for _, tc := range []struct {
		name, content, path string
		valid               bool
	}{
		{"approved", approved, settingsPRWorkflowPath, true},
		{"old trigger", strings.Replace(approved, "  pull_request:", "  pull_request_target:", 1), settingsPRWorkflowPath, false},
		{"other base", strings.Replace(approved, "branches: [main]", "branches: [develop]", 1), settingsPRWorkflowPath, false},
		{"extra branch", strings.Replace(approved, "branches: [main]", "branches: [main, develop]", 1), settingsPRWorkflowPath, false},
		{"issue-comment trigger", strings.Replace(approved, "  pull_request:", "  issue_comment:", 1), settingsPRWorkflowPath, false},
		{"extra trigger", strings.Replace(approved, "on:\n", "on:\n  workflow_dispatch: null\n", 1), settingsPRWorkflowPath, false},
		{"explicit activities", strings.Replace(approved, "    branches: [main]", "    branches: [main]\n    types: [opened]", 1), settingsPRWorkflowPath, false},
		{"other workflow", approved, ".github/workflows/other.yml", false},
		{"extra Environment job", approved + "  extra:\n    environment: gh-settings\n", settingsPRWorkflowPath, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			files := settingsValidationFixtureWorkflows()
			files[tc.path] = []byte(tc.content)
			if err := checkSettingsEnvironmentScope(files); (err == nil) != tc.valid {
				t.Fatalf("valid=%t error=%v", tc.valid, err)
			}
		})
	}
}
