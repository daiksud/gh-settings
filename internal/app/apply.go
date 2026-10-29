package app

import (
	"errors"
	"fmt"
	"github.com/daiksud/gh-settings/internal/settings"
)

// Apply keeps direct repository/topics writes; labels read their current collection before CRUD.
func Apply(load DeclarationLoader, resolve TargetResolver, write Writer) (result ApplyResult, err error) {
	unarchived := false
	defer func() {
		if err != nil && unarchived {
			err = fmt.Errorf("repository was unarchived before subsequent settings failed; earlier changes remain applied: %w", err)
		}
	}()
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
	result = ApplyResult{Outcome: ApplyFailed, Target: repo}
	archiveManaged := desired.Repository.Archived != nil
	other := desired.Repository
	other.Archived = nil
	repositoryManaged, topicsManaged := !other.Empty(), desired.Topics.Names != nil
	var archiveState settings.ArchiveState
	if archiveManaged {
		result.Operations = append(result.Operations, ApplyOperation{Kind: ArchiveRead})
		archiveState, err = write.ReadArchive(repo)
		if err == nil {
			err = archiveState.Validate(repo)
		}
		if err != nil {
			result.failOperation(0, OperationFailedRead)
			return result, fmt.Errorf("read archive state before apply: %w", err)
		}
		result.Operations[0].Status = OperationCompleted
		if archiveState.Archived && *desired.Repository.Archived && archiveConflictsWithOtherManaged(desired) {
			result.failOperation(0, OperationFailedRead)
			return result, fmt.Errorf("archived repository cannot update other managed resources without explicit archived: false")
		}
		if archiveState.Archived && !*desired.Repository.Archived {
			i := len(result.Operations)
			result.Operations = append(result.Operations, ApplyOperation{Kind: ArchiveUnarchive})
			if err := write.Archive(repo, archiveState, false); err != nil {
				result.failOperation(i, OperationFailedUnknown)
				return result, fmt.Errorf("unarchive before other settings: %w", err)
			}
			result.Operations[i].Status = OperationCompleted
			unarchived = true
		}
	}
	firstOrdinary := len(result.Operations)
	if repositoryManaged {
		result.Operations = append(result.Operations, ApplyOperation{Kind: RepositoryUpdate})
	}
	if topicsManaged {
		result.Operations = append(result.Operations, ApplyOperation{Kind: TopicsUpdate})
	}
	if desired.Labels != nil && desired.Labels.Len() > 0 {
		result.Operations = append(result.Operations, ApplyOperation{Kind: LabelsRead})
	}
	index := firstOrdinary
	if repositoryManaged {
		if err := write.Repository(repo, other); err != nil {
			status := OperationFailedUnknown
			var readFailure PreWriteReadFailure
			var writeEffects RepositoryWriteEffectsFailure
			hasWriteEffects := errors.As(err, &writeEffects)
			if !hasWriteEffects && errors.As(err, &readFailure) {
				status = OperationFailedRead
			}
			result.failOperation(index, status)
			if hasWriteEffects {
				result.Outcome = ApplyPartialFailure
			}
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
	hasWrites := repositoryManaged || topicsManaged || (archiveManaged && archiveState.Archived && !*desired.Repository.Archived)
	if desired.Labels != nil && desired.Labels.Len() > 0 {
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
				var id int64
				id, err = write.CreateLabel(repo, change)
				if id > 0 {
					result.CreatedIDs = append(result.CreatedIDs, CreatedObject{Collection: "labels", Name: change.Name(), ID: id})
				}
				if err == nil && id <= 0 {
					err = fmt.Errorf("creation returned no positive ID; remote creation may have succeeded")
				}
				if err == nil {
					result.Operations[index].Status = OperationCompleted
					if write.CreatedID != nil {
						if saveErr := write.CreatedID("labels", change.Name(), id); saveErr != nil {
							result.Outcome = ApplyPartialFailure
							return result, fmt.Errorf("label %q was created with id %d, but local ID writeback failed: %w; add id: %d before retrying", change.Name(), id, saveErr, id)
						}
					}
				}
			case LabelUpdate:
				err = write.UpdateLabel(repo, change)
			case LabelDelete:
				err = write.DeleteLabel(repo, change)
			}
			if err != nil {
				status := OperationFailedUnknown
				var readFailure PreWriteReadFailure
				if errors.As(err, &readFailure) {
					status = OperationFailedRead
				}
				result.failOperation(index, status)
				return result, fmt.Errorf("%s label %q (earlier successful writes remain applied): %w", change.Operation(), change.Name(), err)
			}
			result.Operations[index].Status = OperationCompleted
		}
	}
	if desired.Rulesets != nil && desired.Rulesets.Len() > 0 {
		wrote, err := applyRulesets(&result, repo, desired.Rulesets, write)
		if err != nil {
			return result, err
		}
		hasWrites = hasWrites || wrote
	}
	if desired.Environments != nil && desired.Environments.Len() > 0 {
		wrote, err := applyEnvironments(&result, repo, desired.Environments, write)
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
	if !desired.Actions.RetentionEmpty() {
		wrote, err := applyRetention(&result, repo, desired.Actions, write)
		if err != nil {
			return result, err
		}
		hasWrites = hasWrites || wrote
	}
	if desired.Actions.ForkApprovalPolicy != nil {
		wrote, err := applyForkApproval(&result, repo, desired.Actions, write)
		if err != nil {
			return result, err
		}
		hasWrites = hasWrites || wrote
	}
	if !desired.Actions.PrivateFork.Empty() {
		wrote, err := applyPrivateFork(&result, repo, desired.Actions, write)
		if err != nil {
			return result, err
		}
		hasWrites = hasWrites || wrote
	}
	if !desired.Security.SecretScanningEmpty() {
		wrote, err := applySecretScanning(&result, repo, desired.Security, write)
		if err != nil {
			return result, err
		}
		hasWrites = hasWrites || wrote
	}
	if desired.Security.PrivateVulnerabilityReporting != nil {
		wrote, err := applyPrivateReporting(&result, repo, desired.Security, write)
		if err != nil {
			return result, err
		}
		hasWrites = hasWrites || wrote
	}
	if desired.Security.ImmutableReleases != nil {
		wrote, err := applyImmutableReleases(&result, repo, desired.Security, write)
		if err != nil {
			return result, err
		}
		hasWrites = hasWrites || wrote
	}
	if !desired.Pages.Empty() {
		wrote, err := applyPages(&result, repo, desired.Pages, write)
		if err != nil {
			return result, err
		}
		hasWrites = hasWrites || wrote
	}
	if archiveManaged && *desired.Repository.Archived && !archiveState.Archived {
		i := len(result.Operations)
		result.Operations = append(result.Operations, ApplyOperation{Kind: ArchiveRead})
		fresh, err := write.ReadArchive(repo)
		if err == nil {
			err = fresh.Validate(repo)
		}
		if err == nil && (fresh.ID != archiveState.ID || fresh.FullName != archiveState.FullName || fresh.Archived != archiveState.Archived) {
			err = fmt.Errorf("repository identity or archive state changed during apply")
		}
		if err != nil {
			result.failOperation(i, OperationFailedRead)
			return result, fmt.Errorf("read archive state before final transition: %w", err)
		}
		result.Operations[i].Status = OperationCompleted
		i = len(result.Operations)
		result.Operations = append(result.Operations, ApplyOperation{Kind: ArchiveUpdate})
		if err := write.Archive(repo, fresh, true); err != nil {
			result.failOperation(i, OperationFailedUnknown)
			return result, fmt.Errorf("archive after other settings: %w", err)
		}
		result.Operations[i].Status = OperationCompleted
		hasWrites = true
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

// Archive does not make other declared resources writable on an already archived target.
func archiveConflictsWithOtherManaged(desired settings.Declaration) bool {
	other := desired.Repository
	other.Archived = nil
	return !other.Empty() || desired.Topics.Names != nil || (desired.Labels != nil && desired.Labels.Len() > 0) || (desired.Rulesets != nil && desired.Rulesets.Len() > 0) || (desired.Environments != nil && desired.Environments.Len() > 0) || !desired.Actions.Empty() || !desired.Security.Empty() || !desired.Pages.Empty()
}
