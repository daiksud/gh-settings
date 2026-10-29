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

func TestRepositoryPreflightFailureRetainsKnownProgress(t *testing.T) {
	for _, earlier := range []bool{false, true} {
		for _, prewrite := range []bool{false, true} {
			t.Run(fmt.Sprintf("earlier=%t/prewrite=%t", earlier, prewrite), func(t *testing.T) {
				if result, err := applyRepositoryFailure(t, earlier, prewrite); err == nil || result.Outcome != expectedRepositoryFailureOutcome(earlier) || result.Operations[len(result.Operations)-1].Status != expectedRepositoryFailureStatus(prewrite) {
					t.Fatalf("result=%+v err=%v", result, err)
				} else if earlier && (len(result.Operations) != 3 || result.Operations[0].Status != OperationCompleted || result.Operations[1].Status != OperationCompleted || result.Operations[1].Kind != ArchiveUnarchive) {
					t.Fatal("earlier completed unarchive was lost", result)
				}
			})
		}
	}
}

func applyRepositoryFailure(t *testing.T, earlier, prewrite bool) (ApplyResult, error) {
	t.Helper()
	description, archived := "desired", false
	desired := settings.Declaration{Repository: settings.RepositoryDeclaration{Description: &description}}
	if earlier {
		desired.Repository.Archived = &archived
	}
	cause := errors.New("fixture readback failed")
	var failure error = cause
	if prewrite {
		failure = fmt.Errorf("wrapped prerequisite: %w", &repositoryReadFailure{cause})
	}
	return Apply(func() (settings.Declaration, error) { return desired, nil }, checkoutRepository, Writer{
		ReadArchive: func(repo repository.Repository) (settings.ArchiveState, error) {
			return settings.ArchiveState{ID: 1, FullName: repo.Owner + "/" + repo.Name, Archived: true}, nil
		},
		Archive:    func(repository.Repository, settings.ArchiveState, bool) error { return nil },
		Repository: func(repository.Repository, settings.RepositoryDeclaration) error { return failure },
	})
}

func expectedRepositoryFailureOutcome(earlier bool) ApplyOutcome {
	if earlier {
		return ApplyPartialFailure
	}
	return ApplyFailed
}

func expectedRepositoryFailureStatus(prewrite bool) ApplyOperationStatus {
	if prewrite {
		return OperationFailedRead
	}
	return OperationFailedUnknown
}
