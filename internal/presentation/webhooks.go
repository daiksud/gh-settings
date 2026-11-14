package presentation

import (
	"fmt"
	"github.com/daiksud/gh-settings/internal/settings"
	"strings"
)

func WebhookLine(c settings.WebhookChange) string {
	if settings.ValidateWebhookChanges([]settings.WebhookChange{c}) != nil {
		return settings.WebhooksPath + ": invalid comparison"
	}
	if c.Before == nil {
		d := c.After
		return fmt.Sprintf("%s: create payload_url: %q, content_type: %q, active: %t, events: %q (GitHub sends an automatic ping; no Secret supplied)", settings.WebhooksPath, *d.PayloadURL, *d.ContentType, *d.Active, *d.Events)
	}
	fields := []string{}
	if c.After.PayloadURL != nil {
		fields = append(fields, fmt.Sprintf("payload_url: %q → %q", *c.Before.PayloadURL, *c.After.PayloadURL))
	}
	if c.After.ContentType != nil {
		fields = append(fields, fmt.Sprintf("content_type: %q → %q", *c.Before.ContentType, *c.After.ContentType))
	}
	if c.After.Active != nil {
		fields = append(fields, fmt.Sprintf("active guard: %t", *c.After.Active))
	}
	if c.After.Events != nil {
		fields = append(fields, fmt.Sprintf("events guard: %q", *c.After.Events))
	}
	return fmt.Sprintf("%s (id %d): %s", settings.WebhooksPath, c.After.ID, strings.Join(fields, ", "))
}
