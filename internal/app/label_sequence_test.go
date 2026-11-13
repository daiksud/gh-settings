package app

import (
	"errors"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
	"reflect"
	"strings"
	"testing"
)

func TestApplyLabelSequenceAndWritebackFailureStop(t *testing.T) {
	for _, fail := range []bool{false, true} {
		t.Run(map[bool]string{false: "complete", true: "writeback failure"}[fail], func(t *testing.T) {
			color := "abcdef"
			disabled := false
			labels := appLabelDeclarations(t, settings.LabelDeclaration{Name: "a", Color: &color}, settings.LabelDeclaration{Name: "b", Color: &color})
			var calls []string
			cause := errors.New("settings changed during apply")
			result, err := Apply(func() (settings.Declaration, error) {
				return settings.Declaration{Repository: settings.RepositoryDeclaration{HasIssues: &disabled}, Labels: &labels}, nil
			}, checkoutRepository, Writer{
				Repository: func(repository.Repository, settings.RepositoryDeclaration) error {
					calls = append(calls, "repository")
					return nil
				},
				ReadLabels: func(repository.Repository) (settings.LabelObservation, error) {
					calls = append(calls, "read")
					return settings.NewLabelObservation([]settings.LabelState{{ID: 9, Name: "unmanaged", Color: "abcdef"}})
				},
				CreateLabel: func(_ repository.Repository, c settings.LabelChange) (int64, error) {
					calls = append(calls, "create:"+c.Name())
					if c.Name() == "a" {
						return 17, nil
					}
					return 18, nil
				},
				CreatedID: func(kind, name string, id int64) error {
					calls = append(calls, "save:"+name)
					if kind != "labels" || id <= 0 {
						t.Fatal(kind, name, id)
					}
					if fail {
						return cause
					}
					return nil
				},
				DeleteLabel: func(repository.Repository, settings.LabelChange) error { t.Fatal("unmanaged deletion"); return nil },
			})
			want := []string{"repository", "read", "create:a", "save:a", "create:b", "save:b"}
			if fail {
				want = want[:4]
				if !errors.Is(err, cause) || result.Outcome != ApplyPartialFailure || len(result.CreatedIDs) != 1 || result.CreatedIDs[0].ID != 17 {
					t.Fatal(result, err)
				}
			} else if err != nil || !result.Completed() || len(result.CreatedIDs) != 2 {
				t.Fatal(result, err)
			}
			if !reflect.DeepEqual(calls, want) {
				t.Fatal(calls, want)
			}
		})
	}
}

func TestLabelCreateReadbackFailureRetainsIDWithoutWriteback(t *testing.T) {
	for _, failName := range []string{"a", "b"} {
		t.Run(failName, func(t *testing.T) {
			color := "abcdef"
			labels := appLabelDeclarations(t, settings.LabelDeclaration{Name: "a", Color: &color}, settings.LabelDeclaration{Name: "b", Color: &color}, settings.LabelDeclaration{Name: "c", Color: &color})
			cause := errors.New("read back label: state did not converge")
			var calls []string
			result, err := Apply(func() (settings.Declaration, error) {
				return settings.Declaration{Labels: &labels}, nil
			}, checkoutRepository, Writer{
				ReadLabels: func(repository.Repository) (settings.LabelObservation, error) {
					return settings.NewLabelObservation(nil)
				},
				CreateLabel: func(_ repository.Repository, change settings.LabelChange) (int64, error) {
					calls = append(calls, "create:"+change.Name())
					id := int64(17)
					if change.Name() == "b" {
						id++
					}
					if change.Name() == failName {
						return id, cause
					}
					return id, nil
				},
				CreatedID: func(_, name string, _ int64) error {
					calls = append(calls, "save:"+name)
					return nil
				},
			})
			wantCalls := []string{"create:a"}
			wantOutcome, failedIndex := ApplyFailed, 1
			wantIDs := []CreatedObject{{Collection: "labels", Name: "a", ID: 17}}
			if failName == "b" {
				wantCalls = []string{"create:a", "save:a", "create:b"}
				wantOutcome, failedIndex = ApplyPartialFailure, 2
				wantIDs = append(wantIDs, CreatedObject{Collection: "labels", Name: "b", ID: 18})
			}
			if !errors.Is(err, cause) || !reflect.DeepEqual(calls, wantCalls) || !reflect.DeepEqual(result.CreatedIDs, wantIDs) || result.Outcome != wantOutcome || result.Operations[failedIndex].Status != OperationFailedUnknown {
				t.Fatalf("result=%+v calls=%v err=%v", result, calls, err)
			}
			if strings.Contains(err.Error(), "earlier successful writes remain applied") != (failName == "b") {
				t.Fatal("incorrect confirmed progress", err)
			}
			for _, operation := range result.Operations[failedIndex+1:] {
				if operation.Status != OperationNotExecuted {
					t.Fatal("later write attempted", result)
				}
			}
		})
	}
}

func TestLabelUpdateReadbackFailureStopsCreateAndPrune(t *testing.T) {
	color, id := "abcdef", int64(7)
	labels := appLabelDeclarations(t, settings.LabelDeclaration{ID: &id, Name: "renamed", Color: &color}, settings.LabelDeclaration{Name: "z", Color: &color})
	cause := errors.New("read back label: wrong identity")
	result, err := Apply(func() (settings.Declaration, error) {
		return settings.Declaration{Labels: &labels}, nil
	}, checkoutRepository, Writer{
		ReadLabels: func(repository.Repository) (settings.LabelObservation, error) {
			return settings.NewLabelObservation([]settings.LabelState{{ID: 7, Name: "old", Color: color}, {ID: 9, Name: "remove", Color: color}})
		},
		UpdateLabel: func(repository.Repository, settings.LabelChange) error { return cause },
		CreateLabel: func(repository.Repository, settings.LabelChange) (int64, error) {
			t.Fatal("create after failed verification")
			return 0, nil
		},
		DeleteLabel: func(repository.Repository, settings.LabelChange) error {
			t.Fatal("delete after failed verification")
			return nil
		},
	}, mustPrune(t, "labels"))
	if !errors.Is(err, cause) || result.Outcome != ApplyFailed || len(result.Operations) != 4 || result.Operations[1].Status != OperationFailedUnknown || result.Operations[2].Status != OperationNotExecuted || result.Operations[3].Status != OperationNotExecuted {
		t.Fatal(result, err)
	}
}

func TestLabelIdentityReadFailureIsNotAnUnknownWrite(t *testing.T) {
	color, id := "abcdef", int64(7)
	labels := appLabelDeclarations(t, settings.LabelDeclaration{ID: &id, Name: "renamed", Color: &color})
	cause := errors.New("label identity changed before update")
	result, err := Apply(func() (settings.Declaration, error) {
		return settings.Declaration{Labels: &labels}, nil
	}, checkoutRepository, Writer{
		ReadLabels: func(repository.Repository) (settings.LabelObservation, error) {
			return settings.NewLabelObservation([]settings.LabelState{{ID: 7, Name: "old", Color: "abcdef"}})
		},
		UpdateLabel: func(repository.Repository, settings.LabelChange) error {
			return &workflowReadFailure{cause}
		},
	})
	var readFailure *workflowReadFailure
	if !errors.As(err, &readFailure) || readFailure.error != cause || result.Outcome != ApplyFailed || len(result.Operations) != 2 || result.Operations[1].Status != OperationFailedRead {
		t.Fatalf("result=%+v error=%v", result, err)
	}
}
