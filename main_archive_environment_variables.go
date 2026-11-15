package main

import (
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/github"
	"github.com/daiksud/gh-settings/internal/settings"
)

func readArchivedEnvironmentVariables(repo repository.Repository, environment string, names []string) (int64, []settings.VariableObservation, error) {
	client, err := github.NewVariablesClient(repo.Host)
	if err != nil {
		return 0, nil, err
	}
	return client.ReadArchivedEnvironmentVariables(repo, environment, names)
}
