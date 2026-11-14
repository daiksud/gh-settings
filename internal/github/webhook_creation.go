package github

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func (c *Client) CreateWebhook(repo repository.Repository, expected settings.WebhookObservation, desired settings.Webhook) (settings.WebhookCreateResult, error) {
	result := settings.WebhookCreateResult{}
	pre := func(err error) (settings.WebhookCreateResult, error) { return result, &securityPreWriteFailure{err} }
	if desired.ID != 0 || desired.Validate() != nil || expected.ValidateComplete() != nil {
		return pre(fmt.Errorf("invalid webhook creation selection"))
	}
	current, err := c.ReadWebhooks(repo, nil)
	if err != nil {
		return pre(err)
	}
	if err = settings.VerifyWebhookInventory(expected, current); err != nil {
		return pre(err)
	}
	if current.Target.Archived {
		return pre(fmt.Errorf("archived repository cannot create webhooks"))
	}
	if _, _, err = settings.PrepareWebhooks(current, []settings.Webhook{desired}); err != nil {
		return pre(err)
	}
	native := "json"
	if *desired.ContentType == "application/x-www-form-urlencoded" {
		native = "form"
	}
	payload := struct {
		Name   string            `json:"name"`
		Config map[string]string `json:"config"`
		Active bool              `json:"active"`
		Events []string          `json:"events"`
	}{"web", map[string]string{"url": *desired.PayloadURL, "content_type": native, "insecure_ssl": "0"}, *desired.Active, *desired.Events}
	data, err := json.Marshal(payload)
	if err != nil {
		return pre(fmt.Errorf("invalid webhook creation request"))
	}
	result.Attempted = true
	response, err := c.rest.Request(http.MethodPost, webhookPath(repo, 0), bytes.NewReader(data))
	if err != nil {
		return result, fmt.Errorf("webhook creation request unavailable")
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusCreated {
		return result, fmt.Errorf("unexpected webhook creation status")
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, (8<<20)+1))
	if err != nil || len(body) > 8<<20 {
		return result, fmt.Errorf("incomplete webhook creation response")
	}
	// Recovery comes only from an unambiguous new identity in this successful body.
	// It is retained when the rest of the acknowledgment or readback fails.
	id, err := webhookCreationID(body)
	if err != nil {
		return result, err
	}
	if _, known := expected.Find(id); known {
		return result, fmt.Errorf("webhook creation returned an existing identity")
	}
	result.RecoveryID = id
	if response.Header.Get("Link") != "" {
		return result, fmt.Errorf("unexpected webhook creation pagination")
	}
	var raw map[string]json.RawMessage
	if webhookJSON(body, &raw) != nil {
		return result, fmt.Errorf("incomplete webhook creation response")
	}
	ack, err := decodeWebhook(repo, raw)
	if err != nil {
		return result, err
	}
	if !settings.WebhookCreationMatches(ack, desired) {
		return result, fmt.Errorf("webhook creation acknowledgment differs from requested configuration")
	}
	actual, err := c.readWebhook(repo, id)
	if err != nil {
		return result, err
	}
	if !settings.WebhookStateEqual(ack, actual) {
		return result, fmt.Errorf("webhook creation detail verification differs")
	}
	observed, err := c.ReadWebhooks(repo, nil)
	if err != nil {
		return result, err
	}
	want, err := expected.WithCreated(ack)
	if err != nil {
		return result, err
	}
	if err = settings.VerifyWebhookInventory(want, observed); err != nil {
		return result, err
	}
	result.Observation = observed
	return result, nil
}

// Recover identity separately from the complete acknowledgment. Other malformed
// values cannot confirm creation, but an unambiguous top-level ID remains useful
// for manual recovery after a successful response. Invalid JSON or repeated ID
// members never supply recovery evidence.
func webhookCreationID(body []byte) (int64, error) {
	fail := func() (int64, error) { return 0, fmt.Errorf("invalid webhook creation identity") }
	d := json.NewDecoder(bytes.NewReader(body))
	t, err := d.Token()
	if err != nil || t != json.Delim('{') {
		return fail()
	}
	var id int64
	seen := false
	for d.More() {
		t, err = d.Token()
		if err != nil {
			return fail()
		}
		key, ok := t.(string)
		if !ok {
			return fail()
		}
		var raw json.RawMessage
		if d.Decode(&raw) != nil {
			return fail()
		}
		if key == "id" {
			if seen || json.Unmarshal(raw, &id) != nil || id <= 0 {
				return fail()
			}
			seen = true
		}
	}
	if t, err = d.Token(); err != nil || t != json.Delim('}') {
		return fail()
	}
	if _, err = d.Token(); err != io.EOF || !seen {
		return fail()
	}
	return id, nil
}
