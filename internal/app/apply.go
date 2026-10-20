package app

import (
	"errors"
	"fmt"
	"github.com/daiksud/gh-settings/internal/settings"
)

// Apply keeps direct repository/topics writes; labels read their current collection before CRUD.
func Apply(load DeclarationLoader, resolve TargetResolver, write Writer) (ApplyResult, error) {
	desired, err := Validate(load)
	if err != nil {
		return ApplyResult{Outcome: ApplyFailed}, err
	}
	if desired.Empty() {
		return ApplyResult{}, nil
	}
	repo, err := resolve()
	if err != nil {
		return ApplyResult{Outcome: ApplyFailed}, err
	}
	result := ApplyResult{Outcome: ApplyFailed, Target: repo}
	repositoryManaged, topicsManaged := !desired.Repository.Empty(), desired.Topics.Names != nil
	if repositoryManaged {
		result.Operations = append(result.Operations, ApplyOperation{Kind: RepositoryUpdate})
	}
	if topicsManaged {
		result.Operations = append(result.Operations, ApplyOperation{Kind: TopicsUpdate})
	}
	if desired.Labels != nil {
		result.Operations = append(result.Operations, ApplyOperation{Kind: LabelsRead})
	}
	index := 0
	if repositoryManaged {
		if err := write.Repository(repo, desired.Repository); err != nil {
			result.failOperation(index, OperationFailedUnknown)
			return result, err
		}
		result.Operations[index].Status = OperationCompleted
		index++
	}
	if topicsManaged {
		if err := write.Topics(repo, desired.Topics); err != nil {
			result.failOperation(index, OperationFailedUnknown)
			return result, err
		}
		result.Operations[index].Status = OperationCompleted
		index++
	}
	hasWrites := repositoryManaged || topicsManaged
	if desired.Labels != nil {
		current, err := write.ReadLabels(repo)
		if err != nil {
			result.failOperation(index, OperationFailedRead)
			return result, fmt.Errorf("read labels for apply (earlier resource writes remain applied): %w", err)
		}
		comparison, err := settings.CompareLabels(current, desired.Labels)
		if err != nil {
			result.failOperation(index, OperationFailedRead)
			return result, fmt.Errorf("compare labels for apply (earlier resource writes remain applied): %w", err)
		}
		result.Operations[index].Status = OperationCompleted
		changes := comparison.Labels.Values()
		hasWrites = hasWrites || len(changes) > 0
		firstLabel := len(result.Operations)
		for _, change := range changes {
			result.Operations = append(result.Operations, ApplyOperation{Kind: labelApplyKind(change), LabelName: change.Name()})
		}
		for i, change := range changes {
			index := firstLabel + i
			var err error
			switch result.Operations[index].Kind {
			case LabelCreate:
				err = write.CreateLabel(repo, change)
			case LabelUpdate:
				err = write.UpdateLabel(repo, change)
			case LabelDelete:
				err = write.DeleteLabel(repo, change)
			}
			if err != nil {
				result.failOperation(index, OperationFailedUnknown)
				return result, fmt.Errorf("%s label %q (earlier successful writes remain applied): %w", change.Operation(), change.Name(), err)
			}
			result.Operations[index].Status = OperationCompleted
		}
	}
	if desired.Rulesets != nil {
		wrote, err := applyRulesets(&result, repo, desired.Rulesets, write)
		if err != nil {
			return result, err
		}
		hasWrites = hasWrites || wrote
	}
	if !desired.Actions.ExecutionEmpty() {
		index := len(result.Operations)
		result.Operations = append(result.Operations, ApplyOperation{Kind: ActionsPolicyUpdate})
		if err := write.Actions(repo, desired.Actions); err != nil {
			result.failOperation(index, OperationFailedUnknown)
			return result, fmt.Errorf("apply Actions policy (earlier resource writes remain applied): %w", err)
		}
		result.Operations[index].Status = OperationCompleted
		hasWrites = true
	}
	if !desired.Actions.WorkflowPermissionsEmpty() {
		index := len(result.Operations)
		result.Operations = append(result.Operations, ApplyOperation{Kind: WorkflowPermissionsUpdate})
		if err := write.WorkflowPermissions(repo, desired.Actions); err != nil {
			status := OperationFailedUnknown
			var readFailure PreWriteReadFailure
			if errors.As(err, &readFailure) {
				status = OperationFailedRead
			}
			result.failOperation(index, status)
			return result, fmt.Errorf("apply workflow permissions (earlier resource writes remain applied): %w", err)
		}
		result.Operations[index].Status = OperationCompleted
		hasWrites = true
	}
	if !desired.Security.Empty() {
		wrote, err := applySecretScanning(&result, repo, desired.Security, write)
		if err != nil {
			return result, err
		}
		hasWrites = hasWrites || wrote
	}
	result.Outcome = ApplyCompleted
	if !hasWrites {
		result.Outcome = ApplyMatchingLabels
	}
	return result, nil
}

func labelApplyKind(change settings.LabelChange) ApplyOperationKind {
	switch change.Operation() {
	case "create":
		return LabelCreate
	case "update":
		return LabelUpdate
	case "delete":
		return LabelDelete
	default:
		panic("unknown typed label operation")
	}
}
