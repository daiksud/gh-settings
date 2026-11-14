package app

import (
	"fmt"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

// archivedGroup uses the same typed reader and comparator as reconciliation.
// A selected group must be observed completely and contain no write-required
// normalization or change. This is evidence only for its selected ownership.
func archivedGroup[O, D, C any](repo repository.Repository, desired D, read func(repository.Repository) (O, error), compare func(O, D) (int, []C, error), name string) error {
	if read == nil {
		return fmt.Errorf("verify archived %s: reader unavailable", name)
	}
	observed, err := read(repo)
	if err != nil {
		return fmt.Errorf("verify archived %s: %w", name, err)
	}
	managed, changes, err := compare(observed, desired)
	if err != nil {
		return fmt.Errorf("verify archived %s: %w", name, err)
	}
	if managed == 0 || len(changes) != 0 {
		return fmt.Errorf("archived %s differs or lacks selected evidence", name)
	}
	return nil
}

func verifyArchivedGroups(repo repository.Repository, desired settings.Declaration, read Writer) error {
	if !desired.PullRequestLimits.Empty() {
		if err := archivedGroup(repo, desired.PullRequestLimits, read.ReadPullRequestLimits, settings.ComparePullRequestLimits, "pull request limits"); err != nil {
			return err
		}
	}
	if desired.PullRequestExemptions != nil {
		if err := archivedGroup(repo, desired.PullRequestExemptions, read.ReadPullRequestExemptions, settings.ComparePullRequestExemptions, "pull request exemptions"); err != nil {
			return err
		}
	}
	if desired.CopilotReviewEffort != nil {
		if read.ReadCopilotReviewEffort == nil {
			return fmt.Errorf("verify archived Copilot review effort: reader unavailable")
		}
		observation, err := read.ReadCopilotReviewEffort(repo)
		if err != nil {
			return fmt.Errorf("verify archived Copilot review effort: %w", err)
		}
		configured, err := observation.Configured()
		if err != nil || configured == nil {
			return fmt.Errorf("verify archived Copilot review effort: configured override cannot be established")
		}
		readOnce := func(repository.Repository) (settings.CopilotReviewEffortObservation, error) { return observation, nil }
		if err := archivedGroup(repo, desired.CopilotReviewEffort, readOnce, settings.CompareCopilotReviewEffort, "Copilot review effort"); err != nil {
			return err
		}
	}
	if desired.Actions.RetentionDays != nil {
		if err := archivedGroup(repo, desired.Actions, read.ReadRetention, settings.CompareRetention, "Actions retention"); err != nil {
			return err
		}
	}
	if desired.Actions.ForkApprovalPolicy != nil {
		if err := archivedGroup(repo, desired.Actions, read.ReadForkApproval, settings.CompareForkApproval, "Actions fork approval"); err != nil {
			return err
		}
	}
	if !desired.Pages.Empty() {
		if err := archivedGroup(repo, desired.Pages, read.ReadPages, settings.ComparePages, "Pages"); err != nil {
			return err
		}
	}
	return nil
}
