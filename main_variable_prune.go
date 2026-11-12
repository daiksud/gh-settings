package main

import (
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/github"
	"github.com/daiksud/gh-settings/internal/settings"
)

func readVariables(repo repository.Repository) (settings.VariableInventory, error) {
	c, err := github.NewVariablesClient(repo.Host)
	if err != nil {
		return settings.VariableInventory{}, err
	}
	return c.ReadVariables(repo)
}
func readAgentVariables(repo repository.Repository) (settings.VariableInventory, error) {
	c, err := github.NewVariablesClient(repo.Host)
	if err != nil {
		return settings.VariableInventory{}, err
	}
	return c.ReadAgentVariables(repo)
}
func readEnvironmentVariables(repo repository.Repository, environment string) (settings.VariableInventory, error) {
	c, err := github.NewVariablesClient(repo.Host)
	if err != nil {
		return settings.VariableInventory{}, err
	}
	return c.ReadEnvironmentVariables(repo, environment)
}
func deleteVariable(repo repository.Repository, change settings.VariableChange) error {
	c, err := github.NewVariablesClient(repo.Host)
	if err != nil {
		return &variableClientReadFailure{err}
	}
	return c.DeleteVariable(repo, change)
}
func deleteAgentVariable(repo repository.Repository, change settings.VariableChange) error {
	c, err := github.NewVariablesClient(repo.Host)
	if err != nil {
		return &variableClientReadFailure{err}
	}
	return c.DeleteAgentVariable(repo, change)
}
func deleteEnvironmentVariable(repo repository.Repository, environment string, id int64, change settings.VariableChange) error {
	c, err := github.NewVariablesClient(repo.Host)
	if err != nil {
		return &variableClientReadFailure{err}
	}
	return c.DeleteEnvironmentVariable(repo, environment, id, change)
}
