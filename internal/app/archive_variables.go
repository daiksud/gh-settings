package app

import (
	"fmt"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

// The two families own only the selected names; no inventory or writes are
// needed. Do not include observed values or transport errors in diagnostics.
func verifyArchivedVariables(repo repository.Repository, actions, agents *settings.VariableDeclarations, read Writer) error {
	for _, family := range []struct {
		name    string
		desired *settings.VariableDeclarations
		read    func(repository.Repository, string) (settings.VariableObservation, error)
	}{
		{"Actions", actions, read.ReadArchivedVariable},
		{"Agents", agents, read.ReadArchivedAgentVariable},
	} {
		if family.desired == nil || family.desired.Len() == 0 {
			continue
		}
		if family.read == nil {
			return fmt.Errorf("verify archived %s variables: reader unavailable", family.name)
		}
		for _, selected := range settings.SortedVariables(*family.desired) {
			observed, err := family.read(repo, selected.Name)
			if err != nil {
				return fmt.Errorf("verify archived %s variables: read failed", family.name)
			}
			change, err := settings.CompareVariable(observed, selected)
			if err != nil {
				return fmt.Errorf("verify archived %s variables: invalid observation", family.name)
			}
			if change != nil {
				return fmt.Errorf("archived %s variables differ", family.name)
			}
		}
	}
	return nil
}
