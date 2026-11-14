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
	if c.After == nil {
		d := c.Before
		return fmt.Sprintf("%s (id %d): delete payload_url: %q, content_type: %q, ssl_verification: %t, active: %t, events: %q (irreversible whole-webhook deletion removes its opaque Secret)", settings.WebhooksPath, d.ID, *d.PayloadURL, *d.ContentType, *d.SSLVerification, *d.Active, *d.Events)
	}
	if c.Before == nil {
		d := c.After
		ssl := d.SSLVerification == nil || *d.SSLVerification
		warning := ""
		if !ssl {
			warning = "; disabled SSL verification permits man-in-the-middle attacks"
		}
		return fmt.Sprintf("%s: create payload_url: %q, content_type: %q, ssl_verification: %t, active: %t, events: %q (GitHub sends an automatic ping; no Secret supplied%s)", settings.WebhooksPath, *d.PayloadURL, *d.ContentType, ssl, *d.Active, *d.Events, warning)
	}
	fields := []string{}
	if c.After.PayloadURL != nil {
		fields = append(fields, fmt.Sprintf("payload_url: %q → %q", *c.Before.PayloadURL, *c.After.PayloadURL))
	}
	if c.After.ContentType != nil {
		fields = append(fields, fmt.Sprintf("content_type: %q → %q", *c.Before.ContentType, *c.After.ContentType))
	}
	if c.After.SSLVerification != nil {
		warning := ""
		if !*c.After.SSLVerification {
			warning = " (disabled SSL verification permits man-in-the-middle attacks)"
		}
		fields = append(fields, fmt.Sprintf("ssl_verification: %t → %t%s", *c.Before.SSLVerification, *c.After.SSLVerification, warning))
	}
	if c.After.Active != nil {
		fields = append(fields, fmt.Sprintf("active guard: %t", *c.After.Active))
	}
	if c.After.Events != nil {
		fields = append(fields, fmt.Sprintf("events guard: %q", *c.After.Events))
	}
	return fmt.Sprintf("%s (id %d): %s", settings.WebhooksPath, c.After.ID, strings.Join(fields, ", "))
}
