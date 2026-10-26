package githubactions

import (
	"strings"
	"testing"
)

func TestSettingsEnvironmentScopeTrustedPRFixtures(t *testing.T) {
	const trigger = "on: {pull_request_target: {branches: [main], types: [opened, synchronize, reopened]}}\n"
	const job = "jobs:\n  plan:\n    environment: gh-settings\n    if: "
	approved := trigger + job + settingsPRCondition + "\n"
	for _, tc := range []struct {
		name, content, path string
		valid               bool
	}{
		{"approved", approved, settingsPRWorkflowPath, true},
		{"default activities", strings.Replace(approved, ", types: [opened, synchronize, reopened]", "", 1), settingsPRWorkflowPath, true},
		{"ordinary PR", strings.Replace(approved, "on: {pull_request_target:", "on: {pull_request:", 1), settingsPRWorkflowPath, false},
		{"other base", strings.Replace(approved, "branches: [main]", "branches: [develop]", 1), settingsPRWorkflowPath, false},
		{"extra branch", strings.Replace(approved, "branches: [main]", "branches: [main, develop]", 1), settingsPRWorkflowPath, false},
		{"issue-comment trigger", strings.Replace(approved, "on: {pull_request_target:", "on: {issue_comment:", 1), settingsPRWorkflowPath, false},
		{"extra trigger", strings.Replace(approved, "on: {pull_request_target:", "on: {workflow_dispatch: null, pull_request_target:", 1), settingsPRWorkflowPath, false},
		{"closed activity", strings.Replace(approved, "opened, synchronize, reopened", "opened, synchronize, closed", 1), settingsPRWorkflowPath, false},
		{"missing main condition", strings.Replace(approved, settingsPRCondition, "github.event_name == 'pull_request_target'", 1), settingsPRWorkflowPath, false},
		{"wrong repository condition", strings.Replace(approved, "github.repository == 'daiksud/gh-settings'", "github.repository == 'other/repo'", 1), settingsPRWorkflowPath, false},
		{"wrong job", strings.Replace(approved, "  plan:", "  apply:", 1), settingsPRWorkflowPath, false},
		{"other workflow", approved, ".github/workflows/other.yml", false},
		{"extra Environment job", approved + "  extra:\n    environment: gh-settings\n", settingsPRWorkflowPath, false},
		{"case variation", strings.Replace(approved, "environment: gh-settings", "environment: GH-SETTINGS", 1), settingsPRWorkflowPath, false},
		{"dynamic name", strings.Replace(approved, "environment: gh-settings", "environment: ${{ vars.ENV }}", 1), settingsPRWorkflowPath, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			files := map[string][]byte{
				settingsWorkflowPath:      []byte("on: {push: {branches: [main]}}\njobs:\n  apply:\n    environment: gh-settings\n"),
				settingsGuardWorkflowPath: []byte("jobs:\n  gh-settings:\n    name: gh-settings\n"),
				tc.path:                   []byte(tc.content),
			}
			if err := checkSettingsEnvironmentScope(files); (err == nil) != tc.valid {
				t.Fatal(tc.name, err)
			}
		})
	}
}
