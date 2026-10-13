package main

import (
	"encoding/json"
	"fmt"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/cli/go-gh/v2/pkg/repository"
)

func readRepositorySettings(repo repository.Repository, client *api.RESTClient) (settings, error) {
	var response map[string]json.RawMessage
	if err := client.Get("repos/"+repo.Owner+"/"+repo.Name, &response); err != nil {
		return settings{}, fmt.Errorf("read repository settings: %w", err)
	}
	return decodeRepositoryResponse(response)
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
			switch field.name {
			case "description":
				// GET uses null for unset strings; export uses the update schema's empty string.
				result.Description = new(string)
			case "homepage":
				result.Homepage = new(string)
			default:
				return settings{}, fmt.Errorf("GitHub returned null for %s", field.name)
			}
			continue
		}
		if err := json.Unmarshal(value, field.target); err != nil {
			return settings{}, fmt.Errorf("decode %s: %w", field.name, err)
		}
	}
	return result, nil
}
