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
	client, err := api.NewRESTClient(api.ClientOptions{Host: repo.Host, Timeout: 30 * time.Second})
	if err != nil {
		return nil, fmt.Errorf("configure GitHub API client: %w", err)
	}
	return exportSettings(repo, client)
}

func exportSettings(repo repository.Repository, client *api.RESTClient) ([]byte, error) {
	result, err := readRepositorySettings(repo, client)
	if err != nil {
		return nil, err
	}
	return yaml.Marshal(Config{SchemaVersion: 1, Settings: configSettings{Repository: result}})
}
