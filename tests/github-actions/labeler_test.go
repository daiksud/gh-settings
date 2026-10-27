package githubactions

import (
	"bytes"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"go.yaml.in/yaml/v4"
)

func TestPathLabelPolicy(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(root(), ".github/labeler.yml"))
	if err != nil {
		t.Fatal(err)
	}
	var rules map[string][]struct {
		Files []struct {
			All []string `yaml:"any-glob-to-all-files"`
		} `yaml:"changed-files"`
	}
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(&rules); err != nil {
		t.Fatal(err)
	}
	if len(rules) != 4 {
		t.Fatal("only four unambiguous path categories may be automated")
	}
	patterns := map[string][]string{}
	for label, groups := range rules {
		if !slices.Contains([]string{"docs", "test", "build", "ci"}, label) || len(groups) != 1 || len(groups[0].Files) != 1 || len(groups[0].Files[0].All) != 1 {
			t.Fatalf("%s must use one all-files brace-union, not branch or semantic rules", label)
		}
		union := groups[0].Files[0].All[0]
		if !strings.HasPrefix(union, "{") || !strings.HasSuffix(union, "}") {
			t.Fatalf("%s must use one union covering all category paths", label)
		}
		patterns[label] = strings.Split(union[1:len(union)-1], ",")
	}
	for _, test := range []struct {
		name  string
		files []string
		want  []string
	}{
		{"root and nested documents", []string{"README.md", "docs/contributing/labels.md", ".agents/skills/code-review/SKILL.md"}, []string{"docs"}},
		{"tests across owners", []string{"main_test.go", "internal/settings/labels_test.go", "tests/github-actions/fixture.sh"}, []string{"test"}},
		{"build inputs", []string{"go.mod", "go.sum", "mise.toml"}, []string{"build"}},
		{"CI automation", []string{".github/workflows/ci-pr.yml", ".github/workflows/ci-push.yml", ".github/workflows/prlint.yml", ".github/workflows/labeler.yml", ".github/actions/ci-commit/action.yml", ".github/scripts/ci-artifacts.test.sh", ".github/labeler.yml"}, []string{"ci"}},
		{"PR template", []string{".github/pull_request_template.md"}, []string{"docs"}},
		{"nested skill documentation", []string{".agents/skills/code-review/SKILL.md"}, []string{"docs"}},
		{"release workflow excluded", []string{".github/workflows/release.yml"}, nil},
		{"settings workflow excluded", []string{".github/workflows/settings-main.yml"}, nil},
		{"release Action excluded", []string{".github/actions/release-build/action.yml"}, nil},
		{"old internal commit Action excluded", []string{".github/actions/commit-stage/action.yml"}, nil},
		{"unrelated internal helper excluded", []string{".github/scripts/verify-settings-handoff.sh"}, nil},
		{"product and supporting docs", []string{"main.go", "README.md", "main_test.go"}, nil},
		{"product and test", []string{"internal/settings/labels.go", "internal/settings/labels_test.go"}, nil},
		{"mixed maintenance categories", []string{"README.md", "go.mod"}, nil},
		{"public Action metadata", []string{"action.yml"}, nil},
		{"public Action implementation", []string{"action/install-cli.sh"}, nil},
		{"public and internal Actions", []string{"action.yml", ".github/workflows/ci-pr.yml"}, nil},
		{"consumer examples", []string{"examples/github-actions/settings-main.yml"}, nil},
		{"same code may be feature fix or refactor", []string{"main.go"}, nil},
		{"prefix lookalike", []string{"docs-other/code.go"}, nil},
		{"empty diff", nil, nil},
	} {
		t.Run(test.name, func(t *testing.T) {
			var got []string
			for label, alternatives := range patterns {
				all := len(test.files) > 0
				for _, file := range test.files {
					matched := false
					for _, pattern := range alternatives {
						matched = pathPolicyMatch(t, pattern, file) || matched
					}
					all = all && matched
				}
				if all {
					got = append(got, label)
				}
			}
			slices.Sort(got)
			if !slices.Equal(got, test.want) {
				t.Fatalf("%v: labels %v, want %v", test.files, got, test.want)
			}
		})
	}
}

// Model the current policy patterns, not the upstream Labeler glob engine.
func pathPolicyMatch(t *testing.T, pattern, file string) bool {
	t.Helper()
	switch {
	case pattern == "**/*_test.go":
		return strings.HasSuffix(path.Base(file), "_test.go")
	case strings.Contains(pattern, "/**/"):
		prefix, suffix, _ := strings.Cut(pattern, "/**/")
		return strings.HasPrefix(file, prefix+"/") && strings.HasSuffix(file, "/"+suffix)
	case strings.HasSuffix(pattern, "/**"):
		prefix := strings.TrimSuffix(pattern, "/**")
		if strings.ContainsAny(prefix, "*?[") {
			parent := path.Dir(prefix)
			relative := strings.TrimPrefix(file, parent+"/")
			child, _, nested := strings.Cut(relative, "/")
			if relative == file || !nested {
				return false
			}
			matched, err := path.Match(path.Base(prefix), child)
			if err != nil {
				t.Fatalf("invalid policy glob %q: %v", pattern, err)
			}
			return matched
		}
		return strings.HasPrefix(file, prefix+"/")
	default:
		matched, err := path.Match(pattern, file)
		if err != nil {
			t.Fatalf("invalid policy glob %q: %v", pattern, err)
		}
		return matched
	}
}

func TestLabelerDoesNotExecuteCandidateOrRemoveManualLabels(t *testing.T) {
	w, source := readWorkflow(t, "../../.github/workflows/labeler.yml")
	if len(w.On) != 1 || w.On["pull_request_target"].Kind == 0 || len(w.Jobs) != 1 || !reflect.DeepEqual(w.Permissions, map[string]string{"contents": "read", "pull-requests": "write"}) {
		t.Fatal("labeler must keep its API-only event and least-privilege boundary")
	}
	j := w.Jobs["labeler"]
	if len(j.Steps) != 1 || len(j.Permissions) != 0 || len(j.Env) != 0 || len(w.Env) != 0 || j.Environment != "" || j.Continue || j.Timeout != 5 {
		t.Fatal("labeler must not add credential-bearing execution paths")
	}
	s := j.Steps[0]
	if s.Uses != "actions/labeler@bf12e9b00b37c5c0ca2b87b79b2daf7891dbda13" || s.Run != "" || s.Continue || !reflect.DeepEqual(s.With, map[string]string{"sync-labels": "false", "dot": "true"}) {
		t.Fatal("use the pinned API-only action with add-only labels and dot paths")
	}
	if strings.Contains(source, "secrets.") || strings.Contains(source, "actions/checkout@") || strings.Contains(source, "head.ref") || strings.Contains(source, "pull_request.title") {
		t.Fatal("untrusted execution or branch/title-derived classification")
	}
}
