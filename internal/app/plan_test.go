package app

import (
	"bytes"
	"errors"
	"os"
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
		want        settings.Comparison
	}{
		{"changes", "schemaVersion: 1\nsettings:\n  repository:\n    description: after\n    has_issues: false\n", settings.RepositoryState{Description: "before", HasIssues: true}, settings.Comparison{Managed: 2, Changes: []settings.Change{comparisonChange(t, "repository.description", "before", "after"), comparisonChange(t, "repository.has_issues", true, false)}}},
		{"no differences", "schemaVersion: 1\nsettings:\n  repository:\n    has_issues: false\n", settings.RepositoryState{}, settings.Comparison{Managed: 1}},
	} {
		t.Run(test.name, func(t *testing.T) {
			path := appCheckout(t, test.input, true)
			t.Setenv("GH_REPO", "other/repository")
			target := repository.Repository{Host: "github.com", Owner: "owner", Name: "current"}
			calls := 0
			got, err := Plan(Reader{Repository: func(repo repository.Repository) (settings.RepositoryState, error) {
				calls++
				if repo != target {
					t.Fatalf("read target=%+v, want %+v", repo, target)
				}
				return test.current, nil
			}})
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

func TestPlanNoManagedSettings(t *testing.T) {
	for _, test := range []struct{ name, input string }{
		{"version only", "schemaVersion: 1\n"},
		{"empty settings", "schemaVersion: 1\nsettings: {}\n"},
		{"empty repository", "schemaVersion: 1\nsettings:\n  repository: {}\n"},
	} {
		t.Run(test.name, func(t *testing.T) {
			path := appCheckout(t, test.input, false)
			calls := 0
			got, err := Plan(Reader{Repository: func(repository.Repository) (settings.RepositoryState, error) {
				calls++
				return settings.RepositoryState{}, nil
			}})
			if err != nil || !reflect.DeepEqual(got, PlanResult{}) || calls != 0 {
				t.Fatalf("result=%+v error=%v reads=%d, want local no-op", got, err, calls)
			}
			after, err := os.ReadFile(path)
			if err != nil || string(after) != test.input {
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
		appCheckout(t, test.input, false) // No origin: validation must win over target resolution.
		calls := 0
		got, err := Plan(Reader{Repository: func(repository.Repository) (settings.RepositoryState, error) {
			calls++
			return settings.RepositoryState{}, nil
		}})
		if err == nil || !strings.Contains(err.Error(), "validate "+workspace.SettingsPath) || !strings.Contains(err.Error(), test.want) || !reflect.DeepEqual(got, PlanResult{}) || calls != 0 {
			t.Fatalf("result=%+v error=%v reads=%d, want validation %q before target/read", got, err, calls, test.want)
		}
		var failure *PlanError
		if !errors.As(err, &failure) || failure.Stage != "invalid-config" || errors.Unwrap(err) == nil || err.Error() != errors.Unwrap(err).Error() {
			t.Fatalf("validation failure lost stage, original text or cause: %v", err)
		}
	}
}

func TestPlanTargetFailureBeforeRead(t *testing.T) {
	appCheckout(t, "schemaVersion: 1\nsettings:\n  repository:\n    has_issues: false\n", false)
	calls := 0
	got, err := Plan(Reader{Repository: func(repository.Repository) (settings.RepositoryState, error) {
		calls++
		return settings.RepositoryState{}, nil
	}})
	if err == nil || !strings.Contains(err.Error(), "a GitHub origin remote is required") || !reflect.DeepEqual(got, PlanResult{}) || calls != 0 {
		t.Fatalf("result=%+v error=%v reads=%d, want target failure before read", got, err, calls)
	}
	var failure *PlanError
	if !errors.As(err, &failure) || failure.Stage != "target-failure" || errors.Unwrap(err) == nil || err.Error() != errors.Unwrap(err).Error() {
		t.Fatalf("target failure lost stage, original text or cause: %v", err)
	}
}

func TestPlanReadFailure(t *testing.T) {
	appCheckout(t, "schemaVersion: 1\nsettings:\n  repository:\n    has_issues: false\n", true)
	readError := errors.New("read unavailable")
	calls := 0
	got, err := Plan(Reader{Repository: func(repository.Repository) (settings.RepositoryState, error) {
		calls++
		return settings.RepositoryState{}, readError
	}})
	if !errors.Is(err, readError) || !reflect.DeepEqual(got, PlanResult{}) || calls != 1 {
		t.Fatalf("result=%+v error=%v reads=%d, want read error and zero result", got, err, calls)
	}
	var failure *PlanError
	if !errors.As(err, &failure) || failure.Stage != "read-failure" || failure.Target != (repository.Repository{Host: "github.com", Owner: "owner", Name: "current"}) || err.Error() != readError.Error() {
		t.Fatalf("read failure lost stage, target or original diagnostic: %v", err)
	}
}

func comparisonChange(t *testing.T, path string, before, after any) settings.Change {
	t.Helper()
	field, err := settings.ParseFieldID(path)
	if err != nil {
		t.Fatal(err)
	}
	change, err := settings.NewChange(field, before, after)
	if err != nil {
		t.Fatal(err)
	}
	return change
}
