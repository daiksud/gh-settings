package app

import (
	"errors"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func TestRunnerAndWebhookPruneGlobalPreflightAndPlan(t *testing.T) {
	runners := []settings.RunnerLabels{}
	hooks := []settings.Webhook{}
	desired := settings.Declaration{Runners: &runners, Webhooks: &hooks}
	load := func() (settings.Declaration, error) { return desired, nil }
	prune, err := settings.ParsePruneScopes("all")
	if err != nil {
		t.Fatal(err)
	}
	for _, fail := range []string{"runner", "webhook"} {
		t.Run(fail, func(t *testing.T) {
			writes := 0
			writer := Writer{
				ReadRunnerPrune: func(repository.Repository) (settings.RunnerPruneObservation, error) {
					if fail == "runner" {
						return settings.RunnerPruneObservation{}, errors.New("runner unavailable")
					}
					return pruneRunnerInventory(), nil
				},
				ReadWebhookPrune: func(repository.Repository) (settings.WebhookPruneObservation, error) {
					if fail == "webhook" {
						return settings.WebhookPruneObservation{}, errors.New("webhook unavailable")
					}
					return appWebhookPrune(t, false, 7), nil
				},
				RunnerLabels: func(repository.Repository, settings.RunnerLabels) (bool, error) { writes++; return true, nil },
				DeleteRunnerPrune: func(repository.Repository, settings.RunnerPruneObservation, []int64, int64) (bool, error) {
					writes++
					return true, nil
				},
				DeleteWebhook: func(repository.Repository, settings.WebhookPruneObservation, int64) (bool, error) {
					writes++
					return true, nil
				},
			}
			result, err := Apply(load, checkoutRepository, writer, prune)
			if err == nil || writes != 0 || len(result.Operations) != 0 {
				t.Fatal(result, err, writes)
			}
		})
	}
	result, err := Plan(load, checkoutRepository, Reader{
		RunnerPrune: func(repository.Repository) (settings.RunnerPruneObservation, error) {
			return pruneRunnerInventory(), nil
		},
		WebhookPrune: func(repository.Repository) (settings.WebhookPruneObservation, error) {
			return appWebhookPrune(t, false, 7), nil
		},
	}, prune)
	if err != nil || result.Comparison.Managed != 2 || len(result.Comparison.RunnerDeletions) != 3 || len(result.Comparison.Webhooks) != 1 || result.Comparison.Webhooks[0].After != nil {
		t.Fatal(result, err)
	}
}
