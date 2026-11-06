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
	settingsWorkflowPath      = ".github/workflows/settings-apply.yml"
	settingsPRWorkflowPath    = ".github/workflows/settings-pr.yml"
	settingsEnvironment       = "gh-settings"
	settingsGuardCheckName    = "settings-env"
	retiredSettingsPlanPath   = ".github/workflows/settings-plan.yml"
	settingsPlanEnvironment   = "gh-settings-plan"
	settingsValidateCheckName = "settings-pr"
)

// The screening check runs this test with the rest of the Go suite.
// Candidate source owns this check; it is not an independent trusted-main guard.
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
	planChecks := 0
	if _, present := workflows[retiredSettingsPlanPath]; present {
		return fmt.Errorf("%s is retired; required settings validation belongs in %s", retiredSettingsPlanPath, settingsPRWorkflowPath)
	}
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
		jobsNode, ok := workflow["jobs"]
		if !ok {
			return fmt.Errorf("%s: missing jobs", path)
		}
		jobs, err := settingsYAMLMap(jobsNode)
		if err != nil {
			return fmt.Errorf("%s: jobs: %w", path, err)
		}
		if path == settingsPRWorkflowPath {
			if err := checkSettingsValidationWorkflow(workflow); err != nil {
				return fmt.Errorf("%s: %w", path, err)
			}
		}
		for jobName, jobNode := range jobs {
			job, err := settingsYAMLMap(jobNode)
			if err != nil {
				return fmt.Errorf("%s: jobs.%s: %w", path, jobName, err)
			}
			// Reserve the validation name and reject recreation of the retired guard.
			// Candidate-controlled tests cannot establish trusted check identity.
			checkName := jobName
			if node, exists := job["name"]; exists {
				if node.Kind != yaml.ScalarNode || strings.Contains(node.Value, "${{") {
					return fmt.Errorf("%s: jobs.%s.name: dynamic or non-scalar job names are not permitted", path, jobName)
				}
				checkName = node.Value
			}
			if jobName == settingsValidateCheckName || checkName == settingsValidateCheckName {
				if path != settingsPRWorkflowPath || jobName != settingsValidateCheckName || checkName != settingsValidateCheckName {
					return fmt.Errorf("%s: jobs.%s: check name %q is reserved", path, jobName, settingsValidateCheckName)
				}
				planChecks++
			}
			if jobName == "settings-validate" || checkName == "settings-validate" || jobName == settingsGuardCheckName || checkName == settingsGuardCheckName || jobName == "settings-plan" || checkName == "settings-plan" {
				return fmt.Errorf("%s: retired check %q must not be recreated", path, checkName)
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
			if path != settingsWorkflowPath || jobName != "settings-apply" || name != settingsEnvironment {
				return fmt.Errorf("%s: jobs.%s.environment: %q is allowed only as a literal in %s jobs.settings-apply", path, jobName, name, settingsWorkflowPath)
			}
			found++
		}
	}
	if found != 1 {
		return fmt.Errorf("expected exactly one %q Environment, at %s jobs.settings-apply; found %d", settingsEnvironment, settingsWorkflowPath, found)
	}
	if planChecks != 1 {
		return fmt.Errorf("expected exactly one settings validation check in %s; found %d", settingsPRWorkflowPath, planChecks)
	}
	return nil
}

// The default pull_request activities validate all updates to the merge candidate.
func checkSettingsPRTrigger(node *yaml.Node) error {
	if node == nil {
		return fmt.Errorf("expected pull_request to main")
	}
	events, err := settingsYAMLMap(node)
	if err != nil {
		return err
	}
	target, ok := events["pull_request"]
	if !ok || len(events) != 1 {
		return fmt.Errorf("expected pull_request only")
	}
	filters, err := settingsYAMLMap(target)
	if err != nil {
		return err
	}
	if len(filters) != 1 || filters["branches"] == nil {
		return fmt.Errorf("required PR validation must use default event types and main only")
	}
	branches := filters["branches"]
	if branches.Kind != yaml.SequenceNode || len(branches.Content) != 1 ||
		branches.Content[0].Kind != yaml.ScalarNode || branches.Content[0].Value != "main" {
		return fmt.Errorf("required PR validation must target main")
	}
	return nil
}

// Keep this guard about the Environment boundary. Privileged steps and Action
// choices remain the responsibility of owner review.
func checkSettingsMainTrigger(node *yaml.Node) error {
	events, err := settingsYAMLMap(node)
	if err != nil || len(events) != 1 || events["workflow_dispatch"] == nil {
		return fmt.Errorf("expected only manual workflow_dispatch")
	}
	dispatch := events["workflow_dispatch"]
	if dispatch.Tag != "!!null" {
		return fmt.Errorf("manual apply must not accept dispatch inputs")
	}
	return nil
}

// Reject merges, aliases and duplicate keys at the workflow/job boundary rather
// than silently missing an inherited or shadowed privileged Environment.
func settingsYAMLMap(node *yaml.Node) (map[string]*yaml.Node, error) {
	if node == nil {
		return nil, fmt.Errorf("expected YAML mapping, got no value")
	}
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
	const allowed = "jobs:\n  settings-apply:\n    environment: gh-settings\n"
	const other = "jobs:\n  check:\n    environment: preview\n"
	for _, test := range []struct {
		name, main, extra string
		valid             bool
	}{
		{"only approved job", allowed, "", true},
		{"approved name mapping", "jobs:\n  settings-apply:\n    environment: {name: gh-settings, url: \"${{ steps.deploy.outputs.url }}\"}\n", "", true},
		{"approved job dynamic name mapping", "jobs:\n  settings-apply:\n    environment: {name: \"${{ vars.DEPLOY_ENV }}\"}\n", "", false},
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
		{"missing approved environment", "jobs:\n  settings-apply:\n    runs-on: ubuntu-24.04\n", "", false},
		{"duplicate environment key", "jobs:\n  settings-apply:\n    environment: gh-settings\n    environment: preview\n", "", false},
		{"merged environment key", "jobs:\n  template: &defaults\n    environment: preview\n  settings-apply:\n    <<: *defaults\n    environment: gh-settings\n", "", false},
		{"aliased environment", "jobs:\n  check:\n    environment: &protected gh-settings\n  settings-apply:\n    environment: *protected\n", "", false},
		{"malformed YAML", allowed, "jobs: [\n", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			files := map[string][]byte{
				settingsWorkflowPath:   []byte("on: {workflow_dispatch: null}\n" + test.main),
				settingsPRWorkflowPath: []byte(settingsValidationFixture),
			}
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
		{"manual dispatch", "on: {workflow_dispatch: null}", true},
		{"automatic main push", "on: {push: {branches: [main]}}", false},
		{"main push with paths", "on:\n  push:\n    branches:\n      - main\n    paths: ['.github/settings.yml']", false},
		{"main push with paths-ignore", "on: {push: {branches: [main], paths-ignore: ['docs/**']}}", false},
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
				settingsWorkflowPath:   []byte(test.trigger + "\njobs:\n  settings-apply:\n    environment: gh-settings\n"),
				settingsPRWorkflowPath: []byte(settingsValidationFixture),
			}
			err := checkSettingsEnvironmentScope(files)
			if (err == nil) != test.valid {
				t.Fatalf("valid=%t, got error: %v", test.valid, err)
			}
		})
	}
}
