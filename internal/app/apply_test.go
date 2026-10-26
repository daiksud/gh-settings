package app

import (
	"errors"
	"reflect"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func TestApplyDeclaredSettings(t *testing.T) {
	empty, disabled := "", false
	want := settings.RepositoryDeclaration{Description: &empty, HasIssues: &disabled}
	target := repository.Repository{Host: "github.com", Owner: "owner", Name: "current"}
	var calls []string
	got, err := Apply(func() (settings.Declaration, error) {
		calls = append(calls, "load")
		return settings.Declaration{Repository: want}, nil
	}, func() (repository.Repository, error) {
		calls = append(calls, "target")
		return target, nil
	}, Writer{Repository: func(repo repository.Repository, desired settings.RepositoryDeclaration) error {
		calls = append(calls, "write")
		if repo != target || !reflect.DeepEqual(desired, want) {
			t.Fatalf("target=%+v declaration=%+v, want %+v %+v", repo, desired, target, want)
		}
		return nil
	}})
	if err != nil || !reflect.DeepEqual(got, ApplyResult{Outcome: ApplyCompleted, Target: target, Operations: []ApplyOperation{{Kind: RepositoryUpdate, Status: OperationCompleted}}}) || !reflect.DeepEqual(calls, []string{"load", "target", "write"}) {
		t.Fatalf("result=%+v error=%v calls=%v, want applied after load/target/write", got, err, calls)
	}
}

func TestApplyNoManagedSettings(t *testing.T) {
	var calls []string
	got, err := Apply(func() (settings.Declaration, error) {
		calls = append(calls, "load")
		return settings.Declaration{}, nil
	}, func() (repository.Repository, error) {
		calls = append(calls, "target")
		return repository.Repository{}, nil
	}, Writer{})
	if err != nil || !reflect.DeepEqual(got, ApplyResult{}) || !reflect.DeepEqual(calls, []string{"load"}) {
		t.Fatalf("result=%+v error=%v calls=%v", got, err, calls)
	}
}

func TestApplyValidatesBeforeWrite(t *testing.T) {
	loadError := errors.New("validate .github/settings.yml: declaration rejected")
	var calls []string
	got, err := Apply(func() (settings.Declaration, error) {
		calls = append(calls, "load")
		disabled := false
		return settings.Declaration{Repository: settings.RepositoryDeclaration{HasIssues: &disabled}}, loadError
	}, func() (repository.Repository, error) {
		calls = append(calls, "target")
		return repository.Repository{}, nil
	}, Writer{})
	if err != loadError || !reflect.DeepEqual(got, ApplyResult{Outcome: ApplyFailed}) || !reflect.DeepEqual(calls, []string{"load"}) {
		t.Fatalf("result=%+v error=%v calls=%v", got, err, calls)
	}
}

func TestApplyTargetFailureBeforeWrite(t *testing.T) {
	cause := errors.New("resolve current repository: original origin failure")
	var calls []string
	got, err := Apply(func() (settings.Declaration, error) {
		calls = append(calls, "load")
		disabled := false
		return settings.Declaration{Repository: settings.RepositoryDeclaration{HasIssues: &disabled}}, nil
	}, func() (repository.Repository, error) {
		calls = append(calls, "target")
		return repository.Repository{Host: "github.com", Owner: "partial", Name: "target"}, cause
	}, Writer{})
	if err != cause || !reflect.DeepEqual(got, ApplyResult{Outcome: ApplyFailed}) || !reflect.DeepEqual(calls, []string{"load", "target"}) {
		t.Fatalf("result=%+v error=%v calls=%v", got, err, calls)
	}
}

func TestApplyWriteFailure(t *testing.T) {
	cause := errors.New("write unavailable")
	target := repository.Repository{Host: "github.com", Owner: "owner", Name: "current"}
	var calls []string
	got, err := Apply(func() (settings.Declaration, error) {
		calls = append(calls, "load")
		disabled := false
		return settings.Declaration{Repository: settings.RepositoryDeclaration{HasIssues: &disabled}}, nil
	}, func() (repository.Repository, error) {
		calls = append(calls, "target")
		return target, nil
	}, Writer{Repository: func(repo repository.Repository, desired settings.RepositoryDeclaration) error {
		calls = append(calls, "write")
		if repo != target || desired.HasIssues == nil || *desired.HasIssues {
			t.Fatalf("target=%+v declaration=%+v", repo, desired)
		}
		return cause
	}})
	if err != cause || !reflect.DeepEqual(got, ApplyResult{Outcome: ApplyFailed, Target: target, Operations: []ApplyOperation{{Kind: RepositoryUpdate, Status: OperationFailedUnknown}}}) || !reflect.DeepEqual(calls, []string{"load", "target", "write"}) {
		t.Fatalf("result=%+v error=%v calls=%v", got, err, calls)
	}
}

func TestApplyManagedEmptyCollectionsResolveTarget(t *testing.T) {
	topics := []string{}
	for _, test := range []struct {
		name    string
		desired settings.Declaration
	}{
		{"topics", settings.Declaration{Topics: settings.TopicsDeclaration{Names: &topics}}},
	} {
		t.Run(test.name, func(t *testing.T) {
			cause := errors.New("origin unavailable")
			var calls []string
			got, err := Apply(func() (settings.Declaration, error) {
				calls = append(calls, "load")
				return test.desired, nil
			},
				func() (repository.Repository, error) {
					calls = append(calls, "target")
					return repository.Repository{}, cause
				}, Writer{})
			if err != cause || !reflect.DeepEqual(got, ApplyResult{Outcome: ApplyFailed}) || !reflect.DeepEqual(calls, []string{"load", "target"}) {
				t.Fatalf("result=%+v error=%v calls=%v", got, err, calls)
			}
		})
	}
}
