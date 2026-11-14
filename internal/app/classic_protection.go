package app

import (
	"errors"
	"fmt"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func applyClassicProtection(result *ApplyResult, repo repository.Repository, desired []settings.ClassicRule, write Writer, prune bool, prepared settings.ClassicPruneObservation) (bool, error) {
	index := len(result.Operations)
	result.Operations = append(result.Operations, ApplyOperation{Kind: ClassicProtectionRead})
	observed, err := write.ReadBranchProtectionRules(repo)
	if err == nil && prune {
		err = prepared.Recheck(observed)
	}
	var selected []settings.ClassicSelection
	if err == nil {
		selected, err = settings.PrepareClassicRules(observed, desired)
	}
	if err != nil {
		result.failOperation(index, OperationFailedRead)
		return false, fmt.Errorf("prepare classic protection: %w", err)
	}
	result.Operations[index].Status = OperationCompleted
	known := map[int64]bool{}
	for _, s := range observed.Rules {
		known[*s.Rule.ID] = true
	}
	first := len(result.Operations)
	for _, s := range selected {
		kind := ClassicProtectionUpdate
		if s.Before == nil {
			kind = ClassicProtectionCreate
		}
		result.Operations = append(result.Operations, ApplyOperation{Kind: kind, ClassicPattern: s.Desired.BranchNamePattern})
	}
	wrote := false
	for i, s := range selected {
		applied, err := write.BranchProtectionRule(repo, s)
		if s.Before == nil {
			if applied.ID > 0 && !known[applied.ID] {
				result.CreatedIDs = append(result.CreatedIDs, CreatedObject{Collection: "branch_protection_rules", Name: s.Desired.BranchNamePattern, ID: applied.ID})
			}
			if err == nil && (applied.ID <= 0 || known[applied.ID] || !applied.Wrote) {
				err = fmt.Errorf("classic creation lacks a new confirmed identity; recovery node %q id %d", applied.NodeID, applied.ID)
			}
		}
		if err != nil {
			status := OperationFailedUnknown
			var pre PreWriteReadFailure
			if errors.As(err, &pre) {
				status = OperationFailedRead
			}
			result.failOperation(first+i, status)
			uncertainty := ""
			if status == OperationFailedUnknown {
				uncertainty = " (unconfirmed effects may remain)"
			}
			return wrote, fmt.Errorf("classic protection %q%s: %w", s.Desired.BranchNamePattern, uncertainty, err)
		}
		result.Operations[first+i].Status = OperationUnchanged
		if applied.Wrote {
			result.Operations[first+i].Status = OperationCompleted
			wrote = true
		}
		if s.Before == nil {
			known[applied.ID] = true
			if write.CreatedID != nil {
				if err := write.CreatedID("branch_protection_rules", s.Desired.BranchNamePattern, applied.ID); err != nil {
					result.Outcome = ApplyPartialFailure
					return wrote, fmt.Errorf("classic rule %q was created with id %d but local ID writeback failed: %w", s.Desired.BranchNamePattern, applied.ID, err)
				}
			}
		}
	}
	if prune {
		deleted, err := applyClassicPrune(result, repo, prepared, write)
		return wrote || deleted, err
	}
	return wrote, nil
}

func applyClassicPrune(result *ApplyResult, repo repository.Repository, prepared settings.ClassicPruneObservation, write Writer) (bool, error) {
	candidates, err := prepared.Candidates()
	if err != nil {
		return false, err
	}
	first := len(result.Operations)
	for _, candidate := range candidates {
		result.Operations = append(result.Operations, ApplyOperation{Kind: ClassicProtectionDelete, ClassicPattern: candidate.Pattern()})
	}
	wrote := false
	for i, candidate := range candidates {
		if err := write.DeleteBranchProtectionRule(repo, candidate); err != nil {
			status := OperationFailedUnknown
			var pre PreWriteReadFailure
			uncertainty := " (unconfirmed effects may remain)"
			if errors.As(err, &pre) {
				status, uncertainty = OperationFailedRead, ""
			}
			result.failOperation(first+i, status)
			return wrote, fmt.Errorf("delete classic protection %q id %d node %q%s: %w", candidate.Pattern(), candidate.ID(), candidate.NodeID(), uncertainty, err)
		}
		result.Operations[first+i].Status = OperationCompleted
		wrote = true
	}
	return wrote, nil
}
