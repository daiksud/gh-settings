package main

import (
	"fmt"

	"github.com/daiksud/gh-settings/internal/github"
	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/workspace"
)

func exportCurrent() ([]byte, error) {
	repo, err := workspace.CurrentRepository()
	if err != nil {
		return nil, err
	}
	root, err := workspace.GitRoot()
	if err != nil {
		return nil, fmt.Errorf("locate %s: %w", workspace.SettingsPath, err)
	}
	client, err := github.NewReadClient(repo.Host)
	if err != nil {
		return nil, err
	}
	current, err := client.ReadRepository(repo)
	if err != nil {
		return nil, err
	}
	data, err := exportSettings(current)
	if err != nil {
		return nil, err
	}
	return nil, workspace.WriteConfig(root, data)
}

func exportSettings(result settings.RepositoryState) ([]byte, error) {
	declaration := settings.RepositoryDeclaration{
		Description: &result.Description, Homepage: &result.Homepage,
		HasIssues: &result.HasIssues, HasProjects: &result.HasProjects, HasWiki: &result.HasWiki,
	}
	return workspace.EncodeConfig(declaration)
}
