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
		"source-sha": "${{ steps.source.outputs.source-sha }}", "version": "${{ steps.metadata.outputs.version }}",
		"previous-tag": "${{ steps.metadata.outputs.previous-tag }}", "previous-sha": "${{ steps.metadata.outputs.previous-sha }}",
	}) {
		t.Fatal("metadata resolution must be read-only and publish one fixed source/version/baseline tuple")
	}
	metadata := findStep(t, job{Steps: resolve.Steps}, "metadata")
	if metadata.Run != `go run ./scripts/release-metadata/main.go resolve "$GITHUB_REPOSITORY" "$SOURCE_SHA" >> "$GITHUB_OUTPUT"` || metadata.If != "" || metadata.Continue || !reflect.DeepEqual(metadata.Env, map[string]string{
		"GH_TOKEN": "${{ github.token }}", "SOURCE_SHA": "${{ steps.source.outputs.source-sha }}",
	}) {
		t.Fatal("resolve must execute the read-only helper against the frozen source without success overrides")
	}
	var checkout step
	for _, s := range resolve.Steps {
		if strings.HasPrefix(s.Uses, "actions/checkout@") {
			checkout = s
		}
	}
	if checkout.With["ref"] != "${{ steps.source.outputs.source-sha }}" || checkout.With["fetch-depth"] != "0" || checkout.With["persist-credentials"] != "false" {
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

func TestNativeReleaseNoteCategories(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(root(), ".github/release.yml"))
	if err != nil {
		t.Fatal(err)
	}
	type category struct {
		Title   string   `yaml:"title"`
		Labels  []string `yaml:"labels"`
		Exclude struct {
			Labels []string `yaml:"labels"`
		} `yaml:"exclude"`
	}
	var config struct {
		Changelog struct {
			Categories []category `yaml:"categories"`
		} `yaml:"changelog"`
	}
	if err := yaml.Unmarshal(data, &config); err != nil {
		t.Fatal(err)
	}
	wantTitles := []string{"🚨 BREAKING CHANGE", "✨ New Feature", "🐛 Bug Fix", "🧹 Chore"}
	wantLabels := [][]string{{"BREAKING CHANGE"}, {"feat", "perf", "DEPRECATED"}, {"fix", "revert"}, {"*"}}
	wantExclusions := [][]string{nil, {"BREAKING CHANGE"}, {"BREAKING CHANGE", "feat", "perf", "DEPRECATED"}, nil}
	if len(config.Changelog.Categories) != len(wantTitles) {
		t.Fatal("native notes must retain the four ordered reader-facing categories")
	}
	for i, c := range config.Changelog.Categories {
		if c.Title != wantTitles[i] || !reflect.DeepEqual(c.Labels, wantLabels[i]) || !reflect.DeepEqual(c.Exclude.Labels, wantExclusions[i]) {
			t.Fatalf("category %d loses compatibility precedence or uncategorized changes: %+v", i, c)
		}
	}
}
