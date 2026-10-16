package app

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/workspace"
)

func TestPlanResults(t *testing.T) {
	for _, test := range []struct {
		name, input string
		current     settings.RepositoryState
		want        settings.RepositoryComparison
	}{
		{"changes", "schemaVersion: 1\nsettings:\n  repository:\n    description: after\n    has_issues: false\n", settings.RepositoryState{Description: "before", HasIssues: true}, settings.RepositoryComparison{Managed: 2, Changes: []settings.RepositoryChange{{Field: "description", Before: "before", After: "after"}, {Field: "has_issues", Before: true, After: false}}}},
		{"no differences", "schemaVersion: 1\nsettings:\n  repository:\n    has_issues: false\n", settings.RepositoryState{}, settings.RepositoryComparison{Managed: 1}},
		{"no managed fields", "schemaVersion: 1\n", settings.RepositoryState{HasIssues: true}, settings.RepositoryComparison{}},
	} {
		t.Run(test.name, func(t *testing.T) {
			path := planCheckout(t, test.input, true)
			t.Setenv("GH_REPO", "other/repository")
			target := repository.Repository{Host: "github.com", Owner: "owner", Name: "current"}
			calls := 0
			got, err := Plan(func(repo repository.Repository) (settings.RepositoryState, error) {
				calls++
				if repo != target {
					t.Fatalf("read target=%+v, want %+v", repo, target)
				}
				return test.current, nil
			})
			want := PlanResult{Target: target, Comparison: test.want}
			if err != nil || !reflect.DeepEqual(got, want) || calls != 1 {
				t.Fatalf("result=%+v error=%v reads=%d, want %+v and one read", got, err, calls, want)
			}
			after, err := os.ReadFile(path)
			if err != nil || !bytes.Equal(after, []byte(test.input)) {
				t.Fatalf("settings changed: %q error=%v", after, err)
			}
		})
	}
}

func TestPlanValidatesBeforeRead(t *testing.T) {
	for _, test := range []struct{ input, want string }{
		{"settings: {}\n", "missing schemaVersion"},
		{"schemaVersion: 1\nsettings:\n  repository:\n    has_issues: [\n", "decode YAML"},
	} {
		planCheckout(t, test.input, false) // No origin: validation must win over target resolution.
		calls := 0
		got, err := Plan(func(repository.Repository) (settings.RepositoryState, error) {
			calls++
			return settings.RepositoryState{}, nil
		})
		if err == nil || !strings.Contains(err.Error(), "validate "+workspace.SettingsPath) || !strings.Contains(err.Error(), test.want) || !reflect.DeepEqual(got, PlanResult{}) || calls != 0 {
			t.Fatalf("result=%+v error=%v reads=%d, want validation %q before target/read", got, err, calls, test.want)
		}
	}
}

func TestPlanTargetFailureBeforeRead(t *testing.T) {
	planCheckout(t, "schemaVersion: 1\nsettings:\n  repository:\n    has_issues: false\n", false)
	calls := 0
	got, err := Plan(func(repository.Repository) (settings.RepositoryState, error) {
		calls++
		return settings.RepositoryState{}, nil
	})
	if err == nil || !strings.Contains(err.Error(), "a GitHub origin remote is required") || !reflect.DeepEqual(got, PlanResult{}) || calls != 0 {
		t.Fatalf("result=%+v error=%v reads=%d, want target failure before read", got, err, calls)
	}
}

func TestPlanReadFailure(t *testing.T) {
	planCheckout(t, "schemaVersion: 1\nsettings:\n  repository:\n    has_issues: false\n", true)
	readError := errors.New("read unavailable")
	calls := 0
	got, err := Plan(func(repository.Repository) (settings.RepositoryState, error) {
		calls++
		return settings.RepositoryState{}, readError
	})
	if !errors.Is(err, readError) || !reflect.DeepEqual(got, PlanResult{}) || calls != 1 {
		t.Fatalf("result=%+v error=%v reads=%d, want read error and zero result", got, err, calls)
	}
}

func planCheckout(t *testing.T, input string, origin bool) string {
	t.Helper()
	root := t.TempDir()
	if output, err := exec.Command("git", "init", "--quiet", root).CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, output)
	}
	if origin {
		if output, err := exec.Command("git", "-C", root, "remote", "add", "origin", "https://github.com/owner/current.git").CombinedOutput(); err != nil {
			t.Fatalf("git origin: %v: %s", err, output)
		}
	}
	if err := os.Mkdir(filepath.Join(root, ".github"), 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, workspace.SettingsPath)
	if err := os.WriteFile(path, []byte(input), 0600); err != nil {
		t.Fatal(err)
	}
	t.Chdir(root)
	return path
}
