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
	data, err := os.ReadFile(filepath.Join(root(), ".github/workflows/release.yml"))
	if err != nil {
		t.Fatal(err)
	}
	var workflow struct {
		On   map[string]yaml.Node `yaml:"on"`
		Jobs map[string]struct {
			Permissions map[string]string `yaml:"permissions"`
			Outputs     map[string]string `yaml:"outputs"`
			Steps       []step            `yaml:"steps"`
		} `yaml:"jobs"`
	}
	if err := yaml.Unmarshal(data, &workflow); err != nil {
		t.Fatal(err)
	}
	if len(workflow.On) != 1 || strings.Contains(string(data), "inputs.version") {
		t.Fatal("version calculation must not add a publication trigger or manual version override")
	}
	resolve := workflow.Jobs["resolve"]
	if !reflect.DeepEqual(resolve.Permissions, map[string]string{"contents": "read"}) || !reflect.DeepEqual(resolve.Outputs, map[string]string{
		"source-sha": "${{ github.sha }}", "version": "${{ steps.metadata.outputs.version }}",
		"previous-tag": "${{ steps.metadata.outputs.previous-tag }}", "previous-sha": "${{ steps.metadata.outputs.previous-sha }}",
		"build-matrix": "${{ steps.metadata.outputs.build-matrix }}", "acceptance-matrix": "${{ steps.metadata.outputs.acceptance-matrix }}",
	}) {
		t.Fatal("metadata resolution must be read-only and publish one fixed source/version/baseline/matrix tuple")
	}
	metadata := findStep(t, job{Steps: resolve.Steps}, "metadata")
	if metadata.Uses != "./.github/actions/release-metadata" || metadata.Run != "" || metadata.If != "" || metadata.Continue || !reflect.DeepEqual(metadata.Env, map[string]string{
		"GH_TOKEN": "${{ github.token }}",
	}) || !reflect.DeepEqual(metadata.With, map[string]string{"source-sha": "${{ github.sha }}"}) {
		t.Fatal("resolve must delegate validated metadata and matrices from the frozen source without success overrides")
	}
	var checkout step
	for _, s := range resolve.Steps {
		if strings.HasPrefix(s.Uses, "actions/checkout@") {
			checkout = s
		}
	}
	if checkout.With["ref"] != "${{ github.sha }}" || checkout.With["fetch-depth"] != "0" || checkout.With["persist-credentials"] != "false" {
		t.Fatal("version calculation requires complete frozen history without persisted credentials")
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
