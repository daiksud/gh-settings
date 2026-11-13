package main

import (
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/github"
	"github.com/daiksud/gh-settings/internal/settings"
)

func readDeployKeys(repo repository.Repository) (settings.DeployKeyObservation, error) {
	client, err := github.NewDeployKeysClient(repo.Host)
	if err != nil {
		return settings.DeployKeyObservation{}, err
	}
	return client.ReadDeployKeys(repo)
}
func createDeployKey(repo repository.Repository, expected settings.DeployKeyObservation, desired settings.DeployKeyDeclaration) (settings.DeployKeyWriteResult, error) {
	client, err := github.NewDeployKeysClient(repo.Host)
	if err != nil {
		return settings.DeployKeyWriteResult{}, err
	}
	return client.CreateDeployKey(repo, expected, desired)
}
func deleteDeployKey(repo repository.Repository, expected settings.DeployKeyObservation, id int64) (settings.DeployKeyObservation, error) {
	client, err := github.NewDeployKeysClient(repo.Host)
	if err != nil {
		return settings.DeployKeyObservation{}, err
	}
	return client.DeleteDeployKey(repo, expected, id)
}
