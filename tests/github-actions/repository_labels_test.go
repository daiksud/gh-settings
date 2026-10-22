package githubactions

import (
	"os"
	"regexp"
	"slices"
	"testing"

	"go.yaml.in/yaml/v4"
)

func TestRepositoryLabelVocabulary(t *testing.T) {
	data, err := os.ReadFile("../../.github/settings.yml")
	if err != nil {
		t.Fatal(err)
	}
	var config struct {
		Settings struct {
			Labels []struct {
				Name        string `yaml:"name"`
				Color       string `yaml:"color"`
				Description string `yaml:"description"`
			} `yaml:"labels"`
		} `yaml:"settings"`
	}
	if err := yaml.Unmarshal(data, &config); err != nil {
		t.Fatal(err)
	}
	colors := regexp.MustCompile(`^[0-9a-f]{6}$`)
	names := map[string]bool{}
	for _, label := range config.Settings.Labels {
		if names[label.Name] || label.Name == "" || !colors.MatchString(label.Color) || label.Description == "" {
			t.Fatalf("invalid or duplicate label: %+v", label)
		}
		names[label.Name] = true
	}
	for _, name := range []string{
		"feat", "fix", "perf", "revert", "docs", "refactor", "style", "test", "build", "ci", "chore",
		"BREAKING CHANGE", "DEPRECATED", "accessibility", "duplicate", "good first issue", "help wanted", "invalid", "question", "wontfix",
	} {
		if !names[name] {
			t.Errorf("missing change or independent triage label %q", name)
		}
	}
	for _, legacy := range []string{"bug", "enhancement", "documentation"} {
		if names[legacy] {
			t.Errorf("retired label %q must not be recreated after assignment migration", legacy)
		}
	}
	for _, form := range []struct{ path, label string }{
		{"../../.github/ISSUE_TEMPLATE/01-bug-report.yml", "fix"},
		{"../../.github/ISSUE_TEMPLATE/02-feature-request.yml", "feat"},
	} {
		data, err := os.ReadFile(form.path)
		if err != nil {
			t.Fatal(err)
		}
		var content struct {
			Labels []string `yaml:"labels"`
		}
		if err := yaml.Unmarshal(data, &content); err != nil {
			t.Fatal(err)
		}
		if !slices.Equal(content.Labels, []string{form.label}) || !names[form.label] {
			t.Errorf("%s must use the declared %q label, got %v", form.path, form.label, content.Labels)
		}
	}
}
