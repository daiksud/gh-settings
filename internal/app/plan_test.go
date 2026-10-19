package app

import (
	"errors"
	"reflect"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func TestPlanResults(t *testing.T) {
	after, disabled := "after", false
	target := repository.Repository{Host: "github.com", Owner: "owner", Name: "current"}
	for _, test := range []struct {
		name    string
		current settings.RepositoryState
		desired settings.RepositoryDeclaration
		want    settings.Comparison
	}{
		{"changes", settings.RepositoryState{Description: "before", HasIssues: true}, settings.RepositoryDeclaration{Description: &after, HasIssues: &disabled}, settings.Comparison{Managed: 2, Changes: []settings.Change{comparisonChange(t, "repository.description", "before", "after"), comparisonChange(t, "repository.has_issues", true, false)}}},
		{"no differences", settings.RepositoryState{}, settings.RepositoryDeclaration{HasIssues: &disabled}, settings.Comparison{Managed: 1}},
	} {
		t.Run(test.name, func(t *testing.T) {
			var calls []string
			got, err := Plan(func() (settings.Declaration, error) {
				calls = append(calls, "load")
				return settings.Declaration{Repository: test.desired}, nil
			}, func() (repository.Repository, error) {
				calls = append(calls, "resolve")
				return target, nil
			}, Reader{Repository: func(repo repository.Repository) (settings.RepositoryState, error) {
				calls = append(calls, "read")
				if repo != target {
					t.Fatalf("read target=%+v, want %+v", repo, target)
				}
				return test.current, nil
			}})
			want := PlanResult{Target: target, Comparison: test.want}
			if err != nil || !reflect.DeepEqual(got, want) || !reflect.DeepEqual(calls, []string{"load", "resolve", "read"}) {
				t.Fatalf("result=%+v error=%v calls=%v, want %+v and load/resolve/read", got, err, calls, want)
			}
		})
	}
}

func TestPlanManagedPresence(t *testing.T) {
	topics, labels := []string{}, []settings.LabelDeclaration{}
	for _, test := range []struct {
		name    string
		desired settings.Declaration
		want    []string
	}{
		{"no managed settings", settings.Declaration{}, []string{"load"}},
		{"empty topics", settings.Declaration{Topics: settings.TopicsDeclaration{Names: &topics}}, []string{"load", "resolve", "topics"}},
		{"empty labels", settings.Declaration{Labels: &labels}, []string{"load", "resolve", "labels"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			var calls []string
			target := repository.Repository{Host: "github.com", Owner: "owner", Name: "current"}
			got, err := Plan(func() (settings.Declaration, error) {
				calls = append(calls, "load")
				return test.desired, nil
			}, func() (repository.Repository, error) {
				calls = append(calls, "resolve")
				return target, nil
			}, Reader{Topics: func(repo repository.Repository) ([]string, error) {
				calls = append(calls, "topics")
				if repo != target {
					t.Fatalf("target=%+v", repo)
				}
				return nil, nil
			}, Labels: func(repo repository.Repository) ([]settings.LabelState, error) {
				calls = append(calls, "labels")
				if repo != target {
					t.Fatalf("target=%+v", repo)
				}
				return nil, nil
			}})
			if err != nil || !reflect.DeepEqual(calls, test.want) {
				t.Fatalf("result=%+v error=%v calls=%v, want %v", got, err, calls, test.want)
			}
			if test.desired.Empty() {
				if !reflect.DeepEqual(got, PlanResult{}) {
					t.Fatalf("no-op result=%+v", got)
				}
			} else if got.Target != target || got.Comparison.Managed != 1 {
				t.Fatalf("managed empty result=%+v", got)
			}
		})
	}
}

func TestPlanFailuresStopBeforeLaterStages(t *testing.T) {
	for _, stage := range []string{"invalid-config", "target-failure", "read-failure"} {
		t.Run(stage, func(t *testing.T) {
			cause := errors.New("original diagnostic")
			target := repository.Repository{Host: "github.com", Owner: "owner", Name: "current"}
			disabled := false
			var calls []string
			got, err := Plan(func() (settings.Declaration, error) {
				calls = append(calls, "load")
				if stage == "invalid-config" {
					return settings.Declaration{}, cause
				}
				return settings.Declaration{Repository: settings.RepositoryDeclaration{HasIssues: &disabled}}, nil
			}, func() (repository.Repository, error) {
				calls = append(calls, "resolve")
				if stage == "target-failure" {
					return repository.Repository{}, cause
				}
				return target, nil
			}, Reader{Repository: func(repository.Repository) (settings.RepositoryState, error) {
				calls = append(calls, "read")
				return settings.RepositoryState{}, cause
			}})
			wantCalls := []string{"load"}
			if stage != "invalid-config" {
				wantCalls = append(wantCalls, "resolve")
			}
			wantTarget := repository.Repository{}
			if stage == "read-failure" {
				wantCalls = append(wantCalls, "read")
				wantTarget = target
			}
			var failure *PlanError
			if !errors.As(err, &failure) || failure.Stage != stage || failure.Target != wantTarget || !errors.Is(err, cause) || err.Error() != cause.Error() || !reflect.DeepEqual(got, PlanResult{}) || !reflect.DeepEqual(calls, wantCalls) {
				t.Fatalf("result=%+v error=%v calls=%v, want %s after %v", got, err, calls, stage, wantCalls)
			}
		})
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
