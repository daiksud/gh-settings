package main

import (
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/github"
	"github.com/daiksud/gh-settings/internal/settings"
)

func readCacheRetention(repo repository.Repository) (settings.CacheRetentionObservation, error) {
	client, err := github.NewCacheRetentionClient(repo.Host)
	if err != nil {
		return settings.CacheRetentionObservation{}, err
	}
	return client.Read(repo)
}
func writeCacheRetention(repo repository.Repository, value int) (bool, error) {
	client, err := github.NewCacheRetentionClient(repo.Host)
	if err != nil {
		return false, err
	}
	return client.Update(repo, value)
}

func readCacheStorage(repo repository.Repository) (settings.CacheStorageObservation, error) {
	client, err := github.NewCacheStorageClient(repo.Host)
	if err != nil {
		return settings.CacheStorageObservation{}, err
	}
	return client.Read(repo)
}
func writeCacheStorage(repo repository.Repository, value int) (bool, error) {
	client, err := github.NewCacheStorageClient(repo.Host)
	if err != nil {
		return false, err
	}
	return client.Update(repo, value)
}

func readActionsAccess(repo repository.Repository) (settings.ActionsAccessObservation, error) {
	client, err := github.NewActionsAccessClient(repo.Host)
	if err != nil {
		return settings.ActionsAccessObservation{}, err
	}
	return client.Read(repo)
}
func writeActionsAccess(repo repository.Repository, value string) (bool, error) {
	client, err := github.NewActionsAccessClient(repo.Host)
	if err != nil {
		return false, err
	}
	return client.Update(repo, value)
}
