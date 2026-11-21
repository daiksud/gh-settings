package bindings

import (
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/github"
	"github.com/daiksud/gh-settings/internal/settings"
)

func upsertVariable(repo repository.Repository, value settings.RepositoryVariable) (bool, error) {
	client, err := github.NewVariablesClient(repo.Host)
	if err != nil {
		return false, &clientPreWriteFailure{err}
	}
	return client.UpsertVariable(repo, value)
}

func upsertAgentVariable(repo repository.Repository, value settings.RepositoryVariable) (bool, error) {
	client, err := github.NewVariablesClient(repo.Host)
	if err != nil {
		return false, &clientPreWriteFailure{err}
	}
	return client.UpsertAgentVariable(repo, value)
}

func readEnvironmentVariable(repo repository.Repository, environment, name string) (settings.VariableObservation, error) {
	client, err := github.NewVariablesClient(repo.Host)
	if err != nil {
		return settings.VariableObservation{}, err
	}
	return client.ReadEnvironmentVariable(repo, environment, name)
}

func upsertEnvironmentVariable(repo repository.Repository, environment string, value settings.RepositoryVariable) (bool, error) {
	client, err := github.NewVariablesClient(repo.Host)
	if err != nil {
		return false, &clientPreWriteFailure{err}
	}
	return client.UpsertEnvironmentVariable(repo, environment, value)
}

func deleteVariable(repo repository.Repository, change settings.VariableChange) error {
	c, err := github.NewVariablesClient(repo.Host)
	if err != nil {
		return &clientPreWriteFailure{err}
	}
	return c.DeleteVariable(repo, change)
}

func deleteAgentVariable(repo repository.Repository, change settings.VariableChange) error {
	c, err := github.NewVariablesClient(repo.Host)
	if err != nil {
		return &clientPreWriteFailure{err}
	}
	return c.DeleteAgentVariable(repo, change)
}

func deleteEnvironmentVariable(repo repository.Repository, environment string, id int64, change settings.VariableChange) error {
	c, err := github.NewVariablesClient(repo.Host)
	if err != nil {
		return &clientPreWriteFailure{err}
	}
	return c.DeleteEnvironmentVariable(repo, environment, id, change)
}

func readArchivedEnvironmentVariables(repo repository.Repository, environment string, names []string) (int64, []settings.VariableObservation, error) {
	client, err := github.NewVariablesClient(repo.Host)
	if err != nil {
		return 0, nil, err
	}
	return client.ReadArchivedEnvironmentVariables(repo, environment, names)
}
