package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func planCurrent() ([]byte, error) {
	config, err := loadCurrentConfig()
	if err != nil {
		return nil, err
	}
	repo, err := currentRepository()
	if err != nil {
		return nil, err
	}
	client, err := api.NewRESTClient(api.ClientOptions{Host: repo.Host, Timeout: 30 * time.Second})
	if err != nil {
		return nil, fmt.Errorf("configure GitHub API client: %w", err)
	}
	return planSettings(config.Settings.Repository, repo, client)
}

func planSettings(desired settings.RepositoryDeclaration, repo repository.Repository, client *api.RESTClient) ([]byte, error) {
	current, err := readRepositorySettings(repo, client)
	if err != nil {
		return nil, err
	}
	return renderRepositoryComparison(settings.CompareRepository(current, desired)), nil
}

func renderRepositoryComparison(comparison settings.RepositoryComparison) []byte {
	var output strings.Builder
	if comparison.Managed == 0 {
		output.WriteString("No managed repository settings.\n")
	} else if len(comparison.Changes) == 0 {
		output.WriteString("No differences for managed repository settings.\n")
	} else {
		for _, change := range comparison.Changes {
			fmt.Fprintf(&output, "settings.repository.%s: %#v → %#v\n", change.Field, change.Before, change.After)
		}
	}
	return []byte(output.String())
}
