package main

import (
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/github"
	"github.com/daiksud/gh-settings/internal/settings"
)

func readAutolinks(repo repository.Repository) (settings.AutolinkObservation, error) {
	c, err := github.NewAutolinksClient(repo.Host)
	if err != nil {
		return settings.AutolinkObservation{}, err
	}
	return c.ReadAutolinks(repo)
}
func createAutolink(repo repository.Repository, expected settings.AutolinkObservation, desired settings.Autolink) (settings.AutolinkWriteResult, error) {
	c, err := github.NewAutolinksClient(repo.Host)
	if err != nil {
		return settings.AutolinkWriteResult{}, err
	}
	return c.CreateAutolink(repo, expected, desired)
}
func deleteAutolink(repo repository.Repository, expected settings.AutolinkObservation, id int64) (bool, error) {
	c, err := github.NewAutolinksClient(repo.Host)
	if err != nil {
		return false, err
	}
	return c.DeleteAutolink(repo, expected, id)
}
