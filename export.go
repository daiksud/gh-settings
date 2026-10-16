package main

import (
	"fmt"
	"time"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/workspace"
)

func exportCurrent() ([]byte, error) {
	repo, err := currentRepository()
	if err != nil {
		return nil, err
	}
	root, err := gitRoot()
	if err != nil {
		return nil, fmt.Errorf("locate %s: %w", workspace.SettingsPath, err)
	}
	client, err := api.NewRESTClient(api.ClientOptions{Host: repo.Host, Timeout: 30 * time.Second})
	if err != nil {
		return nil, fmt.Errorf("configure GitHub API client: %w", err)
	}
	data, err := exportSettings(repo, client)
	if err != nil {
		return nil, err
	}
	return nil, workspace.WriteConfig(root, data)
}

func exportSettings(repo repository.Repository, client *api.RESTClient) ([]byte, error) {
	result, err := readRepositorySettings(repo, client)
	if err != nil {
		return nil, err
	}
	declaration := settings.RepositoryDeclaration{
		Description: &result.Description, Homepage: &result.Homepage,
		HasIssues: &result.HasIssues, HasProjects: &result.HasProjects, HasWiki: &result.HasWiki,
	}
	return workspace.EncodeConfig(declaration)
}
