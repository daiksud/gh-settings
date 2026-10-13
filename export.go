package main

import (
	"encoding/json"
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
	var response map[string]json.RawMessage
	if err := client.Get("repos/"+repo.Owner+"/"+repo.Name, &response); err != nil {
		return nil, fmt.Errorf("read repository settings: %w", err)
	}
	result, err := decodeRepositoryResponse(response)
	if err != nil {
		return nil, err
	}
	return yaml.Marshal(map[string]any{"schemaVersion": 1, "settings": map[string]any{"repository": result}})
}

func decodeRepositoryResponse(response map[string]json.RawMessage) (settings, error) {
	var result settings
	fields := []struct {
		name   string
		target any
	}{
		{"description", &result.Description}, {"homepage", &result.Homepage},
		{"has_issues", &result.HasIssues}, {"has_projects", &result.HasProjects}, {"has_wiki", &result.HasWiki},
	}
	for _, field := range fields {
		value, present := response[field.name]
		if !present {
			return settings{}, fmt.Errorf("GitHub response is missing %s", field.name)
		}
		if string(value) == "null" {
			if field.name == "description" || field.name == "homepage" {
				// GET uses null for unset strings; export uses the update schema's empty string.
				continue
			}
			return settings{}, fmt.Errorf("GitHub returned null for %s", field.name)
		}
		if err := json.Unmarshal(value, field.target); err != nil {
			return settings{}, fmt.Errorf("decode %s: %w", field.name, err)
		}
	}
	return result, nil
}

type settings struct {
	Description string `yaml:"description"`
	Homepage    string `yaml:"homepage"`
	HasIssues   bool   `yaml:"has_issues"`
	HasProjects bool   `yaml:"has_projects"`
	HasWiki     bool   `yaml:"has_wiki"`
}
