package app

import (
	"fmt"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func applyRulesets(result *ApplyResult, repo repository.Repository, desired *settings.RulesetDeclarations, write Writer) (bool, error) {
	index := len(result.Operations)
	result.Operations = append(result.Operations, ApplyOperation{Kind: RulesetsRead})
	observed, err := write.ReadRulesets(repo)
	if err != nil {
		result.failOperation(index, OperationFailedRead)
		return false, fmt.Errorf("read Rulesets for apply (earlier writes remain applied): %w", err)
	}
	changes, err := settings.ReconcileRulesets(observed, desired)
	if err != nil {
		result.failOperation(index, OperationFailedRead)
		return false, fmt.Errorf("prepare Rulesets (earlier writes remain applied): %w", err)
	}
	result.Operations[index].Status = OperationCompleted
	first := len(result.Operations)
	for _, change := range changes {
		var kind ApplyOperationKind
		switch change.Operation() {
		case "create":
			kind = RulesetCreate
		case "update":
			kind = RulesetUpdate
		case "delete":
			kind = RulesetDelete
		}
		result.Operations = append(result.Operations, ApplyOperation{Kind: kind, RulesetName: change.Name()})
	}
	for i, change := range changes {
		if err := write.Ruleset(repo, change); err != nil {
			result.failOperation(first+i, OperationFailedUnknown)
			return false, fmt.Errorf("%s Ruleset %q (earlier successful writes remain applied): %w", change.Operation(), change.Name(), err)
		}
		result.Operations[first+i].Status = OperationCompleted
	}
	return len(changes) > 0, nil
}
