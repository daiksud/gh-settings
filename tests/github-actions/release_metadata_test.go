package githubactions

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"go.yaml.in/yaml/v4"
)

func TestReleaseMetadataIsResolvedBeforeProductBuilds(t *testing.T) {
	for _, name := range []string{"release", "push"} {
		var workflow struct {
			On   map[string]yaml.Node
			Jobs map[string]struct {
				Uses        string
				Needs       yaml.Node
				With        map[string]string
				Permissions map[string]string
				Outputs     map[string]string
			}
		}
		data, err := os.ReadFile(filepath.Join(root(), ".github/workflows", name+".yml"))
		if err != nil {
			t.Fatal(err)
		}
		if err := yaml.Unmarshal(data, &workflow); err != nil {
			t.Fatal(err)
		}
		version := workflow.Jobs["version"]
		if version.Uses != "./.github/workflows/version.yml" || !reflect.DeepEqual(version.Permissions, map[string]string{"contents": "read"}) {
			t.Fatalf("%s must resolve versions using the shared read-only workflow", name)
		}
		if name == "push" {
			if len(version.With) != 0 || workflow.Jobs["candidate"].With["version"] != "${{ needs.version.outputs.version }}" {
				t.Fatal("main must build the version calculated without a promotion override")
			}
			continue
		}
		if len(workflow.On) != 1 || workflow.On["workflow_dispatch"].Kind == 0 || strings.Contains(string(data), "inputs.version") {
			t.Fatal("version resolution must preserve explicit manual publication")
		}
		if version.Needs.Value != "source" || !reflect.DeepEqual(version.With, map[string]string{"promote-to-v1": "${{ inputs.promote_to_v1 }}"}) {
			t.Fatal("resolve after checking the source and forward the explicit graduation choice")
		}
		resolve := workflow.Jobs["resolve"]
		if resolve.Needs.Value != "version" {
			t.Fatal("select artifacts after final version resolution")
		}
		for _, key := range []string{"version", "previous-tag", "previous-sha", "release-needed"} {
			if resolve.Outputs[key] != "${{ needs.version.outputs."+key+" }}" {
				t.Fatalf("artifact selection lost frozen %s", key)
			}
		}
	}
}

func TestReleaseHasNoIssueCommentDispatchBridge(t *testing.T) {
	paths, err := filepath.Glob(filepath.Join(root(), ".github/workflows/*.yml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var w struct {
			On   map[string]yaml.Node `yaml:"on"`
			Jobs map[string]struct {
				Permissions map[string]string `yaml:"permissions"`
				Steps       []struct {
					Run string `yaml:"run"`
				} `yaml:"steps"`
			} `yaml:"jobs"`
		}
		if err := yaml.Unmarshal(data, &w); err != nil {
			t.Fatal(err)
		}
		if _, comments := w.On["issue_comment"]; !comments {
			continue
		}
		for name, job := range w.Jobs {
			if job.Permissions["actions"] == "write" {
				t.Fatalf("%s job %s can dispatch workflows from an Issue comment", filepath.Base(path), name)
			}
			for _, step := range job.Steps {
				if strings.Contains(step.Run, "release.yml") {
					t.Fatalf("%s job %s retains an Issue-comment release bridge", filepath.Base(path), name)
				}
			}
		}
	}
}
