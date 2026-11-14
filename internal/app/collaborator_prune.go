package app

import (
	"fmt"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func applyCollaboratorPrune(result *ApplyResult, repo repository.Repository, prepared settings.CollaboratorPruneObservation, write Writer) (bool, error) {
	i := len(result.Operations)
	result.Operations = append(result.Operations, ApplyOperation{Kind: CollaboratorsRead})
	if write.RefreshCollaboratorPrune == nil {
		result.failOperation(i, OperationFailedRead)
		return false, fmt.Errorf("collaborator prune recheck unavailable")
	}
	current, err := write.RefreshCollaboratorPrune(repo, prepared, nil)
	if err != nil {
		result.failOperation(i, OperationFailedRead)
		return false, err
	}
	result.Operations[i].Status = OperationCompleted
	wrote, err := applySelectedCollaborators(result, repo, current.Selections(), write, func(observed settings.CollaboratorWriteResult) error {
		var err error
		current, err = write.RefreshCollaboratorPrune(repo, current, &observed)
		return err
	})
	if err != nil {
		return wrote, err
	}
	candidates := current.Deletions()
	first := len(result.Operations)
	for _, d := range candidates {
		kind := CollaboratorRemove
		if d.InvitationID > 0 {
			kind = CollaboratorCancelInvitation
		}
		result.Operations = append(result.Operations, ApplyOperation{Kind: kind, CollaboratorUsername: d.Person.Username})
	}
	for offset, d := range candidates {
		i := first + offset
		if write.DeleteCollaborator == nil {
			result.failOperation(i, OperationFailedRead)
			return wrote, fmt.Errorf("collaborator deletion writer unavailable")
		}
		next, observed, err := write.DeleteCollaborator(repo, current, d)
		result.CollaboratorDeletions = append(result.CollaboratorDeletions, observed)
		if err != nil {
			status := OperationFailedUnknown
			// The deletion result explicitly records whether any request was sent,
			// including client construction failures without a transport marker.
			if !observed.Attempted && !observed.Acknowledged && !observed.Verified {
				status = OperationFailedRead
			}
			result.failOperation(i, status)
			return wrote, err
		}
		if observed.Deletion != d || !observed.Attempted || !observed.Acknowledged || !observed.Verified {
			status := OperationFailedUnknown
			if !observed.Attempted && !observed.Acknowledged && !observed.Verified {
				status = OperationFailedRead
			}
			result.failOperation(i, status)
			return wrote, fmt.Errorf("collaborator deletion was not independently verified")
		}
		current = next
		wrote = true
		result.Operations[i].Status = OperationCompleted
	}
	return wrote, nil
}
