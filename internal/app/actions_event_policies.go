package app

import (
	"errors"
	"fmt"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func applyActionsPolicies(result *ApplyResult, repo repository.Repository, desired *settings.ActionsPolicyDeclarations, write Writer) (bool, error) {
	index := len(result.Operations)
	result.Operations = append(result.Operations, ApplyOperation{Kind: ActionsPoliciesRead})
	observed, err := write.ReadActionsPolicies(repo)
	if err != nil {
		result.failOperation(index, OperationFailedRead)
		return false, fmt.Errorf("read Actions policies for apply (earlier writes remain applied): %w", err)
	}
	changes, err := settings.ReconcileActionsPolicies(observed, desired)
	if err != nil {
		result.failOperation(index, OperationFailedRead)
		return false, fmt.Errorf("prepare Actions policies (earlier writes remain applied): %w", err)
	}
	result.Operations[index].Status = OperationCompleted
	states, err := observed.Values()
	if err != nil {
		result.failOperation(index, OperationFailedRead)
		return false, err
	}
	knownIDs := make(map[int64]bool, len(states))
	for _, state := range states {
		knownIDs[state.ID] = true
	}
	first := len(result.Operations)
	for _, change := range changes {
		var kind ApplyOperationKind
		switch change.Operation() {
		case "create":
			kind = ActionsEventPolicyCreate
		case "update":
			kind = ActionsEventPolicyUpdate
		}
		result.Operations = append(result.Operations, ApplyOperation{Kind: kind, ActionsPolicyName: change.Name()})
	}
	for i, change := range changes {
		id, err := write.ActionsEventPolicy(repo, change)
		if change.Operation() == "create" {
			if id > 0 && !knownIDs[id] {
				result.CreatedIDs = append(result.CreatedIDs, CreatedObject{Collection: "actions_policies", Name: change.Name(), ID: id})
			}
			if err == nil && id <= 0 {
				err = fmt.Errorf("creation returned no positive ID; remote creation may have succeeded")
			}
			if knownIDs[id] {
				err = fmt.Errorf("creation returned already-known Actions policy id %d; refusing ID adoption (writes may already have applied)", id)
			}
		}
		if err == nil && change.Operation() == "update" && id != change.ID() {
			err = fmt.Errorf("update returned mismatched Actions policy id %d; expected %d (writes may already have applied)", id, change.ID())
		}
		if err != nil {
			status := OperationFailedUnknown
			var pre PreWriteReadFailure
			if errors.As(err, &pre) {
				status = OperationFailedRead
			}
			result.failOperation(first+i, status)
			return false, fmt.Errorf("%s Actions policy %q (earlier successful writes remain applied): %w", change.Operation(), change.Name(), err)
		}
		result.Operations[first+i].Status = OperationCompleted
		knownIDs[id] = true
		if change.Operation() == "create" && write.CreatedID != nil {
			if err := write.CreatedID("actions_policies", change.Name(), id); err != nil {
				result.Outcome = ApplyPartialFailure
				return false, fmt.Errorf("Actions policy %q was created with id %d, but local ID writeback failed: %w; add id: %d before retrying", change.Name(), id, err, id)
			}
		}
	}
	return len(changes) > 0, nil
}
