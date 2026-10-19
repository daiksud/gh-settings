package app

import (
	"errors"
	"reflect"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func TestPlanResults(t *testing.T) {
	disabled, after := false, "after"
	for _, test := range []struct {
		name    string
		desired settings.RepositoryDeclaration
		current settings.RepositoryState
		want    settings.Comparison
	}{
		{"changes", settings.RepositoryDeclaration{Description: &after, HasIssues: &disabled}, settings.RepositoryState{Description: "before", HasIssues: true}, settings.Comparison{Managed: 2, Changes: []settings.Change{comparisonChange(t, "repository.description", "before", "after"), comparisonChange(t, "repository.has_issues", true, false)}}},
		{"no differences", settings.RepositoryDeclaration{HasIssues: &disabled}, settings.RepositoryState{}, settings.Comparison{Managed: 1}},
	} {
		t.Run(test.name, func(t *testing.T) {
			target := repository.Repository{Host: "github.com", Owner: "owner", Name: "current"}
			var calls []string
			got, err := Plan(func() (settings.Declaration, error) {
				calls = append(calls, "load")
				return settings.Declaration{Repository: test.desired}, nil
			}, func() (repository.Repository, error) {
				calls = append(calls, "target")
				return target, nil
			}, Reader{ObserveRepository: observeBaseFixture(func(repo repository.Repository) (settings.RepositoryState, error) {
				calls = append(calls, "repository")
				if repo != target {
					t.Fatalf("read target=%+v, want %+v", repo, target)
				}
				return test.current, nil
			})})
			want := PlanResult{Target: target, Comparison: test.want}
			if err != nil || !reflect.DeepEqual(got, want) || !reflect.DeepEqual(calls, []string{"load", "target", "repository"}) {
				t.Fatalf("result=%+v error=%v calls=%v, want %+v after load, target and read", got, err, calls, want)
			}
		})
	}
}

func TestPlanNoManagedSettings(t *testing.T) {
	var calls []string
	got, err := Plan(func() (settings.Declaration, error) {
		calls = append(calls, "load")
		return settings.Declaration{}, nil
	}, func() (repository.Repository, error) {
		calls = append(calls, "target")
		return repository.Repository{}, nil
	}, Reader{})
	if err != nil || !reflect.DeepEqual(got, PlanResult{}) || !reflect.DeepEqual(calls, []string{"load"}) {
		t.Fatalf("result=%+v error=%v calls=%v, want local no-op after loading", got, err, calls)
	}
}

func TestPlanManagedEmptyCollectionsResolveTarget(t *testing.T) {
	names, labels := []string{}, settings.LabelDeclarations{}
	for _, test := range []struct {
		name    string
		desired settings.Declaration
	}{
		{"topics", settings.Declaration{Topics: settings.TopicsDeclaration{Names: &names}}},
		{"labels", settings.Declaration{Labels: &labels}},
	} {
		t.Run(test.name, func(t *testing.T) {
			target := repository.Repository{Host: "github.com", Owner: "owner", Name: "current"}
			var calls []string
			record := func(name string, repo repository.Repository) {
				calls = append(calls, name)
				if repo != target {
					t.Fatalf("read target=%+v, want %+v", repo, target)
				}
			}
			got, err := Plan(func() (settings.Declaration, error) {
				calls = append(calls, "load")
				return test.desired, nil
			}, func() (repository.Repository, error) {
				calls = append(calls, "target")
				return target, nil
			}, Reader{
				Topics: func(repo repository.Repository) ([]string, error) {
					record("topics", repo)
					return []string{}, nil
				},
				Labels: func(repo repository.Repository) ([]settings.LabelState, error) {
					record("labels", repo)
					return []settings.LabelState{}, nil
				},
			})
			if err != nil || got.Target != target || got.Comparison.Managed != 1 || !reflect.DeepEqual(calls, []string{"load", "target", test.name}) {
				t.Fatalf("result=%+v error=%v calls=%v, want managed empty collection after target resolution", got, err, calls)
			}
		})
	}
}

func TestPlanValidatesBeforeRead(t *testing.T) {
	loadError := errors.New("validate .github/settings.yml: declaration rejected")
	var calls []string
	got, err := Plan(func() (settings.Declaration, error) {
		calls = append(calls, "load")
		disabled := false
		return settings.Declaration{Repository: settings.RepositoryDeclaration{HasIssues: &disabled}}, loadError
	}, func() (repository.Repository, error) {
		calls = append(calls, "target")
		return repository.Repository{}, nil
	}, Reader{})
	var failure *PlanError
	if !errors.As(err, &failure) || failure.Stage != "invalid-config" || !errors.Is(err, loadError) || err.Error() != loadError.Error() || !reflect.DeepEqual(got, PlanResult{}) || !reflect.DeepEqual(calls, []string{"load"}) {
		t.Fatalf("result=%+v error=%v calls=%v, want original validation failure before target/read", got, err, calls)
	}
}

func TestPlanTargetFailureBeforeRead(t *testing.T) {
	targetError := errors.New("resolve current repository: a GitHub origin remote is required: missing origin")
	var calls []string
	got, err := Plan(func() (settings.Declaration, error) {
		calls = append(calls, "load")
		disabled := false
		return settings.Declaration{Repository: settings.RepositoryDeclaration{HasIssues: &disabled}}, nil
	}, func() (repository.Repository, error) {
		calls = append(calls, "target")
		return repository.Repository{Host: "github.com", Owner: "partial", Name: "target"}, targetError
	}, Reader{})
	var failure *PlanError
	if !errors.As(err, &failure) || failure.Stage != "target-failure" || failure.Target != (repository.Repository{}) || !errors.Is(err, targetError) || err.Error() != targetError.Error() || !reflect.DeepEqual(got, PlanResult{}) || !reflect.DeepEqual(calls, []string{"load", "target"}) {
		t.Fatalf("result=%+v error=%v calls=%v, want original target failure before read", got, err, calls)
	}
}

func TestPlanReadFailure(t *testing.T) {
	readError := errors.New("read unavailable")
	target := repository.Repository{Host: "github.com", Owner: "owner", Name: "current"}
	var calls []string
	got, err := Plan(func() (settings.Declaration, error) {
		calls = append(calls, "load")
		disabled := false
		return settings.Declaration{Repository: settings.RepositoryDeclaration{HasIssues: &disabled}}, nil
	}, func() (repository.Repository, error) {
		calls = append(calls, "target")
		return target, nil
	}, Reader{ObserveRepository: observeBaseFixture(func(repo repository.Repository) (settings.RepositoryState, error) {
		calls = append(calls, "repository")
		if repo != target {
			t.Fatalf("read target=%+v, want %+v", repo, target)
		}
		return settings.RepositoryState{}, readError
	})})
	if !errors.Is(err, readError) || !reflect.DeepEqual(got, PlanResult{}) || !reflect.DeepEqual(calls, []string{"load", "target", "repository"}) {
		t.Fatalf("result=%+v error=%v calls=%v, want read error and zero result", got, err, calls)
	}
	var failure *PlanError
	if !errors.As(err, &failure) || failure.Stage != "read-failure" || failure.Target != target || err.Error() != readError.Error() {
		t.Fatalf("read failure lost stage, target or original diagnostic: %v", err)
	}
}

func comparisonChange(t *testing.T, path string, before, after any) settings.Change {
	t.Helper()
	field, err := settings.ParseFieldID(path)
	if err != nil {
		t.Fatal(err)
	}
	var change settings.Change
	switch before := before.(type) {
	case bool:
		change, err = settings.NewBoolChange(field, before, after.(bool))
	case string:
		change, err = settings.NewStringChange(field, before, after.(string))
	case []string:
		if field != settings.TopicsField() {
			t.Fatal("topic values require the topics field")
		}
		change, err = settings.NewTopicsChange(before, after.([]string))
	default:
		t.Fatalf("unsupported comparison fixture type %T", before)
	}
	if err != nil {
		t.Fatal(err)
	}
	return change
}

func comparisonAfterBool(t *testing.T, change settings.Change) bool {
	t.Helper()
	var result bool
	change.Visit(func(_, after bool) { result = after },
		func(string, string) { t.Fatal("expected boolean comparison") },
		func([]string, []string) { t.Fatal("expected boolean comparison") })
	return result
}
