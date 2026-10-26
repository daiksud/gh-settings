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
			kind = RulesetCreate
		case "update":
			kind = RulesetUpdate
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
			result.failOperation(first+i, OperationFailedUnknown)
			return false, fmt.Errorf("%s Ruleset %q (earlier successful writes remain applied): %w", change.Operation(), change.Name(), err)
		}
		result.Operations[first+i].Status = OperationCompleted
		if after := change.After(); after != nil && after.HasWriteOnlyDismissal() {
			result.Notices = append(result.Notices, fmt.Sprintf("Ruleset %q: dismissal_restriction accepted by GitHub; value not verified (write-only)", change.Name()))
		}
		knownIDs[id] = true
		if change.Operation() == "create" && write.CreatedID != nil {
			if err := write.CreatedID("rulesets", change.Name(), id); err != nil {
				result.Outcome = ApplyPartialFailure
				return false, fmt.Errorf("Ruleset %q was created with id %d, but local ID writeback failed: %w; add id: %d before retrying", change.Name(), id, err, id)
			}
		}
	}
	return len(changes) > 0, nil
}
