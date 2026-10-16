package main

import (
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/github"
	"github.com/daiksud/gh-settings/internal/settings"
)

func readRepository(repo repository.Repository) (settings.RepositoryState, error) {
	client, err := github.NewReadClient(repo.Host)
	if err != nil {
		return settings.RepositoryState{}, err
	}
	return client.ReadRepository(repo)
}
