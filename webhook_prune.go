package main

import (
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/github"
	"github.com/daiksud/gh-settings/internal/settings"
)

func deleteWebhook(repo repository.Repository, expected settings.WebhookPruneObservation, id int64) (bool, error) {
	client, err := github.NewWebhooksClient(repo.Host)
	if err != nil {
		return false, err
	}
	return client.DeleteWebhook(repo, expected, id)
}
