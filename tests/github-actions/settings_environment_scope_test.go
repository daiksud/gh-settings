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
	settingsWorkflowPath = ".github/workflows/settings-main.yml"
	settingsEnvironment  = "gh-settings"
)

// The existing required CI pass check runs this test with the rest of the Go suite.
// The dedicated PR workflow runs it separately to give reviewers a focused result.
func TestSettingsEnvironmentScope(t *testing.T) {
	dir := filepath.Join(root(), ".github", "workflows")
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
			environment, present := job["environment"]
			if !present {
				continue
			}
			name, err := settingsEnvironmentName(environment)
			if err != nil {
				return fmt.Errorf("%s: jobs.%s.environment: %w", path, jobName, err)
			}
			if !strings.EqualFold(name, settingsEnvironment) {
				continue
			}
			if path != settingsWorkflowPath || jobName != "apply" || environment.Kind != yaml.ScalarNode || name != settingsEnvironment {
				return fmt.Errorf("%s: jobs.%s.environment: %q is allowed only as a literal in %s jobs.apply", path, jobName, name, settingsWorkflowPath)
			}
			found++
		}
	}
	if found != 1 {
		return fmt.Errorf("expected exactly one %q Environment, at %s jobs.apply; found %d", settingsEnvironment, settingsWorkflowPath, found)
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
	const other = "jobs:\n  check:\n    environment: preview\n"
	for _, test := range []struct {
		name, main, extra string
		valid             bool
	}{
		{"only approved job", allowed, "", true},
		{"different static environment", allowed, other, true},
		{"unauthorized scalar", allowed, "jobs:\n  check:\n    environment: gh-settings\n", false},
		{"unauthorized quoted scalar", allowed, "jobs:\n  check:\n    environment: 'gh-settings'\n", false},
		{"unauthorized case variant", allowed, "jobs:\n  check:\n    environment: GH-SETTINGS\n", false},
		{"unauthorized name mapping", allowed, "jobs:\n  check:\n    environment: {name: gh-settings}\n", false},
		{"unauthorized job in allowed workflow", allowed + "  other:\n    environment: gh-settings\n", "", false},
		{"dynamic scalar", allowed, "jobs:\n  check:\n    environment: ${{ vars.DEPLOY_ENV }}\n", false},
		{"dynamic name mapping", allowed, "jobs:\n  check:\n    environment: {name: \"${{ 'gh-settings' }}\"}\n", false},
		{"missing approved environment", "jobs:\n  apply:\n    runs-on: ubuntu-24.04\n", "", false},
		{"duplicate environment key", "jobs:\n  apply:\n    environment: gh-settings\n    environment: preview\n", "", false},
		{"merged environment key", "jobs:\n  template: &defaults\n    environment: preview\n  apply:\n    <<: *defaults\n    environment: gh-settings\n", "", false},
		{"aliased environment", "jobs:\n  check:\n    environment: &protected gh-settings\n  apply:\n    environment: *protected\n", "", false},
		{"malformed YAML", allowed, "jobs: [\n", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			files := map[string][]byte{settingsWorkflowPath: []byte(test.main)}
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
