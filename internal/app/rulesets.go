package app

import (
	"errors"
	"fmt"
	"slices"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func applyRulesets(result *ApplyResult, repo repository.Repository, desired *settings.RulesetDeclarations, write Writer, prune bool, initial []settings.RulesetChange) (bool, error) {
	index := len(result.Operations)
	result.Operations = append(result.Operations, ApplyOperation{Kind: RulesetsRead})
	observed, err := write.ReadRulesets(repo)
	if err != nil {
		result.failOperation(index, OperationFailedRead)
		return false, fmt.Errorf("read Rulesets for apply: %w", err)
	}
	changes, err := settings.ReconcileRulesetsWithPrune(observed, desired, prune)
	if err != nil {
		result.failOperation(index, OperationFailedRead)
		return false, fmt.Errorf("prepare Rulesets: %w", err)
	}
	if prune && !sameRulesetDeletions(initial, changes) {
		result.failOperation(index, OperationFailedRead)
		return false, fmt.Errorf("Ruleset prune candidates changed after preflight; refusing collection writes")
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
	slices.SortStableFunc(changes, func(a, b settings.RulesetChange) int {
		if a.Operation() == "delete" && b.Operation() != "delete" {
			return 1
		}
		if a.Operation() != "delete" && b.Operation() == "delete" {
			return -1
		}
		return 0
	})
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
		id, err := write.Ruleset(repo, change)
		if change.Operation() == "create" {
			if id > 0 {
				result.CreatedIDs = append(result.CreatedIDs, CreatedObject{Collection: "rulesets", Name: change.Name(), ID: id})
			}
			if err == nil && id <= 0 {
				err = fmt.Errorf("creation returned no positive ID; remote creation may have succeeded")
			}
			if err == nil && knownIDs[id] {
				err = fmt.Errorf("creation returned already-known Ruleset id %d; refusing ID adoption (writes may already have applied)", id)
			}
		}
		if err != nil {
			status := OperationFailedUnknown
			var readFailure PreWriteReadFailure
			if errors.As(err, &readFailure) {
				status = OperationFailedRead
			}
			result.failOperation(first+i, status)
			return false, fmt.Errorf("%s Ruleset %q: %w", change.Operation(), change.Name(), err)
		}
		result.Operations[first+i].Status = OperationCompleted
		if change.Operation() == "delete" {
			delete(knownIDs, change.ID())
		} else {
			knownIDs[id] = true
		}
		if change.Operation() == "create" && write.CreatedID != nil {
			if err := write.CreatedID("rulesets", change.Name(), id); err != nil {
				result.Outcome = ApplyPartialFailure
				return false, fmt.Errorf("Ruleset %q was created with id %d, but local ID writeback failed: %w; add id: %d before retrying", change.Name(), id, err, id)
			}
		}
	}
	return len(changes) > 0, nil
}
