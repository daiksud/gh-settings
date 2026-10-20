package github

import (
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

// ObserveRepository creates only the clients required by the requested fields.
// Each selected API still validates its complete response before observation.
func ObserveRepository(repo repository.Repository, fields []settings.FieldID) (settings.RepositoryObservation, error) {
	return observeRepository(repo, fields,
		func(repo repository.Repository) (settings.RepositoryState, error) {
			client, err := NewReadClient(repo.Host)
			if err != nil {
				return settings.RepositoryState{}, err
			}
			return client.ReadRepository(repo)
		},
		func(repo repository.Repository) (settings.MergeMethodsState, error) {
			client, err := NewMergeMethodsReadClient(repo.Host)
			if err != nil {
				return settings.MergeMethodsState{}, err
			}
			return client.ReadMergeMethods(repo)
		},
		func(repo repository.Repository) (settings.PRAccessState, error) {
			client, err := NewPRAccessClient(repo.Host)
			if err != nil {
				return settings.PRAccessState{}, err
			}
			return client.ReadPRAccess(repo)
		},
	)
}

func observeRepository(repo repository.Repository, fields []settings.FieldID, readBase func(repository.Repository) (settings.RepositoryState, error), readMerge func(repository.Repository) (settings.MergeMethodsState, error), readPR func(repository.Repository) (settings.PRAccessState, error)) (settings.RepositoryObservation, error) {
	baseNeeded, mergeNeeded, prNeeded, err := settings.RepositoryObservationGroups(fields)
	if err != nil {
		return settings.RepositoryObservation{}, err
	}
	var base *settings.RepositoryState
	var merge *settings.MergeMethodsState
	if baseNeeded {
		state, err := readBase(repo)
		if err != nil {
			return settings.RepositoryObservation{}, err
		}
		base = &state
	}
	if mergeNeeded {
		state, err := readMerge(repo)
		if err != nil {
			return settings.RepositoryObservation{}, err
		}
		merge = &state
	}
	var pr *settings.PRAccessState
	if prNeeded {
		state, err := readPR(repo)
		if err != nil {
			return settings.RepositoryObservation{}, err
		}
		pr = &state
	}
	return settings.NewRepositoryObservation(base, merge, pr), nil
}
