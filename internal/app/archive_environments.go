package app

import (
	"fmt"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

// verifyArchivedEnvironments acquires and compares selected existing
// Environments twice. Matching owns only the values in each declaration;
// separate reads do not provide an atomic snapshot.
func verifyArchivedEnvironments(repo repository.Repository, desired *settings.EnvironmentDeclarations, read func(repository.Repository, []string) (settings.EnvironmentObservation, error)) error {
	if desired == nil || desired.Len() == 0 {
		return nil
	}
	if read == nil {
		return fmt.Errorf("verify archived environments: reader unavailable")
	}
	names := environmentNames(desired)
	observe := func() (map[string]int64, error) {
		observation, err := read(repo, names)
		if err != nil {
			return nil, err
		}
		changes, err := settings.ReconcileEnvironments(observation, desired)
		if err != nil {
			return nil, err
		}
		if len(changes) != 0 {
			return nil, fmt.Errorf("selected environments differ or require creation")
		}
		states, err := observation.Values()
		if err != nil || len(states) != len(names) {
			return nil, fmt.Errorf("incomplete selected environment observation")
		}
		ids := make(map[string]int64, len(states))
		for _, state := range states {
			ids[settings.EnvironmentIdentity(state.Policy.Name)] = state.ID
		}
		for _, name := range names {
			if ids[settings.EnvironmentIdentity(name)] <= 0 {
				return nil, fmt.Errorf("selected environment identity unavailable")
			}
		}
		return ids, nil
	}
	first, err := observe()
	if err != nil {
		return fmt.Errorf("verify archived environments: %w", err)
	}
	fresh, err := observe()
	if err != nil {
		return fmt.Errorf("verify archived environments: %w", err)
	}
	for name, id := range first {
		if fresh[name] != id {
			return fmt.Errorf("verify archived environments: identity changed")
		}
	}
	return nil
}
