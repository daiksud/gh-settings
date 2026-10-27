package githubactions

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.yaml.in/yaml/v4"
)

const (
	settingsWorkflowPath      = ".github/workflows/settings-main.yml"
	settingsPRWorkflowPath    = ".github/workflows/settings-pr.yml"
	settingsEnvironment       = "gh-settings"
	settingsGuardWorkflowPath = ".github/workflows/settings-env.yml"
	settingsGuardCheckName    = "settings-env"
	settingsPlanWorkflowPath  = ".github/workflows/settings-plan.yml"
	settingsPlanEnvironment   = "gh-settings-plan"
	settingsPlanCheckName     = "settings-plan"
	settingsPRCondition       = "github.event_name == 'pull_request_target' && github.ref == 'refs/heads/main' && github.repository == 'daiksud/gh-settings' && github.event.pull_request.base.ref == 'main'"
)

// The ci-pr aggregate check runs this test with the rest of the Go suite.
// The dedicated PR workflow runs it separately to give reviewers a focused result.
func TestSettingsEnvironmentScope(t *testing.T) {
	dir := os.Getenv("GH_SETTINGS_WORKFLOWS_DIR")
	if dir == "" {
		dir = filepath.Join(root(), ".github", "workflows")
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	workflows := make(map[string][]byte)
	for _, entry := range entries {
		if entry.IsDir() || (filepath.Ext(entry.Name()) != ".yml" && filepath.Ext(entry.Name()) != ".yaml") {
			continue
		}
		content, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		workflows[".github/workflows/"+entry.Name()] = content
	}
	if err := checkSettingsEnvironmentScope(workflows); err != nil {
		t.Fatal(err)
	}
}

func checkSettingsEnvironmentScope(workflows map[string][]byte) error {
	found := 0
	prFound := 0
	guardChecks := 0
	planChecks := 0
	_, planPresent := workflows[settingsPlanWorkflowPath]
	for path, content := range workflows {
		decoder := yaml.NewDecoder(bytes.NewReader(content))
		var document yaml.Node
		if err := decoder.Decode(&document); err != nil {
			return fmt.Errorf("%s: decode workflow: %w", path, err)
		}
		var next yaml.Node
		if err := decoder.Decode(&next); err != io.EOF {
			return fmt.Errorf("%s: expected exactly one YAML document (second decode: %v)", path, err)
		}
		if document.Kind != yaml.DocumentNode || len(document.Content) != 1 {
			return fmt.Errorf("%s: expected a workflow document", path)
		}
		workflow, err := settingsYAMLMap(document.Content[0])
		if err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		if path == settingsWorkflowPath {
			if err := checkSettingsMainTrigger(workflow["on"]); err != nil {
				return fmt.Errorf("%s: on: %w", path, err)
			}
		}
		if path == settingsPlanWorkflowPath {
			if err := checkSettingsPlanTrigger(workflow["on"]); err != nil {
				return fmt.Errorf("%s: on: %w", path, err)
			}
			if err := checkUnprivilegedSettingsPlan(workflow); err != nil {
				return fmt.Errorf("%s: %w", path, err)
			}
		}
		jobsNode, ok := workflow["jobs"]
		if !ok {
			return fmt.Errorf("%s: missing jobs", path)
		}
		jobs, err := settingsYAMLMap(jobsNode)
		if err != nil {
			return fmt.Errorf("%s: jobs: %w", path, err)
		}
		for jobName, jobNode := range jobs {
			job, err := settingsYAMLMap(jobNode)
			if err != nil {
				return fmt.Errorf("%s: jobs.%s: %w", path, jobName, err)
			}
			// Keep the dedicated guard's check name unique among current workflows.
			// Uniqueness helps avoid ambiguous statuses; it does not prevent spoofing
			// by PR-controlled GitHub Actions, so do not remove Environment approval
			// without an independently enforceable required check.
			checkName := jobName
			if node, exists := job["name"]; exists {
				if node.Kind != yaml.ScalarNode || strings.Contains(node.Value, "${{") {
					return fmt.Errorf("%s: jobs.%s.name: dynamic or non-scalar job names are not permitted", path, jobName)
				}
				checkName = node.Value
			}
			if jobName == settingsPlanCheckName || checkName == settingsPlanCheckName {
				if path != settingsPlanWorkflowPath || jobName != settingsPlanCheckName || checkName != settingsPlanCheckName {
					return fmt.Errorf("%s: jobs.%s: check name %q is reserved", path, jobName, settingsPlanCheckName)
				}
				planChecks++
			}
			if jobName == settingsGuardCheckName || checkName == settingsGuardCheckName {
				if path != settingsGuardWorkflowPath || jobName != settingsGuardCheckName || checkName != settingsGuardCheckName {
					return fmt.Errorf("%s: jobs.%s: check name %q is reserved for %s", path, jobName, settingsGuardCheckName, settingsGuardWorkflowPath)
				}
				guardChecks++
			}
			// Called workflows can select this repository's Environment and
			// inherit its secrets. Reject calls rather than trusting unseen code.
			if _, reusable := job["uses"]; reusable {
				return fmt.Errorf("%s: jobs.%s: reusable workflow jobs are not permitted", path, jobName)
			}
			environment, present := job["environment"]
			if !present {
				continue
			}
			name, err := settingsEnvironmentName(environment)
			if err != nil {
				return fmt.Errorf("%s: jobs.%s.environment: %w", path, jobName, err)
			}
			if strings.EqualFold(strings.TrimSpace(name), settingsPlanEnvironment) {
				return fmt.Errorf("%s: jobs.%s.environment: retired observer Environment %q is forbidden", path, jobName, name)
			}
			if !strings.EqualFold(strings.TrimSpace(name), settingsEnvironment) {
				continue
			}
			if path == settingsPRWorkflowPath && jobName == "plan" && name == settingsEnvironment {
				if err := checkSettingsPRTrigger(workflow["on"]); err != nil {
					return fmt.Errorf("%s: on: %w", path, err)
				}
				condition := job["if"]
				if condition == nil || condition.Kind != yaml.ScalarNode || condition.Value != settingsPRCondition {
					return fmt.Errorf("%s: jobs.plan: missing trusted main PR-target condition", path)
				}
				prFound++
				continue
			}
			if path != settingsWorkflowPath || jobName != "apply" || name != settingsEnvironment {
				return fmt.Errorf("%s: jobs.%s.environment: %q is allowed only as a literal in %s jobs.apply or %s jobs.plan", path, jobName, name, settingsWorkflowPath, settingsPRWorkflowPath)
			}
			found++
		}
	}
	if found != 1 {
		return fmt.Errorf("expected exactly one %q Environment, at %s jobs.apply; found %d", settingsEnvironment, settingsWorkflowPath, found)
	}
	if prFound > 1 {
		return fmt.Errorf("expected at most one trusted PR plan Environment; found %d", prFound)
	}
	if guardChecks != 1 {
		return fmt.Errorf("expected exactly one guard status %q in %s; found %d", settingsGuardCheckName, settingsGuardWorkflowPath, guardChecks)
	}
	if planPresent && planChecks != 1 {
		return fmt.Errorf("expected one validation check in %s; found %d", settingsPlanWorkflowPath, planChecks)
	}
	return nil
}

// The settings-plan workflow validates candidate code without an Environment.
func checkUnprivilegedSettingsPlan(workflow map[string]*yaml.Node) error {
	permissions, err := settingsYAMLMap(workflow["permissions"])
	if err != nil || len(permissions) != 0 {
		return fmt.Errorf("PR-only settings validation must have no default token grants")
	}
	jobs, err := settingsYAMLMap(workflow["jobs"])
	if err != nil || len(jobs) != 1 || jobs[settingsPlanCheckName] == nil {
		return fmt.Errorf("PR-only settings validation requires exactly the settings-plan job")
	}
	job, err := settingsYAMLMap(jobs[settingsPlanCheckName])
	if err != nil {
		return err
	}
	if job["environment"] != nil || job["uses"] != nil || job["if"] == nil || job["if"].Kind != yaml.ScalarNode || job["if"].Value != "github.event_name == 'pull_request'" {
		return fmt.Errorf("PR-only settings validation cannot select an Environment or delegate execution")
	}
	grants, err := settingsYAMLMap(job["permissions"])
	if err != nil || len(grants) != 1 || grants["contents"] == nil || grants["contents"].Kind != yaml.ScalarNode || grants["contents"].Value != "read" {
		return fmt.Errorf("PR-only settings validation requires only Contents read")
	}
	return nil
}

// The allowlist can be deployed before its new caller. Absence does not grant
// access; any PR caller selecting this Environment must meet the full boundary.
func checkSettingsPRTrigger(node *yaml.Node) error {
	if node == nil {
		return fmt.Errorf("expected pull_request_target to main only")
	}
	events, err := settingsYAMLMap(node)
	if err != nil {
		return err
	}
	target, ok := events["pull_request_target"]
	if !ok || len(events) != 1 {
		return fmt.Errorf("expected pull_request_target only")
	}
	filters, err := settingsYAMLMap(target)
	if err != nil {
		return err
	}
	for key := range filters {
		if key != "branches" && key != "paths" && key != "types" {
			return fmt.Errorf("PR-target filter %q is not permitted", key)
		}
	}
	branches := filters["branches"]
	if branches == nil || branches.Kind != yaml.SequenceNode || len(branches.Content) != 1 || branches.Content[0].Kind != yaml.ScalarNode || branches.Content[0].Value != "main" {
		return fmt.Errorf("PR-target branches must be exactly [main]")
	}
	if types := filters["types"]; types != nil {
		if types.Kind != yaml.SequenceNode || len(types.Content) != 3 {
			return fmt.Errorf("PR-target types must be opened, synchronize and reopened")
		}
		seen := map[string]bool{}
		for _, kind := range types.Content {
			if kind.Kind != yaml.ScalarNode || seen[kind.Value] || (kind.Value != "opened" && kind.Value != "synchronize" && kind.Value != "reopened") {
				return fmt.Errorf("unsupported PR-target activity")
			}
			seen[kind.Value] = true
		}
	}
	return nil
}

// The protected manual plan remains dispatch-only; the same workflow also
// exposes a credential-free merge-candidate validation job for required PR checks.
func checkSettingsPlanTrigger(node *yaml.Node) error {
	if node == nil {
		return fmt.Errorf("expected pull_request only")
	}
	events, err := settingsYAMLMap(node)
	if err != nil {
		return err
	}
	pr, ok := events["pull_request"]
	if !ok || len(events) != 1 {
		return fmt.Errorf("expected pull_request only")
	}
	return checkSettingsPlanPREvent(pr)
}

func checkSettingsPlanPREvent(pr *yaml.Node) error {
	filters, err := settingsYAMLMap(pr)
	if err != nil {
		return err
	}
	if len(filters) != 2 || filters["branches"] == nil || filters["types"] == nil {
		return fmt.Errorf("required PR check must target main with explicit activities")
	}
	branches, types := filters["branches"], filters["types"]
	if branches.Kind != yaml.SequenceNode || len(branches.Content) != 1 ||
		branches.Content[0].Kind != yaml.ScalarNode || branches.Content[0].Value != "main" {
		return fmt.Errorf("required PR check must target only main")
	}
	if types.Kind != yaml.SequenceNode || len(types.Content) != 4 {
		return fmt.Errorf("required PR check must handle opened, synchronize, reopened and edited")
	}
	seen := make(map[string]bool)
	for _, value := range types.Content {
		if value.Kind != yaml.ScalarNode || seen[value.Value] {
			return fmt.Errorf("duplicate or invalid PR event")
		}
		switch value.Value {
		case "opened", "synchronize", "reopened", "edited":
			seen[value.Value] = true
		default:
			return fmt.Errorf("unsupported PR event %q", value.Value)
		}
	}
	return nil
}

// Keep this guard about the Environment boundary. Privileged steps and Action
// choices remain the responsibility of owner review.
func checkSettingsMainTrigger(node *yaml.Node) error {
	if node == nil {
		return fmt.Errorf("expected only push to main")
	}
	events, err := settingsYAMLMap(node)
	if err != nil {
		return err
	}
	push, ok := events["push"]
	if !ok || len(events) != 1 {
		return fmt.Errorf("expected only the push event")
	}
	filters, err := settingsYAMLMap(push)
	if err != nil {
		return err
	}
	for key := range filters {
		switch key {
		case "branches", "paths", "paths-ignore":
		default:
			return fmt.Errorf("push filter %q is not permitted", key)
		}
	}
	branches := filters["branches"]
	if branches == nil || branches.Kind != yaml.SequenceNode || len(branches.Content) != 1 ||
		branches.Content[0].Kind != yaml.ScalarNode || branches.Content[0].Value != "main" {
		return fmt.Errorf("push.branches must be exactly [main]")
	}
	return nil
}

// Reject merges, aliases and duplicate keys at the workflow/job boundary rather
// than silently missing an inherited or shadowed privileged Environment.
func settingsYAMLMap(node *yaml.Node) (map[string]*yaml.Node, error) {
	if node.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("expected YAML mapping, got kind %d", node.Kind)
	}
	values := make(map[string]*yaml.Node)
	for i := 0; i < len(node.Content); i += 2 {
		key := node.Content[i]
		if key.Kind != yaml.ScalarNode {
			return nil, fmt.Errorf("non-scalar mapping key")
		}
		if key.Value == "<<" {
			return nil, fmt.Errorf("YAML merges are not supported for workflow/job configuration")
		}
		if _, exists := values[key.Value]; exists {
			return nil, fmt.Errorf("duplicate YAML key %q", key.Value)
		}
		values[key.Value] = node.Content[i+1]
	}
	return values, nil
}

func settingsEnvironmentName(node *yaml.Node) (string, error) {
	var name string
	switch node.Kind {
	case yaml.ScalarNode:
		name = node.Value
	case yaml.MappingNode:
		fields, err := settingsYAMLMap(node)
		if err != nil {
			return "", err
		}
		n, ok := fields["name"]
		if !ok || n.Kind != yaml.ScalarNode {
			return "", fmt.Errorf("Environment mapping needs a literal name")
		}
		name = n.Value
	default:
		return "", fmt.Errorf("Environment name must be a static scalar or mapping with a static name")
	}
	// An expression could resolve to gh-settings at runtime, even if the source
	// does not contain that name. Reject it for every job, including unrelated jobs.
	if name == "" || strings.Contains(name, "${{") {
		return "", fmt.Errorf("dynamic or empty Environment name is prohibited")
	}
	return name, nil
}

func TestSettingsEnvironmentScopeFixtures(t *testing.T) {
	const allowed = "jobs:\n  apply:\n    environment: gh-settings\n"
	const guard = "jobs:\n  settings-env:\n    name: settings-env\n"
	const other = "jobs:\n  check:\n    environment: preview\n"
	for _, test := range []struct {
		name, main, extra string
		valid             bool
	}{
		{"only approved job", allowed, "", true},
		{"approved name mapping", "jobs:\n  apply:\n    environment: {name: gh-settings, url: \"${{ steps.deploy.outputs.url }}\"}\n", "", true},
		{"approved job dynamic name mapping", "jobs:\n  apply:\n    environment: {name: \"${{ vars.DEPLOY_ENV }}\"}\n", "", false},
		{"different static environment", allowed, other, true},
		{"unauthorized scalar", allowed, "jobs:\n  check:\n    environment: gh-settings\n", false},
		{"unauthorized quoted scalar", allowed, "jobs:\n  check:\n    environment: 'gh-settings'\n", false},
		{"unauthorized case variant", allowed, "jobs:\n  check:\n    environment: GH-SETTINGS\n", false},
		{"unauthorized padded name", allowed, "jobs:\n  check:\n    environment: \" gh-settings \"\n", false},
		{"unauthorized name mapping", allowed, "jobs:\n  check:\n    environment: {name: gh-settings}\n", false},
		{"unauthorized job in allowed workflow", allowed + "  other:\n    environment: gh-settings\n", "", false},
		{"reusable workflow job", allowed, "jobs:\n  delegate:\n    uses: example/repo/.github/workflows/deploy.yml@main\n", false},
		{"duplicate guard job id", allowed, "jobs:\n  settings-env:\n    runs-on: ubuntu-24.04\n", false},
		{"duplicate guard display name", allowed, "jobs:\n  unrelated:\n    name: settings-env\n", false},
		{"dynamic job display name", allowed, "jobs:\n  unrelated:\n    name: ${{ vars.CHECK_NAME }}\n", false},
		{"dynamic scalar", allowed, "jobs:\n  check:\n    environment: ${{ vars.DEPLOY_ENV }}\n", false},
		{"dynamic name mapping", allowed, "jobs:\n  check:\n    environment: {name: \"${{ 'gh-settings' }}\"}\n", false},
		{"missing approved environment", "jobs:\n  apply:\n    runs-on: ubuntu-24.04\n", "", false},
		{"duplicate environment key", "jobs:\n  apply:\n    environment: gh-settings\n    environment: preview\n", "", false},
		{"merged environment key", "jobs:\n  template: &defaults\n    environment: preview\n  apply:\n    <<: *defaults\n    environment: gh-settings\n", "", false},
		{"aliased environment", "jobs:\n  check:\n    environment: &protected gh-settings\n  apply:\n    environment: *protected\n", "", false},
		{"malformed YAML", allowed, "jobs: [\n", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			files := map[string][]byte{settingsWorkflowPath: []byte("on: {push: {branches: [main]}}\n" + test.main), settingsGuardWorkflowPath: []byte(guard)}
			if test.extra != "" {
				files[".github/workflows/other.yaml"] = []byte(test.extra)
			}
			err := checkSettingsEnvironmentScope(files)
			if (err == nil) != test.valid {
				t.Fatalf("valid=%t, got error: %v", test.valid, err)
			}
		})
	}
}

func TestSettingsEnvironmentMainTriggerFixtures(t *testing.T) {
	for _, test := range []struct {
		name, trigger string
		valid         bool
	}{
		{"main push", "on: {push: {branches: [main]}}", true},
		{"main push with paths", "on:\n  push:\n    branches:\n      - main\n    paths: ['.github/settings.yml']", true},
		{"main push with paths-ignore", "on: {push: {branches: [main], paths-ignore: ['docs/**']}}", true},
		{"missing trigger", "", false},
		{"scalar push", "on: push", false},
		{"sequence push", "on: [push]", false},
		{"unfiltered push", "on: {push: null}", false},
		{"extra event", "on: {push: {branches: [main]}, workflow_dispatch: null}", false},
		{"pull request event", "on: {pull_request: {branches: [main]}}", false},
		{"missing branches", "on: {push: {paths: ['.github/settings.yml']}}", false},
		{"other branch", "on: {push: {branches: [develop]}}", false},
		{"extra branch", "on: {push: {branches: [main, develop]}}", false},
		{"wildcard branch", "on: {push: {branches: ['*']}}", false},
		{"scalar branch", "on: {push: {branches: main}}", false},
		{"dynamic branch", "on: {push: {branches: ['${{ vars.BRANCH }}']}}", false},
		{"tag trigger", "on: {push: {branches: [main], tags: ['v*']}}", false},
		{"tag exclusion", "on: {push: {branches: [main], tags-ignore: ['*']}}", false},
		{"branch exclusion", "on: {push: {branches: [main], branches-ignore: [develop]}}", false},
		{"duplicate push", "on: {push: {branches: [main]}, push: {tags: ['*']}}", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			files := map[string][]byte{
				settingsWorkflowPath:      []byte(test.trigger + "\njobs:\n  apply:\n    environment: gh-settings\n"),
				settingsGuardWorkflowPath: []byte("jobs:\n  settings-env:\n    name: settings-env\n"),
			}
			err := checkSettingsEnvironmentScope(files)
			if (err == nil) != test.valid {
				t.Fatalf("valid=%t, got error: %v", test.valid, err)
			}
		})
	}
}
