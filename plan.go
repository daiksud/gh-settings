package main

import (
	"fmt"
	"time"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/cli"
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
	client, err := api.NewRESTClient(api.ClientOptions{Host: repo.Host, Timeout: 30 * time.Second})
	if err != nil {
		return nil, fmt.Errorf("configure GitHub API client: %w", err)
	}
	return planSettings(desired, repo, client)
}

func planSettings(desired settings.RepositoryDeclaration, repo repository.Repository, client *api.RESTClient) ([]byte, error) {
	current, err := readRepositorySettings(repo, client)
	if err != nil {
		return nil, err
	}
	return cli.RenderRepositoryComparison(settings.CompareRepository(current, desired)), nil
}
