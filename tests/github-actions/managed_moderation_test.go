package githubactions

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/daiksud/gh-settings/internal/github"
	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/workspace"
	"go.yaml.in/yaml/v4"
)

// This pinned operational fixture is not a CLI importer or compatibility path.
// It proves the future single-file nesting change without making v0.44.1 read an
// unsupported key. Remove the projection when a compatible package is adopted.
func managedModerationCandidate(t *testing.T, data []byte) []byte {
	t.Helper()
	w, _ := readWorkflow(t, "../../.github/workflows/settings-apply.yml")
	if w.Jobs["settings-apply"].Steps[2].Uses != "daiksud/gh-settings@2079ede46bcb3369d9575c1941ac88e4a37c081d" {
		t.Fatal("reassess the operational package boundary before removing this fixture projection")
	}
	if _, err := workspace.DecodeConfig(data); err == nil || !strings.Contains(err.Error(), "repository.settings.interaction_limits is no longer supported") {
		t.Fatalf("expected the known source/published shape boundary: %v", err)
	}
	var document yaml.Node
	if err := yaml.Unmarshal(data, &document); err != nil {
		t.Fatal(err)
	}
	child := func(n *yaml.Node, key string) *yaml.Node {
		for i := 0; i < len(n.Content); i += 2 {
			if n.Content[i].Value == key {
				return n.Content[i+1]
			}
		}
		t.Fatalf("missing %s", key)
		return nil
	}
	controls := child(child(document.Content[0], "repository"), "settings")
	for i := 0; i < len(controls.Content); i += 2 {
		if controls.Content[i].Value == "moderation" {
			t.Fatal("the old package must not receive moderation yet")
		}
	}
	found := false
	for i := 0; i < len(controls.Content); i += 2 {
		if controls.Content[i].Value != "interaction_limits" {
			continue
		}
		key, value := controls.Content[i], controls.Content[i+1]
		controls.Content[i] = &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "moderation"}
		controls.Content[i+1] = &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map", Content: []*yaml.Node{key, value}}
		found = true
		break
	}
	if !found {
		t.Fatal("missing operational interaction limits")
	}
	candidate, err := yaml.Marshal(&document)
	if err != nil {
		t.Fatal(err)
	}
	return candidate
}

func TestManagedModerationCandidatePreservesCurrentPolicy(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(root(), ".github/settings.yml"))
	if err != nil {
		t.Fatal(err)
	}
	candidate := managedModerationCandidate(t, data)
	ui, err := workspace.DecodeConfig(candidate)
	if err != nil {
		t.Fatal(err)
	}
	d, err := github.DeclarationFromUI(ui)
	if err != nil {
		t.Fatal(err)
	}
	observed, _ := settings.NewPullRequestLimitsObservation(settings.PullRequestLimitsValues{Enabled: false, Maximum: 1, IncludeDrafts: false})
	managed, changes, err := settings.ComparePullRequestLimits(observed, d.PullRequestLimits)
	if err != nil || managed != 3 || len(changes) != 0 || d.PullRequestExemptions == nil || len(d.PullRequestExemptions.Logins()) != 0 {
		t.Fatalf("candidate changed observed cap policy: %#v %d %v %v", d, managed, changes, err)
	}
	// Unwrap only the newly added node: every original value, list and sibling
	// must survive this candidate without relying on a second format decoder.
	var old, next map[string]any
	if err := yaml.Unmarshal(data, &old); err != nil {
		t.Fatal(err)
	}
	if err := yaml.Unmarshal(candidate, &next); err != nil {
		t.Fatal(err)
	}
	controls := next["repository"].(map[string]any)["settings"].(map[string]any)
	controls["interaction_limits"] = controls["moderation"].(map[string]any)["interaction_limits"]
	delete(controls, "moderation")
	if !reflect.DeepEqual(old, next) {
		t.Fatal("the candidate changed additional declaration semantics")
	}
}
