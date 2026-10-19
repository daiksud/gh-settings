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
	read := func(repo repository.Repository) (settings.LabelObservation, error) {
		if repo != target {
			t.Fatalf("target=%+v", repo)
		}
		reads++
		return settings.NewLabelObservation(state)
	}
	plan, err := Plan(checkoutDeclaration, checkoutRepository, Reader{Labels: read})
	if err != nil || plan.Comparison.Managed != 1 || plan.Comparison.Labels.Len() != 1 || plan.Comparison.Labels.Values()[0].Operation() != "delete" {
		t.Fatalf("plan=%+v err=%v", plan, err)
	}
	calls := 0
	applied, err := Apply(checkoutDeclaration, checkoutRepository, Writer{ReadLabels: read, DeleteLabel: func(repo repository.Repository, change settings.LabelChange) error {
		calls++
		if repo != target || !reflect.DeepEqual(change, plan.Comparison.Labels.Values()[0]) {
			t.Fatalf("change=%+v", change)
		}
		return nil
	}})
	if err != nil || !applied.Completed() || calls != 1 || reads != 2 {
		t.Fatalf("result=%+v reads=%d writes=%d err=%v", applied, reads, calls, err)
	}
}

func TestLabelsApplyReadsFreshStateAndSkipsUnchangedWrites(t *testing.T) {
	labels := appLabelDeclarations(t, settings.LabelDeclaration{Name: "bug"})
	target := repository.Repository{Host: "github.com", Owner: "owner", Name: "current"}
	var calls []string
	result, err := Apply(func() (settings.Declaration, error) {
		calls = append(calls, "load")
		return settings.Declaration{Labels: &labels}, nil
	}, func() (repository.Repository, error) {
		calls = append(calls, "target")
		return target, nil
	}, Writer{ReadLabels: func(repo repository.Repository) (settings.LabelObservation, error) {
		calls = append(calls, "labels-read")
		if repo != target {
			t.Fatalf("target=%+v", repo)
		}
		return settings.NewLabelObservation([]settings.LabelState{{Name: "BUG", Color: "abcdef", Description: "external edit"}})
	}})
	if err != nil || !result.Completed() || !reflect.DeepEqual(result, ApplyResult{Outcome: ApplyMatchingLabels, Target: target, Operations: []ApplyOperation{{Kind: LabelsRead, Status: OperationCompleted}}}) || !reflect.DeepEqual(calls, []string{"load", "target", "labels-read"}) {
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
			disabled, names, labels := false, []string{}, settings.LabelDeclarations{}
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
				ReadLabels: func(repo repository.Repository) (settings.LabelObservation, error) {
					err := step("labels-read", repo)
					if err != nil {
						return settings.LabelObservation{}, err
					}
					return settings.NewLabelObservation([]settings.LabelState{{Name: "old", Color: "abcdef"}})
				},
				DeleteLabel: func(repo repository.Repository, change settings.LabelChange) error {
					if change.Operation() != "delete" || change.Name() != "old" {
						t.Fatalf("change=%+v", change)
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
			if failureStage == "labels-read" {
				expected := []ApplyOperation{{Kind: RepositoryUpdate, Status: OperationCompleted}, {Kind: TopicsUpdate, Status: OperationCompleted}, {Kind: LabelsRead, Status: OperationFailedRead}}
				if result.Outcome != ApplyPartialFailure || !reflect.DeepEqual(result.Operations, expected) {
					t.Fatalf("read failure progress=%+v", result)
				}
			}
			if !reflect.DeepEqual(calls, want) || (failureStage != "" && !errors.Is(err, failure)) || (failureStage == "" && (err != nil || !result.Completed())) || (failureStage != "" && result.Completed()) {
				t.Fatalf("calls=%v result=%+v err=%v", calls, result, err)
			}
		})
	}
}

func TestLabelsPlanFailureDoesNotExposeEarlierComparison(t *testing.T) {
	appCheckout(t, "schemaVersion: 1\nsettings:\n  repository: {has_issues: false}\n  labels: []\n", true)
	for _, failureMode := range []string{"later-page", "ambiguous", "missing"} {
		result, err := Plan(checkoutDeclaration, checkoutRepository, Reader{
			ObserveRepository: observeBaseFixture(func(repository.Repository) (settings.RepositoryState, error) {
				return settings.RepositoryState{HasIssues: true}, nil
			}),
			Labels: func(repository.Repository) (settings.LabelObservation, error) {
				if failureMode == "later-page" {
					return settings.LabelObservation{}, errors.New("later page failed")
				}
				if failureMode == "missing" {
					return settings.LabelObservation{}, nil
				}
				return settings.NewLabelObservation([]settings.LabelState{{Name: "bug", Color: "abcdef"}, {Name: "BUG", Color: "abcdef"}})
			},
		})
		var failure *PlanError
		if !errors.As(err, &failure) || failure.Stage != "read-failure" || failure.Target.Owner != "owner" || result.Comparison.Managed != 0 || len(result.Comparison.Changes) != 0 || result.Comparison.Labels.Len() != 0 {
			t.Fatalf("result=%+v error=%v", result, err)
		}
	}
}

func appLabelDeclarations(t *testing.T, values ...settings.LabelDeclaration) settings.LabelDeclarations {
	t.Helper()
	result, err := settings.NewLabelDeclarations(values)
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func TestMissingLabelObservationPreventsExportAndApplyWrites(t *testing.T) {
	appCheckout(t, "schemaVersion: 1\nsettings: {labels: []}\n", true)
	missing := func(repository.Repository) (settings.LabelObservation, error) {
		return settings.LabelObservation{}, nil
	}
	result, err := Apply(checkoutDeclaration, checkoutRepository, Writer{
		ReadLabels: missing,
		DeleteLabel: func(repository.Repository, settings.LabelChange) error {
			t.Fatal("label writes after missing observation")
			return nil
		},
	})
	if err == nil || result.Completed() || result.Outcome != ApplyFailed || !reflect.DeepEqual(result.Operations, []ApplyOperation{{Kind: LabelsRead, Status: OperationFailedRead}}) {
		t.Fatalf("apply=%+v error=%v", result, err)
	}
	read := exportFixtureReader(t, settings.RepositoryState{}, func(string, repository.Repository) error { return nil })
	read.Labels = missing
	err = Export(ExportCheckout{
		Repository: checkoutRepository,
		PrepareSave: func() (func(settings.Declaration) error, error) {
			return func(settings.Declaration) error {
				t.Fatal("save after missing observation")
				return nil
			}, nil
		},
	}, read)
	if err == nil {
		t.Fatal("export accepted missing observation")
	}
}
