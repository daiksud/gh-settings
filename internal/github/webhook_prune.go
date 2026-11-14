package github

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

// The destructive path is deliberately stricter than ordinary config selection.
// Known delivery/transport metadata is not configuration and is never retained.
func decodeWebhookPrune(repo repository.Repository, raw map[string]json.RawMessage) (settings.WebhookPruneState, error) {
	for key := range raw {
		switch key {
		case "id", "type", "name", "url", "created_at", "updated_at", "active", "events", "config":
		case "test_url", "ping_url", "deliveries_url", "last_response":
		default:
			return settings.WebhookPruneState{}, fmt.Errorf("webhook parent contains unsupported destructive evidence")
		}
	}
	state, err := decodeWebhook(repo, raw)
	if err != nil {
		return settings.WebhookPruneState{}, err
	}
	var updated string
	if !webhookRequired(raw, "updated_at", &updated) {
		return settings.WebhookPruneState{}, fmt.Errorf("webhook prune timestamp unavailable")
	}
	if _, err := time.Parse(time.RFC3339, updated); err != nil {
		return settings.WebhookPruneState{}, fmt.Errorf("webhook prune timestamp invalid")
	}
	return settings.WebhookPruneState{State: state, UpdatedAt: updated}, nil
}
func (c *Client) readWebhookPruneState(repo repository.Repository, id int64) (settings.WebhookPruneState, error) {
	var raw map[string]json.RawMessage
	link, err := c.webhookRequest(http.MethodGet, webhookPath(repo, id), nil, &raw)
	if err != nil {
		return settings.WebhookPruneState{}, err
	}
	state, err := decodeWebhookPrune(repo, raw)
	if err != nil {
		return settings.WebhookPruneState{}, err
	}
	if link != "" || state.State.ID != id {
		return settings.WebhookPruneState{}, fmt.Errorf("webhook prune detail identity incomplete")
	}
	var rawConfig map[string]json.RawMessage
	link, err = c.webhookRequest(http.MethodGet, webhookPath(repo, id)+"/config", nil, &rawConfig)
	if err != nil {
		return settings.WebhookPruneState{}, err
	}
	config, err := decodeWebhookConfig(rawConfig)
	if err != nil {
		return settings.WebhookPruneState{}, err
	}
	if link != "" || config != state.State.Config {
		return settings.WebhookPruneState{}, fmt.Errorf("webhook prune parent and config disagree")
	}
	return state, nil
}
func (c *Client) listWebhookPrune(repo repository.Repository) ([]settings.WebhookPruneState, error) {
	inventory := []settings.WebhookPruneState{}
	seen := map[int64]bool{}
	for page := 1; page <= 10000; page++ {
		var rows []map[string]json.RawMessage
		link, err := c.webhookRequest(http.MethodGet, fmt.Sprintf("%s?per_page=100&page=%d", webhookPath(repo, 0), page), nil, &rows)
		if err != nil {
			return nil, err
		}
		next, err := webhookHasNext(link)
		if err != nil || rows == nil || len(rows) > 100 || len(rows) < 100 && next {
			return nil, fmt.Errorf("webhook prune inventory incomplete")
		}
		for _, row := range rows {
			state, err := decodeWebhookPrune(repo, row)
			if err != nil {
				return nil, err
			}
			if seen[state.State.ID] {
				return nil, fmt.Errorf("webhook prune inventory identity repeated")
			}
			seen[state.State.ID] = true
			inventory = append(inventory, state)
		}
		if len(rows) < 100 {
			return inventory, nil
		}
	}
	return nil, fmt.Errorf("webhook prune inventory too large")
}

func (c *Client) ReadWebhookPrune(repo repository.Repository) (settings.WebhookPruneObservation, error) {
	fail := func(err error) (settings.WebhookPruneObservation, error) {
		return settings.WebhookPruneObservation{}, err
	}
	target, err := c.webhookTarget(repo)
	if err != nil {
		return fail(err)
	}
	inventory, err := c.listWebhookPrune(repo)
	if err != nil {
		return fail(err)
	}
	listed, err := settings.NewWebhookPruneObservation(target, inventory)
	if err != nil {
		return fail(err)
	}
	details := make([]settings.WebhookPruneState, 0, len(inventory))
	for _, state := range inventory {
		fresh, err := c.readWebhookPruneState(repo, state.State.ID)
		if err != nil {
			return fail(err)
		}
		details = append(details, fresh)
	}
	endInventory, err := c.listWebhookPrune(repo)
	if err != nil {
		return fail(err)
	}
	endList, err := settings.NewWebhookPruneObservation(target, endInventory)
	if err != nil || listed.Verify(endList) != nil {
		return fail(fmt.Errorf("webhook inventory changed during acquisition"))
	}
	after, err := c.webhookTarget(repo)
	if err != nil {
		return fail(err)
	}
	observed, err := settings.NewWebhookPruneObservation(after, details)
	if err != nil {
		return fail(err)
	}
	if err := listed.Verify(observed); err != nil {
		return fail(err)
	}
	return observed, nil
}

// A true result means DELETE was attempted, including uncertain failure. Complete
// absence and unchanged retained evidence are required before reporting success.
func (c *Client) DeleteWebhook(repo repository.Repository, expected settings.WebhookPruneObservation, id int64) (bool, error) {
	pre := func(err error) (bool, error) { return false, &securityPreWriteFailure{err} }
	if err := expected.VerifyDelete(id, expected); err != nil {
		return pre(err)
	}
	fresh, err := c.ReadWebhookPrune(repo)
	if err != nil {
		return pre(err)
	}
	if err := expected.VerifyDelete(id, fresh); err != nil {
		return pre(err)
	}
	candidate, err := c.readWebhookPruneState(repo, id)
	if err != nil {
		return pre(err)
	}
	target, err := c.webhookTarget(repo)
	if err != nil {
		return pre(err)
	}
	if err := expected.VerifyCandidate(id, candidate, target); err != nil {
		return pre(err)
	}
	response, err := c.rest.Request(http.MethodDelete, webhookPath(repo, id), nil)
	if err != nil {
		return true, fmt.Errorf("webhook deletion request unavailable; effects unconfirmed")
	}
	body, readErr := io.ReadAll(io.LimitReader(response.Body, 1))
	response.Body.Close()
	if readErr != nil || response.StatusCode != http.StatusNoContent || len(body) != 0 {
		return true, fmt.Errorf("webhook deletion acknowledgment unavailable; effects unconfirmed")
	}
	remaining, err := c.ReadWebhookPrune(repo)
	if err != nil {
		return true, fmt.Errorf("webhook deletion verification unavailable; effects unconfirmed")
	}
	want, err := expected.Without(id)
	if err != nil || want.Verify(remaining) != nil {
		return true, fmt.Errorf("webhook deletion absence or retained state unconfirmed")
	}
	return true, nil
}
