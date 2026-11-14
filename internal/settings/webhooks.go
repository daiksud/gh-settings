package settings

import (
	"fmt"
	"net/url"
	"slices"
	"strings"
	"unicode"
)

const WebhooksPath = "repository.settings.webhooks"

// Webhook contains only selected public controls. Credentials never enter this model.
type Webhook struct {
	ID          int64   `json:"id"`
	PayloadURL  *string `json:"payload_url,omitempty"`
	ContentType *string `json:"content_type,omitempty"`
}

func ValidateWebhookURL(value string) error {
	invalid := func() error {
		return fmt.Errorf("webhook payload URL must be an absolute HTTP(S) URL without credentials, fragments or whitespace")
	}
	if value == "" || strings.ContainsFunc(value, func(r rune) bool { return unicode.IsSpace(r) || unicode.IsControl(r) }) || strings.Contains(value, "#") {
		return invalid()
	}
	if _, err := url.PathUnescape(value); err != nil {
		return invalid()
	}
	u, err := url.Parse(value)
	if err != nil || u == nil || (!strings.EqualFold(u.Scheme, "http") && !strings.EqualFold(u.Scheme, "https")) || u.Hostname() == "" || u.User != nil || u.Opaque != "" || u.Fragment != "" {
		return invalid()
	}
	return nil
}
func (v Webhook) Validate() error {
	if v.ID <= 0 || v.PayloadURL == nil && v.ContentType == nil {
		return fmt.Errorf("webhook requires a positive id and at least one selected control")
	}
	if v.PayloadURL != nil {
		if err := ValidateWebhookURL(*v.PayloadURL); err != nil {
			return err
		}
	}
	if v.ContentType != nil && *v.ContentType != "application/json" && *v.ContentType != "application/x-www-form-urlencoded" {
		return fmt.Errorf("webhook content type must be application/json or application/x-www-form-urlencoded")
	}
	return nil
}
func (v Webhook) Clone() Webhook {
	if v.PayloadURL != nil {
		x := *v.PayloadURL
		v.PayloadURL = &x
	}
	if v.ContentType != nil {
		x := *v.ContentType
		v.ContentType = &x
	}
	return v
}
func ValidateWebhooks(values []Webhook) error {
	seen := map[int64]bool{}
	for _, v := range values {
		if err := v.Validate(); err != nil {
			return err
		}
		if seen[v.ID] {
			return fmt.Errorf("duplicate webhook id")
		}
		seen[v.ID] = true
	}
	return nil
}

type WebhookTarget struct {
	ID, OwnerID int64
	FullName    string
	Archived    bool
}

func (t WebhookTarget) Validate() error {
	if t.ID <= 0 || t.OwnerID <= 0 || t.FullName == "" {
		return fmt.Errorf("incomplete webhook repository identity")
	}
	return nil
}

// WebhookConfig stores all readable nonsecret controls; secret bytes are discarded at decoding.
type WebhookConfig struct {
	PayloadURL, ContentType string
	InsecureSSL             bool
}

func (c WebhookConfig) Validate() error {
	return (Webhook{ID: 1, PayloadURL: &c.PayloadURL, ContentType: &c.ContentType}).Validate()
}
func (c WebhookConfig) Selected(d Webhook) Webhook {
	v := Webhook{ID: d.ID}
	if d.PayloadURL != nil {
		x := c.PayloadURL
		v.PayloadURL = &x
	}
	if d.ContentType != nil {
		x := c.ContentType
		v.ContentType = &x
	}
	return v
}
func (c WebhookConfig) Overlay(d Webhook) WebhookConfig {
	if d.PayloadURL != nil {
		c.PayloadURL = *d.PayloadURL
	}
	if d.ContentType != nil {
		c.ContentType = *d.ContentType
	}
	return c
}
func (c WebhookConfig) Matches(d Webhook) bool { return d.Validate() == nil && c.Overlay(d) == c }

type WebhookState struct {
	ID        int64
	CreatedAt string
	Config    WebhookConfig
	Active    bool
	Events    []string
}

func (s WebhookState) Clone() WebhookState { s.Events = slices.Clone(s.Events); return s }
func (s WebhookState) Validate() error {
	if s.ID <= 0 || s.CreatedAt == "" || s.Events == nil {
		return fmt.Errorf("incomplete webhook observation")
	}
	if err := s.Config.Validate(); err != nil {
		return err
	}
	seen := map[string]bool{}
	for _, event := range s.Events {
		if event == "" || seen[event] {
			return fmt.Errorf("invalid webhook event set")
		}
		seen[event] = true
	}
	return nil
}

type WebhookObservation struct {
	Target WebhookTarget
	Values []WebhookState
}

func (o WebhookObservation) Validate() error {
	if err := o.Target.Validate(); err != nil {
		return err
	}
	if o.Values == nil {
		return fmt.Errorf("webhook observation unavailable")
	}
	seen := map[int64]bool{}
	for _, v := range o.Values {
		if err := v.Validate(); err != nil {
			return err
		}
		if seen[v.ID] {
			return fmt.Errorf("duplicate webhook observation identity")
		}
		seen[v.ID] = true
	}
	return nil
}
func (o WebhookObservation) Declaration() ([]Webhook, error) {
	if err := o.Validate(); err != nil {
		return nil, err
	}
	values := make([]Webhook, 0, len(o.Values))
	for _, s := range o.Values {
		u, c := s.Config.PayloadURL, s.Config.ContentType
		values = append(values, Webhook{ID: s.ID, PayloadURL: &u, ContentType: &c})
	}
	slices.SortFunc(values, func(a, b Webhook) int {
		if a.ID < b.ID {
			return -1
		}
		if a.ID > b.ID {
			return 1
		}
		return 0
	})
	return values, nil
}

type WebhookSelection struct {
	Target  WebhookTarget
	Before  WebhookState
	Desired Webhook
}

func PrepareWebhooks(o WebhookObservation, desired []Webhook) ([]WebhookSelection, []WebhookChange, error) {
	if err := ValidateWebhooks(desired); err != nil {
		return nil, nil, err
	}
	if err := o.Validate(); err != nil {
		return nil, nil, err
	}
	states := map[int64]WebhookState{}
	for _, s := range o.Values {
		states[s.ID] = s
	}
	selections := make([]WebhookSelection, 0, len(desired))
	changes := []WebhookChange{}
	for _, d := range desired {
		s, ok := states[d.ID]
		if !ok {
			return nil, nil, fmt.Errorf("selected webhook id is missing")
		}
		selections = append(selections, WebhookSelection{o.Target, s.Clone(), d.Clone()})
		if !s.Config.Matches(d) {
			changes = append(changes, WebhookChange{Before: s.Config.Selected(d), After: d.Clone()})
		}
	}
	return selections, changes, nil
}
func WebhookIdentityMatches(a, b WebhookState) bool {
	return a.ID == b.ID && a.CreatedAt == b.CreatedAt
}
func WebhookPreserved(before, after WebhookState, desired Webhook) bool {
	a, b := slices.Clone(before.Events), slices.Clone(after.Events)
	slices.Sort(a)
	slices.Sort(b)
	return WebhookIdentityMatches(before, after) && after.Config == before.Config.Overlay(desired) && before.Active == after.Active && slices.Equal(a, b)
}

type WebhookChange struct {
	Before Webhook `json:"before"`
	After  Webhook `json:"after"`
}

func ValidateWebhookChanges(changes []WebhookChange) error {
	seen := map[int64]bool{}
	for _, c := range changes {
		if c.Before.Validate() != nil || c.After.Validate() != nil || c.Before.ID != c.After.ID || (c.Before.PayloadURL == nil) != (c.After.PayloadURL == nil) || (c.Before.ContentType == nil) != (c.After.ContentType == nil) || seen[c.After.ID] {
			return fmt.Errorf("invalid webhook comparison")
		}
		seen[c.After.ID] = true
		same := true
		if c.After.PayloadURL != nil && *c.Before.PayloadURL != *c.After.PayloadURL {
			same = false
		}
		if c.After.ContentType != nil && *c.Before.ContentType != *c.After.ContentType {
			same = false
		}
		if same {
			return fmt.Errorf("webhook comparison has no change")
		}
	}
	return nil
}
