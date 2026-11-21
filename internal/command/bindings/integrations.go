package bindings

import (
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/github"
	"github.com/daiksud/gh-settings/internal/settings"
)

func createWebhook(repo repository.Repository, expected settings.WebhookObservation, desired settings.Webhook) (settings.WebhookCreateResult, error) {
	client, err := github.NewWebhooksClient(repo.Host)
	if err != nil {
		return settings.WebhookCreateResult{}, err
	}
	return client.CreateWebhook(repo, expected, desired)
}

func deleteWebhook(repo repository.Repository, expected settings.WebhookPruneObservation, id int64) (bool, error) {
	client, err := github.NewWebhooksClient(repo.Host)
	if err != nil {
		return false, err
	}
	return client.DeleteWebhook(repo, expected, id)
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
