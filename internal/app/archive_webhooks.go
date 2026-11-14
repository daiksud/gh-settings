package app

import (
	"fmt"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

// verifyArchivedWebhooks mirrors the no-op part of the existing webhook writer:
// a prepared selection must still match after a fresh identity and state read.
// Secret values are outside the declaration and observation models.
func verifyArchivedWebhooks(repo repository.Repository, desired *[]settings.Webhook, archive settings.ArchiveState, read func(repository.Repository, []int64) (settings.WebhookObservation, error)) error {
	if desired == nil || len(*desired) == 0 {
		return nil
	}
	ids := webhookIDs(*desired)
	if ids == nil || read == nil {
		return fmt.Errorf("verify archived webhooks: existing IDs and reader required")
	}
	observe := func() (settings.WebhookObservation, error) {
		o, err := read(repo, ids)
		if err != nil {
			return o, err
		}
		if o.Target.ID != archive.ID || o.Target.FullName != archive.FullName || !o.Target.Archived || len(o.Values) != len(ids) {
			return o, fmt.Errorf("webhook target or selected observation differs")
		}
		selected, changes, err := settings.PrepareWebhooks(o, *desired)
		if err != nil {
			return o, err
		}
		if len(selected) != len(ids) || len(changes) != 0 {
			return o, fmt.Errorf("webhook selection differs")
		}
		return o, nil
	}
	first, err := observe()
	if err != nil {
		return fmt.Errorf("verify archived webhooks: %w", err)
	}
	fresh, err := observe()
	if err != nil {
		return fmt.Errorf("verify archived webhooks: %w", err)
	}
	if first.Target != fresh.Target {
		return fmt.Errorf("verify archived webhooks: repository identity changed")
	}
	for _, id := range ids {
		before, ok := first.Find(id)
		after, found := fresh.Find(id)
		if !ok || !found || !settings.WebhookIdentityMatches(before, after) {
			return fmt.Errorf("verify archived webhooks: webhook identity changed")
		}
	}
	return nil
}
