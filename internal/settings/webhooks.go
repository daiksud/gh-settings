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
	ID          int64     `json:"id,omitempty"`
	PayloadURL  *string   `json:"payload_url,omitempty"`
	ContentType *string   `json:"content_type,omitempty"`
	Active      *bool     `json:"active,omitempty"`
	Events      *[]string `json:"events,omitempty"`
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
	if v.ID < 0 || v.PayloadURL == nil && v.ContentType == nil && v.Active == nil && v.Events == nil {
		return fmt.Errorf("webhook requires selected controls and a positive id when present")
	}
	if v.ID == 0 && (v.PayloadURL == nil || v.ContentType == nil || v.Active == nil || v.Events == nil) {
		return fmt.Errorf("webhook creation requires payload URL, content type, active and events")
	}
	if v.Events != nil {
		if err := ValidateWebhookEvents(*v.Events); err != nil {
			return err
		}
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
	if v.Active != nil {
		x := *v.Active
		v.Active = &x
	}
	if v.Events != nil {
		x := slices.Clone(*v.Events)
		slices.Sort(x)
		v.Events = &x
	}
	return v
}
func ValidateWebhookEvents(events []string) error {
	if events == nil {
		return fmt.Errorf("webhook events must be an explicit array")
	}
	seen := map[string]bool{}
	for _, event := range events {
		if event == "" || seen[event] || strings.ContainsFunc(event, func(r rune) bool { return unicode.IsSpace(r) || unicode.IsControl(r) }) {
			return fmt.Errorf("invalid webhook event set")
		}
		seen[event] = true
	}
	return nil
}
func ValidateWebhooks(values []Webhook) error {
	seen := map[int64]bool{}
	creates := map[string]bool{}
	urls := map[string]int{}
	for _, v := range values {
		if err := v.Validate(); err != nil {
			return err
		}
		if v.ID > 0 {
			if seen[v.ID] {
				return fmt.Errorf("duplicate webhook id")
			}
			seen[v.ID] = true
		} else {
			creates[*v.PayloadURL] = true
		}
		if v.PayloadURL != nil {
			urls[*v.PayloadURL]++
		}
	}
	for url := range creates {
		if urls[url] > 1 {
			return fmt.Errorf("webhook creation URL collides with another declaration")
		}
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
	return ValidateWebhookEvents(s.Events)
}

type WebhookObservation struct {
	Complete bool
	Target   WebhookTarget
	Values   []WebhookState
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
	if err := o.ValidateComplete(); err != nil {
		return nil, err
	}
	values := make([]Webhook, 0, len(o.Values))
	for _, s := range o.Values {
		u, c := s.Config.PayloadURL, s.Config.ContentType
		a, e := s.Active, slices.Clone(s.Events)
		values = append(values, Webhook{ID: s.ID, PayloadURL: &u, ContentType: &c, Active: &a, Events: &e}.Clone())
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
	Target   WebhookTarget
	Before   WebhookState
	Desired  Webhook
	Expected *WebhookObservation // Present only for mixed creation/configuration work.
}

func PrepareWebhooks(o WebhookObservation, desired []Webhook) ([]WebhookSelection, []WebhookChange, error) {
	if err := ValidateWebhooks(desired); err != nil {
		return nil, nil, err
	}
	if err := o.Validate(); err != nil {
		return nil, nil, err
	}
	creates := false
	for _, d := range desired {
		creates = creates || d.ID == 0
	}
	if creates {
		if err := o.ValidateComplete(); err != nil {
			return nil, nil, err
		}
	}
	selections := make([]WebhookSelection, 0, len(desired))
	changes := []WebhookChange{}
	for _, d := range desired {
		d = d.Clone()
		if d.ID == 0 {
			for _, live := range o.Values {
				if live.Config.PayloadURL == *d.PayloadURL {
					return nil, nil, fmt.Errorf("webhook creation URL already exists; recover and declare its ID")
				}
			}
			selections = append(selections, WebhookSelection{Target: o.Target, Desired: d})
			changes = append(changes, WebhookChange{After: &d})
			continue
		}
		s, ok := o.Find(d.ID)
		if !ok {
			return nil, nil, fmt.Errorf("selected webhook id is missing")
		}
		if !s.GuardsMatch(d) {
			return nil, nil, fmt.Errorf("webhook active or events guard differs; parent changes are unsupported")
		}
		selections = append(selections, WebhookSelection{Target: o.Target, Before: s.Clone(), Desired: d})
		if !s.Config.Matches(d) {
			before := s.Selected(d)
			changes = append(changes, WebhookChange{Before: &before, After: &d})
		}
	}
	return selections, changes, nil
}
func (s WebhookState) GuardsMatch(d Webhook) bool {
	return (d.Active == nil || *d.Active == s.Active) && (d.Events == nil || WebhookEventsEqual(*d.Events, s.Events))
}
func (s WebhookState) Selected(d Webhook) Webhook {
	v := s.Config.Selected(d)
	if d.Active != nil {
		a := s.Active
		v.Active = &a
	}
	if d.Events != nil {
		e := slices.Clone(s.Events)
		v.Events = &e
	}
	return v.Clone()
}
func WebhookEventsEqual(a, b []string) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	a, b = slices.Clone(a), slices.Clone(b)
	slices.Sort(a)
	slices.Sort(b)
	return slices.Equal(a, b)
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
	Before *Webhook `json:"before,omitempty"`
	After  *Webhook `json:"after,omitempty"`
}

func ValidateWebhookChanges(changes []WebhookChange) error {
	desired := make([]Webhook, 0, len(changes))
	for _, c := range changes {
		if c.After == nil || c.After.Validate() != nil {
			return fmt.Errorf("invalid webhook comparison")
		}
		desired = append(desired, *c.After)
		if c.Before == nil {
			if c.After.ID != 0 {
				return fmt.Errorf("webhook create comparison must omit id")
			}
			continue
		}
		a, b := c.Before, c.After
		if a.Validate() != nil || a.ID <= 0 || a.ID != b.ID || (a.PayloadURL == nil) != (b.PayloadURL == nil) || (a.ContentType == nil) != (b.ContentType == nil) || (a.Active == nil) != (b.Active == nil) || (a.Events == nil) != (b.Events == nil) {
			return fmt.Errorf("invalid webhook comparison")
		}
		if a.Active != nil && *a.Active != *b.Active || a.Events != nil && !WebhookEventsEqual(*a.Events, *b.Events) {
			return fmt.Errorf("webhook parent changes are unsupported")
		}
		same := (a.PayloadURL == nil || *a.PayloadURL == *b.PayloadURL) && (a.ContentType == nil || *a.ContentType == *b.ContentType)
		if same {
			return fmt.Errorf("webhook comparison has no change")
		}
	}
	return ValidateWebhooks(desired)
}
