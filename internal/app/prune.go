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
