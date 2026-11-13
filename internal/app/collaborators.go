package app

import (
	"errors"
	"fmt"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func prepareCollaborators(repo repository.Repository, desired settings.CollaboratorDeclarations, read func(repository.Repository) (settings.CollaboratorObservation, error), resolve func(repository.Repository, settings.CollaboratorDeclaration) (settings.CollaboratorIdentity, error)) ([]settings.CollaboratorSelection, error) {
	if read == nil || resolve == nil {
		return nil, fmt.Errorf("collaborator reader unavailable")
	}
	observed, err := read(repo)
	if err != nil {
		return nil, err
	}
	resolved := make([]settings.CollaboratorIdentity, 0, desired.Len())
	for _, person := range desired.Values() {
		identity, err := resolve(repo, person)
		if err != nil {
			return nil, err
		}
		resolved = append(resolved, identity)
	}
	return settings.ReconcileCollaborators(observed, desired, resolved)
}

// CollaboratorResult separates verified local binding from attempted invitation
// progress. A reported invitation ID is recovery evidence, never accepted access.
type CollaboratorResult struct {
	Username string
	settings.CollaboratorWriteResult
	BindingVerified, BindingRecorded bool
}

func applyCollaborators(result *ApplyResult, repo repository.Repository, desired settings.CollaboratorDeclarations, write Writer) (bool, error) {
	readIndex := len(result.Operations)
	result.Operations = append(result.Operations, ApplyOperation{Kind: CollaboratorsRead})
	selected, err := prepareCollaborators(repo, desired, write.ReadCollaborators, write.ResolveCollaborator)
	if err != nil {
		result.failOperation(readIndex, OperationFailedRead)
		return false, err
	}
	result.Operations[readIndex].Status = OperationCompleted
	return applySelectedCollaborators(result, repo, selected, write, nil)
}

func applySelectedCollaborators(result *ApplyResult, repo repository.Repository, selected []settings.CollaboratorSelection, write Writer, after func(settings.CollaboratorWriteResult) error) (bool, error) {
	wrote := false
	for _, person := range selected {
		index := len(result.Operations)
		result.Operations = append(result.Operations, ApplyOperation{Kind: CollaboratorReconcile, CollaboratorUsername: person.DeclaredUsername})
		if write.Collaborator == nil {
			result.failOperation(index, OperationFailedRead)
			return wrote, fmt.Errorf("collaborator writer unavailable")
		}
		observed, err := write.Collaborator(repo, person)
		entry := CollaboratorResult{Username: person.DeclaredUsername, CollaboratorWriteResult: observed}
		result.Collaborators = append(result.Collaborators, entry)
		entryIndex := len(result.Collaborators) - 1
		wrote = wrote || observed.Invited
		if err != nil {
			status := OperationFailedUnknown
			var pre PreWriteReadFailure
			if !observed.Attempted && errors.As(err, &pre) {
				status = OperationFailedRead
			}
			result.failOperation(index, status)
			// A valid 201 establishes a sent invitation even if readback failed.
			if observed.Invited {
				result.Outcome = ApplyPartialFailure
			}
			return wrote, err
		}
		if observed.Person.ID != person.Person.ID || (observed.Status != "active" && observed.Status != "pending") {
			result.failOperation(index, OperationFailedUnknown)
			return wrote, fmt.Errorf("collaborator writer did not return a verified identity and state")
		}
		result.Operations[index].Status = OperationUnchanged
		if observed.Invited {
			result.Operations[index].Status = OperationCompleted
		}
		if person.Bind {
			result.Collaborators[entryIndex].BindingVerified = true
			if write.BindCollaboratorID != nil {
				bindingIndex := len(result.Operations)
				result.Operations = append(result.Operations, ApplyOperation{Kind: CollaboratorIdentityBinding, CollaboratorUsername: person.DeclaredUsername})
				if err := write.BindCollaboratorID(person.DeclaredUsername, observed.Person.ID); err != nil {
					result.failOperation(bindingIndex, OperationFailedLocal)
					return wrote, fmt.Errorf("record verified collaborator user ID: %w", err)
				}
				result.Collaborators[entryIndex].BindingRecorded = true
				result.Operations[bindingIndex].Status = OperationCompleted
			}
		}
		if after != nil {
			i := len(result.Operations)
			result.Operations = append(result.Operations, ApplyOperation{Kind: CollaboratorsRead})
			if err := after(observed); err != nil {
				result.failOperation(i, OperationFailedRead)
				return wrote, err
			}
			result.Operations[i].Status = OperationCompleted
		}
	}
	return wrote, nil
}
