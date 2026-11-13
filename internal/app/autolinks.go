package app

import (
	"fmt"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func applyAutolinks(result *ApplyResult, repo repository.Repository, desired []settings.Autolink, write Writer, prune bool, prepared settings.AutolinkObservation) (bool, error) {
	readIndex := len(result.Operations)
	result.Operations = append(result.Operations, ApplyOperation{Kind: AutolinksRead})
	observed, err := write.ReadAutolinks(repo)
	if err == nil && prune {
		err = settings.VerifyAutolinkInventory(prepared, observed)
	}
	var changes []settings.AutolinkChange
	if err == nil {
		changes, err = settings.CompareAutolinks(observed, desired, prune)
	}
	if err == nil && observed.Target.Archived && len(changes) > 0 {
		err = fmt.Errorf("archived repository cannot change autolinks")
	}
	if err != nil {
		result.failOperation(readIndex, OperationFailedRead)
		return false, fmt.Errorf("prepare autolinks%s: %w", environmentProgress(*result), err)
	}
	result.Operations[readIndex].Status = OperationCompleted
	expected := observed.Clone()
	first := len(result.Operations)
	for _, change := range changes {
		kind := AutolinkCreate
		if change.Before != nil {
			kind = AutolinkDelete
		}
		result.Operations = append(result.Operations, ApplyOperation{Kind: kind})
	}
	wrote := false
	fail := func(index int, attempted bool, err error) (bool, error) {
		status := OperationFailedRead
		uncertain := ""
		if attempted {
			status = OperationFailedUnknown
			uncertain = " (unconfirmed effects may remain)"
		}
		result.failOperation(index, status)
		return wrote, fmt.Errorf("autolinks%s%s: %w", environmentProgress(*result), uncertain, err)
	}
	// Compare orders all declared creates before the globally fixed deletions.
	for i, change := range changes {
		index := first + i
		if change.After != nil {
			applied, err := write.CreateAutolink(repo, expected, *change.After)
			_, known := expected.Find(applied.RecoveryID)
			if applied.Attempted && applied.RecoveryID > 0 && !known {
				result.CreatedIDs = append(result.CreatedIDs, CreatedObject{Collection: "autolinks", Name: change.After.ReferencePrefix, ID: applied.RecoveryID})
			}
			if err == nil && (applied.Verified == nil || !applied.Attempted || known || applied.RecoveryID <= 0 || applied.Verified.Value.ID == nil || *applied.Verified.Value.ID != applied.RecoveryID || !settings.AutolinkMatches(applied.Verified.Value, *change.After)) {
				err = fmt.Errorf("autolink creation lacks independently verified new identity")
			}
			var next settings.AutolinkObservation
			if err == nil {
				next, err = expected.WithCreated(*applied.Verified)
			}
			if err != nil {
				return fail(index, applied.Attempted, err)
			}
			expected = next
			wrote = true
			result.Operations[index].Status = OperationCompleted
			if write.CreatedID != nil {
				if err = write.CreatedID("autolinks", change.After.ReferencePrefix, applied.RecoveryID); err != nil {
					result.Outcome = ApplyPartialFailure
					return wrote, fmt.Errorf("autolink created with id %d but local ID recording failed: %w", applied.RecoveryID, err)
				}
			}
		} else {
			attempted, err := write.DeleteAutolink(repo, expected, *change.Before.ID)
			if err != nil {
				return fail(index, attempted, err)
			}
			if !attempted {
				return fail(index, false, fmt.Errorf("autolink deletion lacks write evidence"))
			}
			expected = expected.Without(*change.Before.ID)
			wrote = true
			result.Operations[index].Status = OperationCompleted
		}
	}
	return wrote, nil
}
