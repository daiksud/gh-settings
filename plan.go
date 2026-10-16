package main

import (
	"github.com/daiksud/gh-settings/internal/cli"
	"github.com/daiksud/gh-settings/internal/github"
	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/workspace"
)

func planCurrent() ([]byte, error) {
	desired, err := loadCurrentConfig()
	if err != nil {
		return nil, err
	}
	repo, err := workspace.CurrentRepository()
	if err != nil {
		return nil, err
	}
	client, err := github.NewReadClient(repo.Host)
	if err != nil {
		return nil, err
	}
	current, err := client.ReadRepository(repo)
	if err != nil {
		return nil, err
	}
	return cli.RenderRepositoryComparison(settings.CompareRepository(current, desired)), nil
}
