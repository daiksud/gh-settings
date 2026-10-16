package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func exportCurrent() ([]byte, error) {
	repo, err := currentRepository()
	if err != nil {
		return nil, err
	}
	root, err := gitRoot()
	if err != nil {
		return nil, fmt.Errorf("locate %s: %w", settingsPath, err)
	}
	client, err := api.NewRESTClient(api.ClientOptions{Host: repo.Host, Timeout: 30 * time.Second})
	if err != nil {
		return nil, fmt.Errorf("configure GitHub API client: %w", err)
	}
	data, err := exportSettings(repo, client)
	if err != nil {
		return nil, err
	}
	path := filepath.Join(root, settingsPath)
	dir := filepath.Dir(path)
	if err := os.Mkdir(dir, 0755); err != nil && !os.IsExist(err) {
		return nil, fmt.Errorf("create .github: %w", err)
	}
	info, err := os.Lstat(dir)
	if err != nil {
		return nil, fmt.Errorf("inspect .github: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf(".github must be a directory, not a symlink or other file")
	}
	info, err = os.Lstat(path)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("inspect %s: %w", settingsPath, err)
	}
	if info != nil && !info.Mode().IsRegular() {
		return nil, fmt.Errorf("%s must be a regular file, not a symlink or other file", settingsPath)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return nil, fmt.Errorf("write %s: %w", settingsPath, err)
	}
	return nil, nil
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
	return marshalConfig(Config{SchemaVersion: 1, Settings: configSettings{Repository: declaration}})
}
