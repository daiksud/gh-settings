package app

import (
	"fmt"
	"reflect"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func operationPrune(options []settings.PruneScopes) (settings.PruneScopes, error) {
	if len(options) > 1 {
		return settings.PruneScopes{}, fmt.Errorf("prune options must be supplied once")
	}
	if len(options) == 1 {
		return options[0], nil
	}
	return settings.PruneScopes{}, nil
}

type prunePreparation struct {
	runners              *settings.RunnerPruneObservation
	webhooks             *settings.WebhookPruneObservation
	deployKeys           *settings.DeployKeyObservation
	autolinks            settings.AutolinkObservation
	collaborators        settings.CollaboratorPruneObservation
	classicProtection    settings.ClassicPruneObservation
	environments         settings.EnvironmentPruneObservation
	actionsPolicies      []settings.ActionsEventPolicyChange
	labels               []settings.LabelChange
	rulesets             []settings.RulesetChange
	variables            []settings.VariableChange
	agentsVariables      []settings.VariableChange
	environmentVariables map[string]environmentVariablePrune
}

// Prepare every requested candidate before even an unrelated or unarchive write.
// Subsequent family reads must not widen or change these deletion candidates.
func preparePrune(repo repository.Repository, desired settings.Declaration, prune settings.PruneScopes, write Writer) (prunePreparation, error) {
	var result prunePreparation
	if prune.Has("runners") && desired.Runners != nil {
		if write.ReadRunnerPrune == nil || write.DeleteRunnerPrune == nil || write.RunnerLabels == nil {
			return result, fmt.Errorf("runner prune reader or writer unavailable")
		}
		observed, err := write.ReadRunnerPrune(repo)
		if err != nil {
			return result, err
		}
		if _, err = observed.Candidates(*desired.Runners); err != nil {
			return result, err
		}
		labelObservation, err := observed.LabelsObservation()
		if err != nil {
			return result, err
		}
		if err = settings.PrepareRunnerLabels(labelObservation, *desired.Runners); err != nil {
			return result, err
		}
		if observed.Target.Archived && (desired.Repository.Archived == nil || *desired.Repository.Archived) {
			return result, fmt.Errorf("runner prune requires explicit unarchive")
		}
		result.runners = &observed
	}
	if prune.Has("webhooks") && desired.Webhooks != nil {
		if write.ReadWebhookPrune == nil || write.DeleteWebhook == nil {
			return result, fmt.Errorf("webhook prune reader or writer unavailable")
		}
		observed, err := write.ReadWebhookPrune(repo)
		if err != nil {
			return result, err
		}
		prepared, err := settings.PrepareWebhookPrune(observed, *desired.Webhooks)
		if err != nil {
			return result, err
		}
		if err := prepared.ValidateApply(desired.Repository.Archived != nil && !*desired.Repository.Archived); err != nil {
			return result, err
		}
		result.webhooks = &prepared
	}
	if prune.Has("deploy-keys") && desired.DeployKeys != nil {
		observed, err := write.ReadDeployKeys(repo)
		if err != nil {
			return result, err
		}
		if _, _, err := settings.PrepareDeployKeys(observed, desired.DeployKeys, true); err != nil {
			return result, err
		}
		target, _, err := observed.Values()
		if err != nil {
			return result, err
		}
		if target.Archived && (desired.Repository.Archived == nil || *desired.Repository.Archived) {
			return result, fmt.Errorf("deploy key pruning requires an active target or explicit unarchive")
		}
		result.deployKeys = &observed
	}

	if prune.Has("autolinks") && desired.Autolinks != nil {
		if write.ReadAutolinks == nil {
			return result, fmt.Errorf("autolink prune reader unavailable")
		}
		var err error
		result.autolinks, err = write.ReadAutolinks(repo)
		if err != nil {
			return result, err
		}
		if _, err = settings.CompareAutolinks(result.autolinks, *desired.Autolinks, true); err != nil {
			return result, err
		}
		if result.autolinks.Target.Archived && (desired.Repository.Archived == nil || *desired.Repository.Archived) {
			return result, fmt.Errorf("autolink prune requires explicit unarchive")
		}
	}
	if prune.Has("collaborators") && desired.Collaborators != nil {
		if write.ReadCollaboratorPrune == nil {
			return result, fmt.Errorf("collaborator prune reader unavailable")
		}
		var err error
		result.collaborators, err = write.ReadCollaboratorPrune(repo, *desired.Collaborators)
		if err != nil {
			return result, err
		}
		if err = result.collaborators.ValidateApply(desired.Repository.Archived != nil && !*desired.Repository.Archived); err != nil {
			return result, err
		}
	}
	if prune.Has("branch-protection-rules") && desired.BranchProtectionRules != nil {
		if write.ReadBranchProtectionRules == nil || write.DeleteBranchProtectionRule == nil {
			return result, fmt.Errorf("classic prune reader or writer unavailable")
		}
		current, err := write.ReadBranchProtectionRules(repo)
		if err != nil {
			return result, err
		}
		result.classicProtection, err = settings.PrepareClassicPrune(current, *desired.BranchProtectionRules)
		if err != nil {
			return result, err
		}
		unarchive := desired.Repository.Archived != nil && !*desired.Repository.Archived
		if err := result.classicProtection.ValidateApply(unarchive); err != nil {
			return result, err
		}
	}
	if prune.Has("actions-policies") && desired.ActionsPolicies != nil {
		current, err := write.ReadActionsPolicies(repo)
		if err != nil {
			return result, err
		}
		changes, err := settings.ReconcileActionsPoliciesWithPrune(current, desired.ActionsPolicies, true)
		if err != nil {
			return result, err
		}
		result.actionsPolicies = actionsPolicyDeletions(changes)
	}
	if prune.Has("labels") && desired.Labels != nil {
		current, err := write.ReadLabels(repo)
		if err != nil {
			return result, err
		}
		comparison, err := settings.CompareLabelsWithPrune(current, desired.Labels, true)
		if err != nil {
			return result, err
		}
		result.labels = labelDeletions(comparison.Labels.Values())
	}
	if prune.Has("rulesets") && desired.Rulesets != nil {
		current, err := write.ReadRulesets(repo)
		if err != nil {
			return result, err
		}
		changes, err := settings.ReconcileRulesetsWithPrune(current, desired.Rulesets, true)
		if err != nil {
			return result, err
		}
		result.rulesets = rulesetDeletions(changes)
	}
	if prune.Has("actions-variables") && desired.Variables != nil {
		inventory, err := write.ReadVariables(repo)
		if err != nil {
			return result, err
		}
		result.variables, err = settings.PrunedVariableChanges(inventory, desired.Variables)
		if err != nil {
			return result, err
		}
	}
	if prune.Has("agents-variables") && desired.AgentsVariables != nil {
		inventory, err := write.ReadAgentVariables(repo)
		if err != nil {
			return result, err
		}
		result.agentsVariables, err = settings.PrunedVariableChanges(inventory, desired.AgentsVariables)
		if err != nil {
			return result, err
		}
	}
	if prune.Has("environment-variables") {
		result.environmentVariables = map[string]environmentVariablePrune{}
		for _, selected := range sortedEnvironmentVariables(desired.EnvironmentVariables) {
			created, err := pruneEnvironmentWillBeCreated(repo, desired.Environments, selected.Environment, write.ReadEnvironments)
			if err != nil {
				return result, err
			}
			entry := environmentVariablePrune{newEnvironment: created}
			if !created {
				inventory, err := write.ReadEnvironmentVariables(repo, selected.Environment)
				if err != nil {
					return result, err
				}
				entry.id, err = inventory.EnvironmentID(selected.Environment)
				if err != nil {
					return result, err
				}
				entry.changes, err = settings.PrunedVariableChanges(inventory, &selected.Variables)
				if err != nil {
					return result, err
				}
			}
			result.environmentVariables[settings.EnvironmentIdentity(selected.Environment)] = entry
		}
	}

	if prune.Has("environments") && desired.Environments != nil {
		var err error
		result.environments, err = write.ReadEnvironmentPrune(repo, settings.EnvironmentRetainedNames(desired))
		if err != nil {
			return result, err
		}
		candidates, err := result.environments.Candidates()
		if err != nil {
			return result, err
		}
		if err := settings.CheckClassicDeploymentPruneConflict(desired.BranchProtectionRules, candidates); err != nil {
			return result, err
		}
		unarchive := desired.Repository.Archived != nil && !*desired.Repository.Archived
		if err = result.environments.ValidateApply(unarchive); err != nil {
			return result, err
		}
	}
	return result, nil
}

func labelDeletions(changes []settings.LabelChange) []settings.LabelChange {
	var result []settings.LabelChange
	for _, c := range changes {
		if c.Operation() == "delete" {
			result = append(result, c)
		}
	}
	return result
}
func rulesetDeletions(changes []settings.RulesetChange) []settings.RulesetChange {
	var result []settings.RulesetChange
	for _, c := range changes {
		if c.Operation() == "delete" {
			result = append(result, c)
		}
	}
	return result
}
func sameLabelDeletions(initial, fresh []settings.LabelChange) bool {
	return reflect.DeepEqual(labelDeletions(initial), labelDeletions(fresh))
}
func sameRulesetDeletions(initial, fresh []settings.RulesetChange) bool {
	return reflect.DeepEqual(rulesetDeletions(initial), rulesetDeletions(fresh))
}

func actionsPolicyDeletions(changes []settings.ActionsEventPolicyChange) []settings.ActionsEventPolicyChange {
	var result []settings.ActionsEventPolicyChange
	for _, c := range changes {
		if c.Operation() == "delete" {
			result = append(result, c)
		}
	}
	return result
}
