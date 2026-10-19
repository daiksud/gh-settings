package app

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func TestLabelsPlanAndApplyEmptyListDeletesAll(t *testing.T) {
	appCheckout(t, "schemaVersion: 1\nsettings: {labels: []}\n", true)
	target := repository.Repository{Host: "github.com", Owner: "owner", Name: "current"}
	state := []settings.LabelState{{Name: "assigned", Color: "abcdef", Description: "old"}}
	reads := 0
	read := func(repo repository.Repository) ([]settings.LabelState, error) {
		if repo != target {
			t.Fatalf("target=%+v", repo)
		}
		reads++
		return state, nil
	}
	plan, err := Plan(checkoutDeclaration, checkoutRepository, Reader{Labels: read})
	if err != nil || plan.Comparison.Managed != 1 || len(plan.Comparison.Labels) != 1 || plan.Comparison.Labels[0].Operation != "delete" {
		t.Fatalf("plan=%+v err=%v", plan, err)
	}
	calls := 0
	applied, err := Apply(checkoutDeclaration, checkoutRepository, Writer{ReadLabels: read, Labels: func(repo repository.Repository, changes []settings.LabelChange) error {
		calls++
		if repo != target || !reflect.DeepEqual(changes, plan.Comparison.Labels) {
			t.Fatalf("changes=%+v", changes)
		}
		return nil
	}})
	if err != nil || !applied.Applied || calls != 1 || reads != 2 {
		t.Fatalf("result=%+v reads=%d writes=%d err=%v", applied, reads, calls, err)
	}
}

func TestLabelsApplyReadsFreshStateAndSkipsUnchangedWrites(t *testing.T) {
	labels := []settings.LabelDeclaration{{Name: "bug"}}
	target := repository.Repository{Host: "github.com", Owner: "owner", Name: "current"}
	var calls []string
	result, err := Apply(func() (settings.Declaration, error) {
		calls = append(calls, "load")
		return settings.Declaration{Labels: &labels}, nil
	}, func() (repository.Repository, error) {
		calls = append(calls, "target")
		return target, nil
	}, Writer{ReadLabels: func(repo repository.Repository) ([]settings.LabelState, error) {
		calls = append(calls, "labels-read")
		if repo != target {
			t.Fatalf("target=%+v", repo)
		}
		return []settings.LabelState{{Name: "BUG", Color: "abcdef", Description: "external edit"}}, nil
	}})
	if err != nil || !result.Applied || !reflect.DeepEqual(calls, []string{"load", "target", "labels-read"}) {
		t.Fatalf("result=%+v err=%v calls=%v", result, err, calls)
	}
}

func TestInvalidLabelsBeforeTargetAndRemoteAccess(t *testing.T) {
	for _, input := range []string{"null", "[{name: bug}, {name: BUG}]", "[{name: bug, color: false}]"} {
		t.Run(input, func(t *testing.T) {
			appCheckout(t, "schemaVersion: 1\nsettings: {labels: "+input+"}", false)
			if _, err := Plan(checkoutDeclaration, checkoutRepository, Reader{}); err == nil {
				t.Fatal("plan accepted invalid labels")
			}
			if _, err := Apply(checkoutDeclaration, checkoutRepository, Writer{}); err == nil {
				t.Fatal("apply accepted invalid labels")
			}
		})
	}
	appCheckout(t, "schemaVersion: 1\nsettings: {labels: []}", false)
	if _, err := Apply(checkoutDeclaration, checkoutRepository, Writer{}); err == nil || !strings.Contains(err.Error(), "origin") {
		t.Fatalf("empty list lost managed presence: %v", err)
	}
}

func TestLabelsMixedApplyFailureOrder(t *testing.T) {
	for _, failureStage := range []string{"", "repository", "topics", "labels-read", "labels-write"} {
		t.Run(failureStage, func(t *testing.T) {
			disabled, names, labels := false, []string{}, []settings.LabelDeclaration{}
			target := repository.Repository{Host: "github.com", Owner: "owner", Name: "current"}
			failure := errors.New("fixture failure")
			var calls []string
			step := func(name string, repo repository.Repository) error {
				if repo != target {
					t.Fatalf("target=%+v", repo)
				}
				calls = append(calls, name)
				if failureStage == name {
					return failure
				}
				return nil
			}
			result, err := Apply(func() (settings.Declaration, error) {
				calls = append(calls, "load")
				return settings.Declaration{Repository: settings.RepositoryDeclaration{HasIssues: &disabled}, Topics: settings.TopicsDeclaration{Names: &names}, Labels: &labels}, nil
			}, func() (repository.Repository, error) {
				calls = append(calls, "target")
				return target, nil
			}, Writer{
				Repository: func(repo repository.Repository, d settings.RepositoryDeclaration) error {
					if d.HasIssues == nil || *d.HasIssues {
						t.Fatalf("declaration=%+v", d)
					}
					return step("repository", repo)
				},
				Topics: func(repo repository.Repository, d settings.TopicsDeclaration) error {
					if d.Names == nil || len(*d.Names) != 0 {
						t.Fatalf("topics=%+v", d)
					}
					return step("topics", repo)
				},
				ReadLabels: func(repo repository.Repository) ([]settings.LabelState, error) {
					err := step("labels-read", repo)
					return []settings.LabelState{{Name: "old", Color: "abcdef"}}, err
				},
				Labels: func(repo repository.Repository, changes []settings.LabelChange) error {
					if len(changes) != 1 || changes[0].Operation != "delete" || changes[0].Name != "old" {
						t.Fatalf("changes=%+v", changes)
					}
					return step("labels-write", repo)
				},
			})
			want := []string{"load", "target", "repository", "topics", "labels-read", "labels-write"}
			if failureStage != "" {
				for i, name := range want {
					if name == failureStage {
						want = want[:i+1]
						break
					}
				}
			}
			if !reflect.DeepEqual(calls, want) || (failureStage != "" && !errors.Is(err, failure)) || (failureStage == "" && (err != nil || !result.Applied)) || (failureStage != "" && result.Applied) {
				t.Fatalf("calls=%v result=%+v err=%v", calls, result, err)
			}
		})
	}
}

func TestLabelsPlanFailureDoesNotExposeEarlierComparison(t *testing.T) {
	appCheckout(t, "schemaVersion: 1\nsettings:\n  repository: {has_issues: false}\n  labels: []\n", true)
	for _, ambiguous := range []bool{false, true} {
		result, err := Plan(checkoutDeclaration, checkoutRepository, Reader{
			ObserveRepository: observeBaseFixture(func(repository.Repository) (settings.RepositoryState, error) {
				return settings.RepositoryState{HasIssues: true}, nil
			}),
			Labels: func(repository.Repository) ([]settings.LabelState, error) {
				if !ambiguous {
					return nil, errors.New("later page failed")
				}
				return []settings.LabelState{{Name: "bug", Color: "abcdef"}, {Name: "BUG", Color: "abcdef"}}, nil
			},
		})
		var failure *PlanError
		if !errors.As(err, &failure) || failure.Stage != "read-failure" || failure.Target.Owner != "owner" || result.Comparison.Managed != 0 || len(result.Comparison.Changes) != 0 || len(result.Comparison.Labels) != 0 {
			t.Fatalf("result=%+v error=%v", result, err)
		}
	}
}
