package main

import (
	"bytes"
	"testing"

	"github.com/daiksud/gh-settings/internal/settings"
	"go.yaml.in/yaml/v4"
)

// TestExportFiveSettings covers the first export.feature.md scenario.
func TestExportFiveSettings(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"export"}, &stdout, &stderr, commandOperations{export: func() ([]byte, error) {
		return exportSettings(settings.RepositoryState{Description: "Example repository", Homepage: "https://example.com", HasIssues: true, HasWiki: true})
	}})
	if code != 0 || stderr.Len() != 0 {
		t.Fatalf("status=%d, stderr=%q", code, stderr.String())
	}
	output := exportedRepository(t, stdout.Bytes())
	expected := map[string]any{"description": "Example repository", "homepage": "https://example.com", "has_issues": true, "has_projects": false, "has_wiki": true}
	if len(output) != len(expected) {
		t.Fatalf("keys=%v", output)
	}
	for key, value := range expected {
		if output[key] != value {
			t.Errorf("%s = %#v, want %#v", key, output[key], value)
		}
	}
}

func exportedRepository(t *testing.T, data []byte) map[string]any {
	t.Helper()
	var root map[string]any
	if err := yaml.Unmarshal(data, &root); err != nil {
		t.Fatal(err)
	}
	groups, ok := root["settings"].(map[string]any)
	if !ok || len(root) != 2 || root["schemaVersion"] != 1 || len(groups) != 1 {
		t.Fatalf("export root=%v, want schemaVersion: 1 and settings.repository", root)
	}
	repository, ok := groups["repository"].(map[string]any)
	if !ok {
		t.Fatalf("settings=%v, want repository mapping", groups)
	}
	return repository
}

func TestExportLiteralOutput(t *testing.T) {
	for _, test := range []struct {
		name  string
		state settings.RepositoryState
		want  string
	}{
		{
			name:  "five values",
			state: settings.RepositoryState{Description: "Example repository", Homepage: "https://example.com", HasIssues: true, HasWiki: true},
			want:  "schemaVersion: 1\nsettings:\n    repository:\n        description: Example repository\n        homepage: https://example.com\n        has_issues: true\n        has_projects: false\n        has_wiki: true\n",
		},
		{
			name:  "zero values",
			state: settings.RepositoryState{},
			want:  "schemaVersion: 1\nsettings:\n    repository:\n        description: \"\"\n        homepage: \"\"\n        has_issues: false\n        has_projects: false\n        has_wiki: false\n",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := exportSettings(test.state)
			if err != nil || string(got) != test.want {
				t.Fatalf("output=%q err=%v, want %q", got, err, test.want)
			}
		})
	}
}

func TestExportPreservesZeroValues(t *testing.T) {
	data, err := exportSettings(settings.RepositoryState{})
	if err != nil {
		t.Fatal(err)
	}
	result := exportedRepository(t, data)
	if len(result) != 5 {
		t.Fatalf("fields: %v", result)
	}
	for _, key := range []string{"description", "homepage"} {
		if value, ok := result[key]; !ok || value != "" {
			t.Errorf("%s=%#v, present=%v", key, value, ok)
		}
	}
	for _, key := range []string{"has_issues", "has_projects", "has_wiki"} {
		if value, ok := result[key]; !ok || value != false {
			t.Errorf("%s=%#v, present=%v", key, value, ok)
		}
	}
}

func TestExportRepositoryFailure(t *testing.T) {
	t.Chdir(t.TempDir())
	var stdout, stderr bytes.Buffer
	code := run([]string{"export"}, &stdout, &stderr, commandOperations{export: exportCurrent})
	if code == 0 || stdout.Len() != 0 || stderr.Len() == 0 {
		t.Fatalf("status=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
}
