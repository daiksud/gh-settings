package app

import (
	"errors"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
	"reflect"
	"strings"
	"testing"
)

func TestTopicsApplyOrderAndPartialFailures(t *testing.T) {
	for _, fail := range []string{"", "repository", "topics"} {
		t.Run(fail, func(t *testing.T) {
			disabled, names := false, []string{}
			target := repository.Repository{Host: "github.com", Owner: "owner", Name: "current"}
			var calls []string
			failure := errors.New("write failure")
			result, err := Apply(func() (settings.Declaration, error) {
				calls = append(calls, "load")
				return settings.Declaration{Repository: settings.RepositoryDeclaration{HasIssues: &disabled}, Topics: settings.TopicsDeclaration{Names: &names}}, nil
			}, func() (repository.Repository, error) {
				calls = append(calls, "target")
				return target, nil
			}, Writer{
				Repository: func(repo repository.Repository, d settings.RepositoryDeclaration) (bool, error) {
					calls = append(calls, "repository")
					if repo != target || d.HasIssues == nil || *d.HasIssues {
						t.Fatal("lost false")
					}
					if fail == "repository" {
						return false, failure
					}
					return true, nil
				},
				Topics: func(repo repository.Repository, d settings.TopicsDeclaration) (bool, error) {
					calls = append(calls, "topics")
					if repo != target || d.Names == nil || len(*d.Names) != 0 {
						t.Fatal("lost clear")
					}
					if fail == "topics" {
						return false, failure
					}
					return true, nil
				},
			})
			want := []string{"load", "target", "repository", "topics"}
			if fail == "repository" {
				want = want[:3]
			}
			if !reflect.DeepEqual(calls, want) || (fail != "" && !errors.Is(err, failure)) || (fail == "" && (err != nil || !result.Completed())) || (fail != "" && result.Completed()) {
				t.Fatalf("calls=%v result=%+v err=%v", calls, result, err)
			}
		})
	}
}
func TestTopicsOnlyPlanAndApply(t *testing.T) {
	for _, names := range []string{"[]", "[b, a]"} {
		t.Run(names, func(t *testing.T) {
			appCheckout(t, "version: daiksud/gh-settings@v0\nrepository: {about: {topics: "+names+"}}", true)
			result, err := Plan(checkoutDeclaration, checkoutRepository, Reader{Topics: func(repository.Repository) ([]string, error) { return []string{"a", "b"}, nil }})
			if err != nil || result.Comparison.Managed != 1 {
				t.Fatalf("result=%+v error=%v", result, err)
			}
			changes := 0
			if names == "[]" {
				changes = 1
			}
			if len(result.Comparison.Changes) != changes {
				t.Fatalf("comparison=%+v", result.Comparison)
			}
			calls := 0
			applied, err := Apply(checkoutDeclaration, checkoutRepository, Writer{Topics: func(repository.Repository, settings.TopicsDeclaration) (bool, error) { calls++; return true, nil }})
			if err != nil || !applied.Completed() || calls != 1 {
				t.Fatalf("result=%+v calls=%d error=%v", applied, calls, err)
			}
		})
	}
}
func TestEmptyTopicsIsLocalNoop(t *testing.T) {
	appCheckout(t, "version: daiksud/gh-settings@v0\nrepository: {about: {}}", false)
	if _, err := Plan(checkoutDeclaration, checkoutRepository, Reader{}); err != nil {
		t.Fatal(err)
	}
	if _, err := Apply(checkoutDeclaration, checkoutRepository, Writer{}); err != nil {
		t.Fatal(err)
	}
}
func TestInvalidTopicsBeforeRemote(t *testing.T) {
	appCheckout(t, "version: daiksud/gh-settings@v0\nrepository: {about: {topics: [null]}}", false)
	if _, err := Plan(checkoutDeclaration, checkoutRepository, Reader{}); err == nil || !strings.Contains(err.Error(), "repository.about.topics must contain strings") {
		t.Fatalf("error=%v", err)
	}
	if _, err := Apply(checkoutDeclaration, checkoutRepository, Writer{}); err == nil || !strings.Contains(err.Error(), "repository.about.topics must contain strings") {
		t.Fatalf("error=%v", err)
	}
}

type topicsReadFailure struct{ error }

func (*topicsReadFailure) ReadFailedBeforeWrite() {}

func TestTopicsApplyReportsActualWritesAndReadFailures(t *testing.T) {
	for _, test := range []struct {
		name                                                                   string
		wrote, prewrite, unknown, earlierRepository, repositoryNoop, unarchive bool
		wantOutcome                                                            ApplyOutcome
		wantStatus                                                             ApplyOperationStatus
	}{
		{name: "matching", wantOutcome: ApplyMatchingLabels, wantStatus: OperationUnchanged},
		{name: "changed", wrote: true, wantOutcome: ApplyCompleted, wantStatus: OperationCompleted},
		{name: "preflight failed", prewrite: true, wantOutcome: ApplyFailed, wantStatus: OperationFailedRead},
		{name: "write uncertain", unknown: true, wantOutcome: ApplyFailed, wantStatus: OperationFailedUnknown},
		{name: "earlier repository write", earlierRepository: true, prewrite: true, wantOutcome: ApplyPartialFailure, wantStatus: OperationFailedRead},
		{name: "matching repository then matching topics", repositoryNoop: true, wantOutcome: ApplyMatchingLabels, wantStatus: OperationUnchanged},
		{name: "matching repository then changed topics", repositoryNoop: true, wrote: true, wantOutcome: ApplyCompleted, wantStatus: OperationCompleted},
		{name: "matching repository then topics read failure", repositoryNoop: true, prewrite: true, wantOutcome: ApplyFailed, wantStatus: OperationFailedRead},
		{name: "earlier unarchive", unarchive: true, prewrite: true, wantOutcome: ApplyPartialFailure, wantStatus: OperationFailedRead},
	} {
		t.Run(test.name, func(t *testing.T) {
			names, disabled := []string{}, false
			desired := settings.Declaration{Topics: settings.TopicsDeclaration{Names: &names}}
			if test.earlierRepository || test.repositoryNoop {
				desired.Repository.HasIssues = &disabled
			}
			if test.unarchive {
				desired.Repository.Archived = &disabled
			}
			calls := []string{}
			cause := errors.New("topics unavailable")
			result, err := Apply(func() (settings.Declaration, error) { return desired, nil }, func() (repository.Repository, error) {
				return repository.Repository{Owner: "owner", Name: "current"}, nil
			}, Writer{
				Repository: func(repository.Repository, settings.RepositoryDeclaration) (bool, error) {
					calls = append(calls, "repository")
					return !test.repositoryNoop, nil
				},
				ReadArchive: func(repository.Repository) (settings.ArchiveState, error) {
					return settings.ArchiveState{ID: 2, FullName: "owner/current", Archived: true}, nil
				},
				Archive: func(repository.Repository, settings.ArchiveState, bool) error {
					calls = append(calls, "unarchive")
					return nil
				},
				Topics: func(repository.Repository, settings.TopicsDeclaration) (bool, error) {
					calls = append(calls, "topics")
					if test.prewrite {
						return false, &topicsReadFailure{cause}
					}
					if test.unknown {
						return false, cause
					}
					return test.wrote, nil
				},
			})
			last := result.Operations[len(result.Operations)-1]
			if result.Outcome != test.wantOutcome || last.Kind != TopicsUpdate || last.Status != test.wantStatus || (err != nil) != (test.prewrite || test.unknown) {
				t.Fatal(result, err)
			}
			wantCalls := []string{"topics"}
			if test.earlierRepository || test.repositoryNoop {
				wantCalls = []string{"repository", "topics"}
			}
			if test.repositoryNoop {
				if result.Operations[0].Kind != RepositoryUpdate || result.Operations[0].Status != OperationUnchanged || err != nil && strings.Contains(err.Error(), "earlier repository settings update remains applied") {
					t.Fatal("matching repository reported prior writes", result, err)
				}
			}
			if test.unarchive {
				wantCalls = []string{"unarchive", "topics"}
			}
			if !reflect.DeepEqual(calls, wantCalls) {
				t.Fatal(calls)
			}
		})
	}
}

func TestTopicsNoopContinuesLabelsAndFinalArchive(t *testing.T) {
	for _, mode := range []string{"matching labels", "label read failure", "matching repository and label read failure", "label comparison failure", "archive"} {
		t.Run(mode, func(t *testing.T) {
			names, id, color, archived := []string{}, int64(7), "abcdef", true
			labels := appLabelDeclarations(t, settings.LabelDeclaration{ID: &id, Name: "bug", Color: &color})
			desired := settings.Declaration{Topics: settings.TopicsDeclaration{Names: &names}, Labels: &labels}
			if mode == "matching repository and label read failure" {
				desired.Repository.HasIssues = &archived
			}
			if mode == "archive" {
				desired.Repository.Archived = &archived
			}
			var calls []string
			result, err := Apply(func() (settings.Declaration, error) { return desired, nil }, func() (repository.Repository, error) {
				return repository.Repository{Owner: "owner", Name: "current"}, nil
			}, Writer{
				Repository: func(repository.Repository, settings.RepositoryDeclaration) (bool, error) {
					calls = append(calls, "repository")
					return false, nil
				},
				Topics: func(repository.Repository, settings.TopicsDeclaration) (bool, error) {
					calls = append(calls, "topics")
					return false, nil
				},
				ReadLabels: func(repository.Repository) (settings.LabelObservation, error) {
					calls = append(calls, "labels")
					if strings.Contains(mode, "label read failure") {
						return settings.LabelObservation{}, errors.New("read denied")
					}
					if mode == "label comparison failure" {
						return settings.LabelObservation{}, nil
					}
					return settings.NewLabelObservation([]settings.LabelState{{ID: id, Name: "bug", Color: color}})
				},
				ReadArchive: func(repository.Repository) (settings.ArchiveState, error) {
					calls = append(calls, "archive-read")
					return settings.ArchiveState{ID: 2, FullName: "owner/current"}, nil
				},
				Archive: func(_ repository.Repository, _ settings.ArchiveState, desired bool) error {
					if !desired {
						t.Fatal("unexpected unarchive")
					}
					calls = append(calls, "archive")
					return nil
				},
			})
			wantOutcome, wantCalls := ApplyMatchingLabels, []string{"topics", "labels"}
			if mode == "matching repository and label read failure" {
				wantCalls = []string{"repository", "topics", "labels"}
			}
			if strings.Contains(mode, "failure") {
				wantOutcome = ApplyFailed
				if err == nil || strings.Contains(err.Error(), "earlier resource writes") {
					t.Fatal(err)
				}
			} else if err != nil {
				t.Fatal(err)
			}
			if mode == "archive" {
				wantOutcome, wantCalls = ApplyCompleted, []string{"archive-read", "topics", "labels", "archive-read", "archive"}
			}
			if result.Outcome != wantOutcome || !reflect.DeepEqual(calls, wantCalls) {
				t.Fatal(result, calls)
			}
			for _, operation := range result.Operations {
				if operation.Kind == TopicsUpdate && operation.Status != OperationUnchanged {
					t.Fatal(operation)
				}
			}
		})
	}
}

func TestTopicsWriteThenLabelsReadFailureIsPartial(t *testing.T) {
	names, color := []string{}, "abcdef"
	labels := appLabelDeclarations(t, settings.LabelDeclaration{Name: "bug", Color: &color})
	cause := errors.New("labels denied")
	result, err := Apply(func() (settings.Declaration, error) {
		return settings.Declaration{Topics: settings.TopicsDeclaration{Names: &names}, Labels: &labels}, nil
	}, func() (repository.Repository, error) {
		return repository.Repository{Owner: "owner", Name: "current"}, nil
	}, Writer{
		Topics: func(repository.Repository, settings.TopicsDeclaration) (bool, error) { return true, nil },
		ReadLabels: func(repository.Repository) (settings.LabelObservation, error) {
			return settings.LabelObservation{}, cause
		},
	})
	if !errors.Is(err, cause) || !strings.Contains(err.Error(), "earlier resource writes remain applied") || result.Outcome != ApplyPartialFailure || !reflect.DeepEqual(result.Operations, []ApplyOperation{{Kind: TopicsUpdate, Status: OperationCompleted}, {Kind: LabelsRead, Status: OperationFailedRead}}) {
		t.Fatal(result, err)
	}
}
