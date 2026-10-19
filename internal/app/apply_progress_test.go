package app

import (
	"errors"
	"reflect"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func TestTopicsFailureRetainsKnownApplyProgress(t *testing.T) {
	target := repository.Repository{Host: "github.com", Owner: "owner", Name: "current"}
	disabled, names, labels := false, []string{}, settings.LabelDeclarations{}
	cause := errors.New("connection lost after request")
	var calls []string
	result, err := Apply(func() (settings.Declaration, error) {
		calls = append(calls, "load")
		return settings.Declaration{Repository: settings.RepositoryDeclaration{HasIssues: &disabled}, Topics: settings.TopicsDeclaration{Names: &names}, Labels: &labels}, nil
	}, func() (repository.Repository, error) { calls = append(calls, "target"); return target, nil }, Writer{
		Repository: func(repository.Repository, settings.RepositoryDeclaration) error {
			calls = append(calls, "repository")
			return nil
		},
		Topics: func(repository.Repository, settings.TopicsDeclaration) error {
			calls = append(calls, "topics")
			return cause
		},
		ReadLabels: func(repository.Repository) (settings.LabelObservation, error) {
			t.Fatal("read after failed update")
			return settings.LabelObservation{}, nil
		},
	})
	want := ApplyResult{Outcome: ApplyPartialFailure, Target: target, Operations: []ApplyOperation{
		{Kind: RepositoryUpdate, Status: OperationCompleted},
		{Kind: TopicsUpdate, Status: OperationFailedUnknown},
		{Kind: LabelsRead, Status: OperationNotExecuted},
	}}
	if !errors.Is(err, cause) || !reflect.DeepEqual(result, want) || result.Completed() || !reflect.DeepEqual(calls, []string{"load", "target", "repository", "topics"}) {
		t.Fatalf("result=%+v error=%v calls=%v", result, err, calls)
	}
}

func TestLabelOnlyApplyRetainsCompletedUnknownAndUnexecutedOperations(t *testing.T) {
	for _, failName := range []string{"", "a", "b"} {
		t.Run(failName, func(t *testing.T) {
			target := repository.Repository{Host: "github.com", Owner: "owner", Name: "current"}
			labels := appLabelDeclarations(t, settings.LabelDeclaration{Name: "b"}, settings.LabelDeclaration{Name: "a"})
			cause := errors.New("connection lost after request")
			var calls []string
			result, err := Apply(func() (settings.Declaration, error) { return settings.Declaration{Labels: &labels}, nil }, func() (repository.Repository, error) { return target, nil }, Writer{
				ReadLabels: func(repository.Repository) (settings.LabelObservation, error) {
					return settings.NewLabelObservation([]settings.LabelState{{Name: "old", Color: "abcdef"}})
				},
				CreateLabel: func(repo repository.Repository, change settings.LabelChange) error {
					if repo != target {
						t.Fatalf("target=%+v", repo)
					}
					calls = append(calls, "create:"+change.Name())
					if change.Name() == failName {
						return cause
					}
					return nil
				},
				DeleteLabel: func(repo repository.Repository, change settings.LabelChange) error {
					if repo != target {
						t.Fatalf("target=%+v", repo)
					}
					calls = append(calls, "delete:"+change.Name())
					return nil
				},
			})
			want := ApplyResult{Outcome: ApplyCompleted, Target: target, Operations: []ApplyOperation{
				{Kind: LabelsRead, Status: OperationCompleted},
				{Kind: LabelCreate, LabelName: "a", Status: OperationCompleted},
				{Kind: LabelCreate, LabelName: "b", Status: OperationCompleted},
				{Kind: LabelDelete, LabelName: "old", Status: OperationCompleted},
			}}
			wantCalls := []string{"create:a", "create:b", "delete:old"}
			if failName == "a" {
				want.Outcome = ApplyFailed
				want.Operations[1].Status, want.Operations[2].Status, want.Operations[3].Status = OperationFailedUnknown, OperationNotExecuted, OperationNotExecuted
				wantCalls = wantCalls[:1]
			} else if failName == "b" {
				want.Outcome = ApplyPartialFailure
				want.Operations[2].Status, want.Operations[3].Status = OperationFailedUnknown, OperationNotExecuted
				wantCalls = wantCalls[:2]
			}
			if !reflect.DeepEqual(result, want) || !reflect.DeepEqual(calls, wantCalls) || (failName == "" && err != nil) || (failName != "" && !errors.Is(err, cause)) {
				t.Fatalf("result=%+v want=%+v calls=%v error=%v", result, want, calls, err)
			}
		})
	}
}
