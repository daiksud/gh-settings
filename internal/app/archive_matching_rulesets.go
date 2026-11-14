package app

import (
	"fmt"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

// A Ruleset declaration selects a whole object. Reuse the ordinary apply
// comparison so identity, owned defaults, and normalized observations have the
// same meaning while ensuring this path performs no writes or ID binding.
func verifyArchivedRulesets(repo repository.Repository, desired *settings.RulesetDeclarations, read Writer) error {
	if desired == nil || desired.Len() == 0 {
		return nil
	}
	if read.ReadRulesets == nil {
		return fmt.Errorf("verify archived Rulesets: reader unavailable")
	}
	current, err := read.ReadRulesets(repo)
	if err != nil {
		return fmt.Errorf("verify archived Rulesets: %w", err)
	}
	changes, err := settings.ReconcileRulesets(current, desired)
	if err != nil {
		return fmt.Errorf("verify archived Rulesets: %w", err)
	}
	if len(changes) != 0 {
		return fmt.Errorf("archived repository Rulesets differ")
	}
	return nil
}
