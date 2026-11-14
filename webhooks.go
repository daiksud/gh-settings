package main

import (
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/github"
	"github.com/daiksud/gh-settings/internal/settings"
)

func readWebhooks(repo repository.Repository, ids []int64) (settings.WebhookObservation, error) {
	client, err := github.NewWebhooksClient(repo.Host)
	if err != nil {
		return settings.WebhookObservation{}, err
	}
	return client.ReadWebhooks(repo, ids)
}
func writeWebhook(repo repository.Repository, selection settings.WebhookSelection) (bool, error) {
	client, err := github.NewWebhooksClient(repo.Host)
	if err != nil {
		return false, err
	}
	return client.ApplyWebhook(repo, selection)
}
