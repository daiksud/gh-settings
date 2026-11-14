package app

import (
	"errors"
	"fmt"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func applyDeployKeys(result *ApplyResult, repo repository.Repository, desired *settings.DeployKeyDeclarations, prepared *settings.DeployKeyObservation, prune, unarchived bool, write Writer) (bool, error) {
	index := len(result.Operations)
	result.Operations = append(result.Operations, ApplyOperation{Kind: DeployKeysRead})
	expected, err := write.ReadDeployKeys(repo)
	if err == nil && prepared != nil {
		frozen := *prepared
		if unarchived {
			frozen, err = frozen.AfterOwnUnarchive()
		}
		if err == nil && !settings.DeployKeyInventoryEqual(frozen, expected) {
			err = fmt.Errorf("deploy key inventory changed after global prune preflight")
		}
	}
	var selected []settings.DeployKeySelection
	var changes []settings.DeployKeyChange
	if err == nil {
		selected, changes, err = settings.PrepareDeployKeys(expected, desired, prune)
	}
	if err != nil {
		result.failOperation(index, OperationFailedRead)
		return false, fmt.Errorf("prepare deploy keys: %w", err)
	}
	result.Operations[index].Status = OperationCompleted
	wrote := false
	fail := func(index int, err error) (bool, error) {
		status := OperationFailedUnknown
		var pre PreWriteReadFailure
		if errors.As(err, &pre) {
			status = OperationFailedRead
		}
		result.failOperation(index, status)
		return wrote, fmt.Errorf("apply deploy keys: %w", err)
	}
	for _, selection := range selected {
		if !selection.Create() {
			continue
		}
		d := selection.Desired()
		index := len(result.Operations)
		result.Operations = append(result.Operations, ApplyOperation{Kind: DeployKeyCreate, DeployKeyFingerprint: d.Key.Fingerprint()})
		applied, err := write.CreateDeployKey(repo, expected, d)
		newID := applied.ID > 0 && !expected.HasID(applied.ID)
		if newID {
			result.CreatedIDs = append(result.CreatedIDs, CreatedObject{Collection: "deploy_keys", Name: d.Title + " (" + d.Key.Fingerprint() + ")", ID: applied.ID})
		}
		if err == nil {
			state, found := applied.Observation.State(applied.ID)
			want, preservationErr := expected.WithCreated(state)
			if !newID || !found || !settings.DeployKeyConfigurationMatches(state, d) || preservationErr != nil || !settings.DeployKeyInventoryEqual(want, applied.Observation) {
				err = fmt.Errorf("deploy key creation did not provide a verified new identity; remote effects may remain")
			}
		}
		if err != nil {
			return fail(index, err)
		}
		result.Operations[index].Status = OperationCompleted
		wrote = true
		expected = applied.Observation
		if write.CreatedID != nil {
			if err := write.CreatedID("deploy_keys", d.Key.Fingerprint(), applied.ID); err != nil {
				result.Outcome = ApplyPartialFailure
				return wrote, fmt.Errorf("deploy key %s was created with id %d but local ID binding failed: %w", d.Key.Fingerprint(), applied.ID, err)
			}
		}
	}
	// Candidates were selected before creation. Verified own creations are retained
	// in expected inventory and must never become newly inferred prune candidates.
	for _, change := range changes {
		if change.Operation() != "delete" {
			continue
		}
		before, _ := change.Values()
		id := *before.ID
		index := len(result.Operations)
		result.Operations = append(result.Operations, ApplyOperation{Kind: DeployKeyDelete, DeployKeyFingerprint: before.Fingerprint})
		after, err := write.DeleteDeployKey(repo, expected, id)
		if err != nil {
			return fail(index, err)
		}
		want, err := expected.Without(id)
		if err != nil || !settings.DeployKeyInventoryEqual(want, after) {
			return fail(index, fmt.Errorf("deploy key deletion did not confirm expected absence; remote effects may remain"))
		}
		expected = after
		result.Operations[index].Status = OperationCompleted
		wrote = true
	}
	return wrote, nil
}
