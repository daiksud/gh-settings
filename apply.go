package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/cli/go-gh/v2/pkg/repository"
)

func applyCurrent() ([]byte, error) {
	config, err := loadCurrentConfig()
	if err != nil {
		return nil, err
	}
	desired := config.Settings.Repository
	if desired == (settings{}) {
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

func applySettings(desired settings, repo repository.Repository, client *api.RESTClient) ([]byte, error) {
	body, err := json.Marshal(desired)
	if err != nil {
		return nil, fmt.Errorf("encode repository settings: %w", err)
	}
	if err := client.Patch("repos/"+repo.Owner+"/"+repo.Name, bytes.NewReader(body), nil); err != nil {
		return nil, fmt.Errorf("apply repository settings: %w", err)
	}
	return []byte("Applied repository settings.\n"), nil
}
