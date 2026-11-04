package githubactions

import (
	"fmt"
	"strings"

	"go.yaml.in/yaml/v4"
)

// The default pull_request merge checkout is validated by a pinned, released
// consumer; candidate source runs without privileged credentials.
func checkSettingsValidationWorkflow(workflow map[string]*yaml.Node) error {
	if !settingsScalarIs(workflow["name"], "settings-pr") {
		return fmt.Errorf("settings PR workflow name must match the required check")
	}
	permissions, err := settingsYAMLMap(workflow["permissions"])
	if err != nil || len(permissions) != 0 || workflow["env"] != nil || workflow["defaults"] != nil || workflow["concurrency"] != nil {
		return fmt.Errorf("settings validation must not inject workflow credentials or options")
	}
	jobs, err := settingsYAMLMap(workflow["jobs"])
	if err != nil || len(jobs) != 1 || jobs[settingsValidateCheckName] == nil {
		return fmt.Errorf("settings validation requires exactly one settings-pr job")
	}
	if err := checkSettingsPRTrigger(workflow["on"]); err != nil { return err }
	job, err := settingsYAMLMap(jobs[settingsValidateCheckName])
	if err != nil { return err }
	for _, key := range []string{"environment", "uses", "needs", "env", "container", "services", "strategy", "concurrency", "outputs", "if", "name"} {
		if job[key] != nil { return fmt.Errorf("settings-pr job must not have field %s", key) }
	}
	if job["continue-on-error"] != nil && !settingsScalarIs(job["continue-on-error"], "false") {
		return fmt.Errorf("settings validation must not ignore failure")
	}
	if !settingsScalarIs(job["runs-on"], "ubuntu-slim") || !settingsScalarIs(job["timeout-minutes"], "10") ||
		!settingsStringMapIs(job["permissions"], map[string]string{"contents": "read"}) {
		return fmt.Errorf("settings validation must use ubuntu-slim with Contents read only")
	}
	steps := job["steps"]
	if steps == nil || steps.Kind != yaml.SequenceNode || len(steps.Content) != 2 {
		return fmt.Errorf("settings validation requires default checkout and pinned validator")
	}
	checkout, err := settingsValidationStep(steps.Content[0], "uses")
	if err != nil || !settingsPinnedAction(checkout["uses"], "actions/checkout") {
		return fmt.Errorf("checkout must be pinned with no inputs")
	}
	validate, err := settingsValidationStep(steps.Content[1], "uses", "with")
	if err != nil || !settingsPinnedAction(validate["uses"], "daiksud/gh-settings") ||
		!settingsStringMapIs(validate["with"], map[string]string{"command": "validate"}) {
		return fmt.Errorf("pinned public Action must run validate only")
	}
	return nil
}

func settingsValidationStep(node *yaml.Node, keys ...string) (map[string]*yaml.Node, error) {
	step, err := settingsYAMLMap(node)
	if err != nil { return nil, err }
	allowed := map[string]bool{"name": true}
	for _, key := range keys { allowed[key] = true }
	for key := range step {
		if !allowed[key] { return nil, fmt.Errorf("unsupported validation step field %q", key) }
	}
	return step, nil
}
func settingsScalarIs(node *yaml.Node, value string) bool {
	return node != nil && node.Kind == yaml.ScalarNode && node.Value == value
}
func settingsStringMapIs(node *yaml.Node, expected map[string]string) bool {
	values, err := settingsYAMLMap(node)
	if err != nil || len(values) != len(expected) { return false }
	for key, value := range expected {
		if !settingsScalarIs(values[key], value) { return false }
	}
	return true
}
func settingsPinnedAction(node *yaml.Node, action string) bool {
	if node == nil || node.Kind != yaml.ScalarNode { return false }
	sha, ok := strings.CutPrefix(node.Value, action+"@")
	if !ok || len(sha) != 40 { return false }
	for _, digit := range sha {
		if !strings.ContainsRune("0123456789abcdef", digit) { return false }
	}
	return true
}
