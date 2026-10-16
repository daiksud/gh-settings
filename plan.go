package main

import (
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/app"
	"github.com/daiksud/gh-settings/internal/cli"
	"github.com/daiksud/gh-settings/internal/github"
	"github.com/daiksud/gh-settings/internal/settings"
)

func planCurrent() ([]byte, error) {
	result, err := app.Plan(func(repo repository.Repository) (settings.RepositoryState, error) {
		client, err := github.NewReadClient(repo.Host)
		if err != nil {
			return settings.RepositoryState{}, err
		}
		return client.ReadRepository(repo)
	})
	if err != nil {
		return nil, err
	}
	return cli.RenderRepositoryComparison(result.Comparison), nil
}
