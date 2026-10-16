package main

import (
	"encoding/json"
	"fmt"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func readRepositorySettings(repo repository.Repository, client *api.RESTClient) (settings.RepositoryState, error) {
	var response map[string]json.RawMessage
	if err := client.Get("repos/"+repo.Owner+"/"+repo.Name, &response); err != nil {
		return settings.RepositoryState{}, fmt.Errorf("read repository settings: %w", err)
	}
	return decodeRepositoryResponse(response)
}

func decodeRepositoryResponse(response map[string]json.RawMessage) (settings.RepositoryState, error) {
	var result settings.RepositoryState
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
			return settings.RepositoryState{}, fmt.Errorf("GitHub response is missing %s", field.name)
		}
		if string(value) == "null" {
			// GET uses null for unset strings; the complete state uses empty strings.
			if field.name != "description" && field.name != "homepage" {
				return settings.RepositoryState{}, fmt.Errorf("GitHub returned null for %s", field.name)
			}
			continue
		}
		if err := json.Unmarshal(value, field.target); err != nil {
			return settings.RepositoryState{}, fmt.Errorf("decode %s: %w", field.name, err)
		}
	}
	return result, nil
}
