package app

import (
	"errors"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
	"reflect"
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
