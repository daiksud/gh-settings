package app

import (
	"errors"
	"fmt"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

type repositoryReadFailure struct{ error }

func (*repositoryReadFailure) ReadFailedBeforeWrite() {}
func (failure *repositoryReadFailure) Unwrap() error  { return failure.error }

type repositoryCompletedWriteFailure struct{ error }

func (*repositoryCompletedWriteFailure) RepositoryWritesMayHaveApplied() {}
func (failure *repositoryCompletedWriteFailure) Unwrap() error           { return failure.error }

var repositoryFixtureCause = errors.New("fixture access step failed")

func TestRepositoryPreflightFailureRetainsKnownProgress(t *testing.T) {
	for _, earlier := range []bool{false, true} {
		for _, prewrite := range []bool{false, true} {
			for _, completedREST := range []bool{false, true} {
				t.Run(fmt.Sprintf("earlier=%t/prewrite=%t/completedREST=%t", earlier, prewrite, completedREST), func(t *testing.T) {
					result, err := applyRepositoryFailure(t, earlier, prewrite, completedREST)
					if err == nil || result.Outcome != expectedRepositoryFailureOutcome(earlier, completedREST) || result.Operations[len(result.Operations)-1].Status != expectedRepositoryFailureStatus(prewrite, completedREST) {
						t.Fatalf("result=%+v err=%v", result, err)
					} else if earlier && (len(result.Operations) != 3 || result.Operations[0].Status != OperationCompleted || result.Operations[1].Status != OperationCompleted || result.Operations[1].Kind != ArchiveUnarchive) {
						t.Fatal("earlier completed unarchive was lost", result)
					}
					var readFailure PreWriteReadFailure
					var effects interface{ RepositoryWritesMayHaveApplied() }
					if !errors.Is(err, repositoryFixtureCause) || errors.As(err, &readFailure) != prewrite || errors.As(err, &effects) != completedREST {
						t.Fatal("classification erased original cause or marker evidence", err)
					}
				})
			}
		}
	}
}

func applyRepositoryFailure(t *testing.T, earlier, prewrite, completedREST bool) (ApplyResult, error) {
	t.Helper()
	description, archived := "desired", false
	desired := settings.Declaration{Repository: settings.RepositoryDeclaration{Description: &description}}
	if earlier {
		desired.Repository.Archived = &archived
	}
	cause := repositoryFixtureCause
	var failure error = cause
	if prewrite {
		failure = fmt.Errorf("wrapped prerequisite: %w", &repositoryReadFailure{cause})
	}
	if completedREST {
		failure = fmt.Errorf("earlier REST update completed: %w", &repositoryCompletedWriteFailure{failure})
	}
	return Apply(func() (settings.Declaration, error) { return desired, nil }, checkoutRepository, Writer{
		ReadArchive: func(repo repository.Repository) (settings.ArchiveState, error) {
			return settings.ArchiveState{ID: 1, FullName: repo.Owner + "/" + repo.Name, Archived: true}, nil
		},
		Archive:    func(repository.Repository, settings.ArchiveState, bool) error { return nil },
		Repository: func(repository.Repository, settings.RepositoryDeclaration) error { return failure },
	})
}

func expectedRepositoryFailureOutcome(earlier, completedREST bool) ApplyOutcome {
	if earlier || completedREST {
		return ApplyPartialFailure
	}
	return ApplyFailed
}

func expectedRepositoryFailureStatus(prewrite, completedREST bool) ApplyOperationStatus {
	if prewrite && !completedREST {
		return OperationFailedRead
	}
	return OperationFailedUnknown
}
