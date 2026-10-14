package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/cli/go-gh/v2/pkg/repository"
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

func planSettings(desired settings, repo repository.Repository, client *api.RESTClient) ([]byte, error) {
	current, err := readRepositorySettings(repo, client)
	if err != nil {
		return nil, err
	}
	return repositoryPlan(current, desired), nil
}

// current contains all five fields after readRepositorySettings succeeds.
func repositoryPlan(current, desired settings) []byte {
	var output strings.Builder
	managed := 0
	stringField := func(name string, current, desired *string) {
		if desired == nil {
			return
		}
		managed++
		if *current != *desired {
			fmt.Fprintf(&output, "settings.repository.%s: %s → %s\n", name, strconv.Quote(*current), strconv.Quote(*desired))
		}
	}
	boolField := func(name string, current, desired *bool) {
		if desired == nil {
			return
		}
		managed++
		if *current != *desired {
			fmt.Fprintf(&output, "settings.repository.%s: %t → %t\n", name, *current, *desired)
		}
	}
	stringField("description", current.Description, desired.Description)
	stringField("homepage", current.Homepage, desired.Homepage)
	boolField("has_issues", current.HasIssues, desired.HasIssues)
	boolField("has_projects", current.HasProjects, desired.HasProjects)
	boolField("has_wiki", current.HasWiki, desired.HasWiki)
	if managed == 0 {
		output.WriteString("No managed repository settings.\n")
	} else if output.Len() == 0 {
		output.WriteString("No differences for managed repository settings.\n")
	}
	return []byte(output.String())
}
