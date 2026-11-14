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
	fields := []string{}
	if c.After.PayloadURL != nil {
		fields = append(fields, fmt.Sprintf("payload_url: %q → %q", *c.Before.PayloadURL, *c.After.PayloadURL))
	}
	if c.After.ContentType != nil {
		fields = append(fields, fmt.Sprintf("content_type: %q → %q", *c.Before.ContentType, *c.After.ContentType))
	}
	return fmt.Sprintf("%s (id %d): %s", settings.WebhooksPath, c.After.ID, strings.Join(fields, ", "))
}
