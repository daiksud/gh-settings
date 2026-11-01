package main

import (
	"fmt"
	"os"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/github"
	"github.com/daiksud/gh-settings/internal/settings"
)

func dependabotClient(repo repository.Repository) (*github.DependabotClient, error) {
	c, err := github.NewDependabotClient(repo.Host)
	if err != nil {
		return nil, err
	}
	c.Warning = func(message string) { fmt.Fprintln(os.Stderr, "Warning: "+message) }
	return c, nil
}

func readDependabot(repo repository.Repository) (settings.DependabotObservation, error) {
	c, err := dependabotClient(repo)
	if err != nil {
		return settings.DependabotObservation{}, err
	}
	return c.Read(repo)
}

func writeDependabot(repo repository.Repository, change settings.SecurityChange) (settings.DependabotWriteResult, error) {
	c, err := dependabotClient(repo)
	if err != nil {
		return settings.DependabotWriteResult{}, err
	}
	return c.Update(repo, change)
}
