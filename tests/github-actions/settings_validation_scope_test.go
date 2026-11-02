package githubactions

import (
	"fmt"
	"strings"

	"go.yaml.in/yaml/v4"
)

// Keep the required check an unconditional, credential-free invocation of the
// released validator. The trusted inline metadata step's implementation, like
// the privileged plan steps, remains subject to owner review and behavior tests.
func checkSettingsValidationWorkflow(workflow map[string]*yaml.Node) error {
	permissions, err := settingsYAMLMap(workflow["permissions"])
	if err != nil || len(permissions) != 0 || workflow["env"] != nil {
		return fmt.Errorf("settings validation must have no default token grants or injected environment")
	}
	jobs, err := settingsYAMLMap(workflow["jobs"])
	if err != nil || jobs[settingsPlanCheckName] == nil {
		return fmt.Errorf("settings validation requires the settings-plan job")
	}
	if len(jobs) != 2 || jobs["plan"] == nil || workflow["concurrency"] != nil {
		return fmt.Errorf("PR workflow requires separate validation and plan jobs with independent concurrency")
	}
	if err := checkSettingsPRTrigger(workflow["on"]); err != nil {
		return err
	}
	defaults, err := settingsYAMLMap(workflow["defaults"])
	if err != nil || len(defaults) != 1 || !settingsStringMapIs(defaults["run"], map[string]string{"shell": "bash"}) {
		return fmt.Errorf("settings workflow must use the trusted bash step shell")
	}
	plan, err := settingsYAMLMap(jobs["plan"])
	if err != nil || !settingsScalarIs(plan["environment"], settingsEnvironment) || !settingsConcurrencyIs(plan["concurrency"], "settings-pr-${{ github.repository }}-${{ github.event.pull_request.number }}", true) {
		return fmt.Errorf("plan must retain its Environment and independent cancelling concurrency")
	}
	job, err := settingsYAMLMap(jobs[settingsPlanCheckName])
	if err != nil {
		return err
	}
	for _, key := range []string{"environment", "uses", "needs", "env", "container", "services", "strategy"} {
		if job[key] != nil {
			return fmt.Errorf("settings validation cannot set jobs.settings-plan.%s", key)
		}
	}
	if job["continue-on-error"] != nil && !settingsScalarIs(job["continue-on-error"], "false") {
		return fmt.Errorf("settings validation cannot ignore failure")
	}
	if !settingsScalarIs(job["runs-on"], "ubuntu-24.04") || !settingsStringMapIs(job["permissions"], map[string]string{"contents": "read"}) {
		return fmt.Errorf("settings validation requires the supported runner and only Contents read")
	}
	if job["if"] != nil || !settingsStringMapIs(job["outputs"], map[string]string{"plan-required": "${{ steps.candidate.outputs.plan-required }}"}) || !settingsConcurrencyIs(job["concurrency"], "settings-pr-validation-${{ github.repository }}-${{ github.event.pull_request.number }}", false) {
		return fmt.Errorf("validation must run unconditionally and expose only plan eligibility with independent non-cancelling concurrency")
	}
	steps := job["steps"]
	if steps == nil || steps.Kind != yaml.SequenceNode || len(steps.Content) != 3 {
		return fmt.Errorf("settings validation requires merge checkout, trusted metadata, and validate")
	}
	checkout, err := settingsValidationStep(steps.Content[0], "uses", "with")
	if err != nil || !settingsPinnedAction(checkout["uses"], "actions/checkout") || !settingsStringMapIs(checkout["with"], map[string]string{
		"ref": "${{ format('refs/pull/{0}/merge', github.event.pull_request.number) }}", "persist-credentials": "false",
		"fetch-depth": "2", "allow-unsafe-pr-checkout": "true",
	}) {
		return fmt.Errorf("settings validation must check out the PR merge without persisting credentials")
	}
	candidate, err := settingsValidationStep(steps.Content[1], "id", "env", "run")
	if err != nil || !settingsScalarIs(candidate["id"], "candidate") || !settingsStringMapIs(candidate["env"], map[string]string{
		"EXPECTED_HEAD": "${{ github.event.pull_request.head.sha }}",
		"EXPECTED_BASE": "${{ github.event.pull_request.base.sha }}",
	}) || candidate["run"] == nil || candidate["run"].Kind != yaml.ScalarNode || strings.TrimSpace(candidate["run"].Value) == "" || strings.Contains(candidate["run"].Value, "${{") {
		return fmt.Errorf("validation requires trusted inline metadata handling with explicit event head and base inputs")
	}
	validate, err := settingsValidationStep(steps.Content[2], "uses", "with")
	if err != nil || !settingsPinnedAction(validate["uses"], "daiksud/gh-settings") || !settingsStringMapIs(validate["with"], map[string]string{"command": "validate"}) {
		return fmt.Errorf("settings validation must invoke the immutable public Action with only command: validate")
	}
	return nil
}

// These steps cannot be conditional, continue on error, inject tokens, or add
// extra actions around the approved validation path.
func settingsValidationStep(node *yaml.Node, keys ...string) (map[string]*yaml.Node, error) {
	step, err := settingsYAMLMap(node)
	if err != nil {
		return nil, err
	}
	allowed := map[string]bool{"name": true}
	for _, key := range keys {
		allowed[key] = true
	}
	for key := range step {
		if !allowed[key] {
			return nil, fmt.Errorf("unsupported validation step field %q", key)
		}
	}
	return step, nil
}

func settingsScalarIs(node *yaml.Node, value string) bool {
	return node != nil && node.Kind == yaml.ScalarNode && node.Value == value
}

func settingsStringMapIs(node *yaml.Node, expected map[string]string) bool {
	values, err := settingsYAMLMap(node)
	if err != nil || len(values) != len(expected) {
		return false
	}
	for key, value := range expected {
		if !settingsScalarIs(values[key], value) {
			return false
		}
	}
	return true
}

func settingsConcurrencyIs(node *yaml.Node, group string, cancel bool) bool {
	return settingsStringMapIs(node, map[string]string{"group": group, "cancel-in-progress": fmt.Sprint(cancel)})
}

func settingsPinnedAction(node *yaml.Node, action string) bool {
	if node == nil || node.Kind != yaml.ScalarNode {
		return false
	}
	sha, ok := strings.CutPrefix(node.Value, action+"@")
	if !ok || len(sha) != 40 {
		return false
	}
	for _, digit := range sha {
		if !strings.ContainsRune("0123456789abcdef", digit) {
			return false
		}
	}
	return true
}
