package app

import (
	"errors"
	"fmt"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

// Apply sequences managed resources using their fresh reconciliation results.
func Apply(load DeclarationLoader, resolve TargetResolver, write Writer, options ...settings.PruneScopes) (result ApplyResult, err error) {
	prune, optionErr := operationPrune(options)
	if optionErr != nil {
		return ApplyResult{Outcome: ApplyFailed}, optionErr
	}
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
	if desired.Empty() && !prune.Manages(desired) {
		return ApplyResult{}, nil
	}
	repo, err := resolve()
	if err != nil {
		return ApplyResult{Outcome: ApplyFailed}, err
	}
	result = ApplyResult{Outcome: ApplyFailed, Target: repo}
	prepared, err := preparePrune(repo, desired, prune, write)
	if err != nil {
		return result, fmt.Errorf("prune preflight failed before any writes: %w", err)
	}
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
		if archiveState.Archived && *desired.Repository.Archived && (archiveConflictsWithOtherManaged(desired) || prune.Manages(desired)) {
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
	if desired.Labels != nil && (desired.Labels.Len() > 0 || prune.Has("labels")) {
		result.Operations = append(result.Operations, ApplyOperation{Kind: LabelsRead})
	}
	index := firstOrdinary
	repositoryWrote := false
	if repositoryManaged {
		repositoryWrote, err = write.Repository(repo, other)
		if err != nil {
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
		result.Operations[index].Status = OperationUnchanged
		if repositoryWrote {
			result.Operations[index].Status = OperationCompleted
		}
		index++
	}
	topicsWrote := false
	if topicsManaged {
		topicsWrote, err = write.Topics(repo, desired.Topics)
		if err != nil {
			status := OperationFailedUnknown
			var readFailure PreWriteReadFailure
			if errors.As(err, &readFailure) {
				status = OperationFailedRead
			}
			result.failOperation(index, status)
			if result.Outcome == ApplyPartialFailure && !unarchived {
				return result, fmt.Errorf("earlier repository settings update remains applied: %w", err)
			}
			return result, err
		}
		result.Operations[index].Status = OperationUnchanged
		if topicsWrote {
			result.Operations[index].Status = OperationCompleted
		}
		index++
	}
	hasWrites := repositoryWrote || topicsWrote || (archiveManaged && archiveState.Archived && !*desired.Repository.Archived)
	if desired.Labels != nil && (desired.Labels.Len() > 0 || prune.Has("labels")) {
		progress := ""
		if hasWrites {
			progress = " (earlier resource writes remain applied)"
		}
		current, err := write.ReadLabels(repo)
		if err != nil {
			result.failOperation(index, OperationFailedRead)
			return result, fmt.Errorf("read labels for apply%s: %w", progress, err)
		}
		comparison, err := settings.CompareLabelsWithPrune(current, desired.Labels, prune.Has("labels"))
		if err != nil {
			result.failOperation(index, OperationFailedRead)
			return result, fmt.Errorf("compare labels for apply%s: %w", progress, err)
		}
		if prune.Has("labels") && !sameLabelDeletions(prepared.labels, comparison.Labels.Values()) {
			result.failOperation(index, OperationFailedRead)
			return result, fmt.Errorf("label prune candidates changed after preflight; refusing collection writes")
		}
		result.Operations[index].Status = OperationCompleted
		acquiredLabels, readErr := current.Values()
		if readErr != nil {
			result.failOperation(index, OperationFailedRead)
			return result, readErr
		}
		knownLabelIDs := map[int64]bool{}
		for _, label := range acquiredLabels {
			knownLabelIDs[label.ID] = true
		}
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
				if err == nil && knownLabelIDs[id] {
					err = fmt.Errorf("creation returned already-known label id %d; refusing ID adoption (writes may already have applied)", id)
				}
				if err == nil {
					knownLabelIDs[id] = true
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
				progress := ""
				if result.Outcome == ApplyPartialFailure {
					progress = " (earlier successful writes remain applied)"
				}
				return result, fmt.Errorf("%s label %q%s: %w", change.Operation(), change.Name(), progress, err)
			}
			result.Operations[index].Status = OperationCompleted
		}
	}
	if desired.ActionsPolicies != nil && (desired.ActionsPolicies.Len() > 0 || prune.Has("actions-policies")) {
		wrote, err := applyActionsPolicies(&result, repo, desired.ActionsPolicies, write, prune.Has("actions-policies"), prepared.actionsPolicies)
		if err != nil {
			return result, err
		}
		hasWrites = hasWrites || wrote
	}
	if desired.Rulesets != nil && (desired.Rulesets.Len() > 0 || prune.Has("rulesets")) {
		wrote, err := applyRulesets(&result, repo, desired.Rulesets, write, prune.Has("rulesets"), prepared.rulesets)
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
	if prune.Has("environment-variables") && len(desired.EnvironmentVariables) > 0 {
		for _, selected := range sortedEnvironmentVariables(desired.EnvironmentVariables) {
			environment := selected.Environment
			wrote, err := applyPrunedVariables(&result, repo, selected.Variables, environment, prepared.environmentVariables[settings.EnvironmentIdentity(environment)], EnvironmentVariablesRead, EnvironmentVariableUpsert, EnvironmentVariableDelete,
				func(repo repository.Repository) (settings.VariableInventory, error) {
					return write.ReadEnvironmentVariables(repo, environment)
				},
				func(repo repository.Repository, value settings.RepositoryVariable) (bool, error) {
					return write.EnvironmentVariable(repo, environment, value)
				},
				func(repo repository.Repository, id int64, change settings.VariableChange) error {
					return write.DeleteEnvironmentVariable(repo, environment, id, change)
				})
			if err != nil {
				return result, err
			}
			hasWrites = hasWrites || wrote
		}
	} else if !settings.EnvironmentVariablesEmpty(desired.EnvironmentVariables) {
		wrote, err := applyEnvironmentVariables(&result, repo, desired.EnvironmentVariables, write.EnvironmentVariable)
		if err != nil {
			return result, err
		}
		hasWrites = hasWrites || wrote
	}
	if desired.Variables != nil && (desired.Variables.Len() > 0 || prune.Has("actions-variables")) {
		var wrote bool
		var err error
		if prune.Has("actions-variables") {
			wrote, err = applyPrunedVariables(&result, repo, *desired.Variables, "", environmentVariablePrune{changes: prepared.variables}, VariablesRead, VariableUpsert, VariableDelete, write.ReadVariables, write.Variable, func(repo repository.Repository, _ int64, change settings.VariableChange) error {
				return write.DeleteVariable(repo, change)
			})
		} else {
			wrote, err = applyVariables(&result, repo, *desired.Variables, VariableUpsert, write.Variable)
		}
		if err != nil {
			return result, err
		}
		hasWrites = hasWrites || wrote
	}
	if desired.AgentsVariables != nil && (desired.AgentsVariables.Len() > 0 || prune.Has("agents-variables")) {
		var wrote bool
		var err error
		if prune.Has("agents-variables") {
			wrote, err = applyPrunedVariables(&result, repo, *desired.AgentsVariables, "", environmentVariablePrune{changes: prepared.agentsVariables}, AgentVariablesRead, AgentVariableUpsert, AgentVariableDelete, write.ReadAgentVariables, write.AgentVariable, func(repo repository.Repository, _ int64, change settings.VariableChange) error {
				return write.DeleteAgentVariable(repo, change)
			})
		} else {
			wrote, err = applyVariables(&result, repo, *desired.AgentsVariables, AgentVariableUpsert, write.AgentVariable)
		}
		if err != nil {
			return result, fmt.Errorf("apply Agents variables: %w", err)
		}
		hasWrites = hasWrites || wrote
	}
	if !desired.Actions.ExecutionEmpty() {
		index := len(result.Operations)
		result.Operations = append(result.Operations, ApplyOperation{Kind: ActionsPolicyUpdate})
		wrote, err := write.Actions(repo, desired.Actions)
		if err != nil {
			status := OperationFailedUnknown
			var readFailure PreWriteReadFailure
			if errors.As(err, &readFailure) {
				status = OperationFailedRead
			}
			result.failOperation(index, status)
			progress := ""
			if hasWrites {
				progress = " (earlier resource writes remain applied)"
			}
			return result, fmt.Errorf("apply Actions policy%s: %w", progress, err)
		}
		result.Operations[index].Status = OperationUnchanged
		if wrote {
			result.Operations[index].Status = OperationCompleted
		}
		hasWrites = hasWrites || wrote
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
	if desired.Actions.CacheRetentionDays != nil {
		wrote, err := applyCacheRetention(&result, repo, desired.Actions, write)
		if err != nil {
			return result, err
		}
		hasWrites = hasWrites || wrote
	}
	if desired.Actions.CacheSizeGB != nil {
		wrote, err := applyCacheStorage(&result, repo, desired.Actions, write)
		if err != nil {
			return result, err
		}
		hasWrites = hasWrites || wrote
	}
	if desired.Actions.AccessLevel != nil {
		wrote, err := applyActionsAccess(&result, repo, desired.Actions, write)
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
	if desired.CopilotInstructions != nil {
		wrote, err := applyCopilotInstructions(&result, repo, *desired.CopilotInstructions, write)
		if err != nil {
			return result, err
		}
		hasWrites = hasWrites || wrote
	}
	if desired.CopilotMCP != nil {
		wrote, err := applyCopilotMCP(&result, repo, *desired.CopilotMCP, write)
		if err != nil {
			return result, err
		}
		hasWrites = hasWrites || wrote
	}
	if desired.PullRequestExemptions != nil {
		wrote, err := applyPullRequestExemptions(&result, repo, *desired.PullRequestExemptions, write)
		if err != nil {
			return result, err
		}
		hasWrites = hasWrites || wrote
	}
	if !desired.PullRequestLimits.Empty() {
		wrote, err := applyPullRequestLimits(&result, repo, desired.PullRequestLimits, write)
		if err != nil {
			return result, err
		}
		hasWrites = hasWrites || wrote
	}
	if desired.CopilotReviewEffort != nil {
		wrote, err := applyCopilotReviewEffort(&result, repo, *desired.CopilotReviewEffort, write)
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
	if !desired.Security.DependabotEmpty() {
		wrote, err := applyDependabot(&result, repo, desired.Security, write)
		if err != nil {
			return result, err
		}
		hasWrites = hasWrites || wrote
	}
	if !desired.Security.CodeQLConfiguration.Empty() {
		wrote, err := applyCodeQLConfiguration(&result, repo, desired.Security.CodeQLConfiguration, write)
		if err != nil {
			return result, err
		}
		hasWrites = hasWrites || wrote
	}
	if desired.Security.CodeQLDefaultSetup != nil {
		wrote, err := applyCodeQLDefaultSetup(&result, repo, desired.Security, write)
		if err != nil {
			return result, err
		}
		hasWrites = hasWrites || wrote
	}
	if desired.Security.AIScanPullRequests != nil {
		wrote, err := applyAIScanPullRequests(&result, repo, desired.Security, write)
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
	return desired.PullRequestExemptions != nil || !settings.EnvironmentVariablesEmpty(desired.EnvironmentVariables) || (desired.ActionsPolicies != nil && desired.ActionsPolicies.Len() > 0) || desired.CopilotReviewEffort != nil || !desired.PullRequestLimits.Empty() || desired.CopilotMCP != nil || desired.CopilotInstructions != nil || (desired.Variables != nil && desired.Variables.Len() > 0) || (desired.AgentsVariables != nil && desired.AgentsVariables.Len() > 0) || !other.Empty() || desired.Topics.Names != nil || (desired.Labels != nil && desired.Labels.Len() > 0) || (desired.Rulesets != nil && desired.Rulesets.Len() > 0) || (desired.Environments != nil && desired.Environments.Len() > 0) || !desired.Actions.Empty() || !desired.Security.Empty() || !desired.Pages.Empty()
}
