package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func applyCurrent() ([]byte, error) {
	desired, err := loadCurrentConfig()
	if err != nil {
		return nil, err
	}
	if desired == (settings.RepositoryDeclaration{}) {
		return []byte("No managed repository settings.\n"), nil
	}
	repo, err := currentRepository()
	if err != nil {
		return nil, err
	}
	client, err := api.NewRESTClient(api.ClientOptions{
		Host: repo.Host, Timeout: 30 * time.Second,
		// Redirects can convert PATCH to GET and report success without updating.
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	})
	if err != nil {
		return nil, fmt.Errorf("configure GitHub API client: %w", err)
	}
	return applySettings(desired, repo, client)
}

func applySettings(desired settings.RepositoryDeclaration, repo repository.Repository, client *api.RESTClient) ([]byte, error) {
	payload := struct {
		Description *string `json:"description,omitempty"`
		Homepage    *string `json:"homepage,omitempty"`
		HasIssues   *bool   `json:"has_issues,omitempty"`
		HasProjects *bool   `json:"has_projects,omitempty"`
		HasWiki     *bool   `json:"has_wiki,omitempty"`
	}{
		Description: desired.Description, Homepage: desired.Homepage,
		HasIssues: desired.HasIssues, HasProjects: desired.HasProjects, HasWiki: desired.HasWiki,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("encode repository settings: %w", err)
	}
	if err := client.Patch("repos/"+repo.Owner+"/"+repo.Name, bytes.NewReader(body), nil); err != nil {
		return nil, fmt.Errorf("apply repository settings: %w", err)
	}
	return []byte("Applied repository settings.\n"), nil
}
