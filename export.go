package main

import (
	"fmt"
	"time"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/cli/go-gh/v2/pkg/repository"
	"go.yaml.in/yaml/v4"
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
	rootDir, err := openExportRoot(root)
	if err != nil {
		return nil, fmt.Errorf("open checkout for %s: %w", settingsPath, err)
	}
	defer rootDir.Close()
	client, err := api.NewRESTClient(api.ClientOptions{Host: repo.Host, Timeout: 30 * time.Second})
	if err != nil {
		return nil, fmt.Errorf("configure GitHub API client: %w", err)
	}
	data, err := exportSettings(repo, client)
	if err != nil {
		return nil, err
	}
	if err := writeExportFile(rootDir, data); err != nil {
		return nil, fmt.Errorf("write %s: %w", settingsPath, err)
	}
	return nil, nil
}

func exportSettings(repo repository.Repository, client *api.RESTClient) ([]byte, error) {
	result, err := readRepositorySettings(repo, client)
	if err != nil {
		return nil, err
	}
	return yaml.Marshal(Config{SchemaVersion: 1, Settings: configSettings{Repository: result}})
}
