package settings

import (
	"encoding/json"
	"testing"
)

func webhookPtr(v string) *string { return &v }
func TestWebhookURLAndPartialValidation(t *testing.T) {
	for _, url := range []string{"https://EXAMPLE.com:8443/a%2fb?q=%2F", "http://localhost/private", "https://[::1]:8443/", "https://example.com"} {
		if err := ValidateWebhookURL(url); err != nil {
			t.Fatal(url, err)
		}
	}
	for _, url := range []string{"", "/relative", "https:///missing", "ftp://example.com", "https://user:pass@example.com", "https://example.com/#", "https://example.com/?bad=%GG", "https://example.com/x y", "https://example.com/\nsecret", "https://example.com:wrong"} {
		if ValidateWebhookURL(url) == nil {
			t.Fatalf("accepted invalid URL %q", url)
		}
	}
	for _, v := range []Webhook{{ID: 1, PayloadURL: webhookPtr("https://example.com")}, {ID: 1, ContentType: webhookPtr("application/json")}, {ID: 1, ContentType: webhookPtr("application/x-www-form-urlencoded")}} {
		if v.Validate() != nil {
			t.Fatal(v)
		}
	}
	for _, v := range []Webhook{{ID: 0, ContentType: webhookPtr("application/json")}, {ID: 1}, {ID: 1, ContentType: webhookPtr("json")}, {ID: 1, PayloadURL: webhookPtr("")}} {
		if v.Validate() == nil {
			t.Fatal("accepted invalid declaration")
		}
	}
	d := Webhook{ID: 7, ContentType: webhookPtr("application/json")}
	if ValidateWebhooks([]Webhook{d, d}) == nil {
		t.Fatal("duplicate accepted")
	}
	clone := d.Clone()
	*clone.ContentType = "changed"
	if *d.ContentType != "application/json" {
		t.Fatal("shared pointer")
	}
}
func TestWebhookOwnershipAndPreservation(t *testing.T) {
	s := WebhookState{ID: 7, CreatedAt: "stamp", Config: WebhookConfig{PayloadURL: "https://private.example/omitted", ContentType: "application/json", InsecureSSL: true}, Events: []string{"future-event", "push"}, Active: false}
	d := Webhook{ID: 7, ContentType: webhookPtr("application/x-www-form-urlencoded")}
	o := WebhookObservation{Complete: true, Target: WebhookTarget{ID: 1, OwnerID: 2, FullName: "o/r"}, Values: []WebhookState{s}}
	selected, changes, err := PrepareWebhooks(o, []Webhook{d})
	if err != nil || len(selected) != 1 || len(changes) != 1 || changes[0].Before.PayloadURL != nil {
		t.Fatal("partial intent leaked", err)
	}
	*selected[0].Desired.ContentType = "other"
	selected[0].Before.Events[0] = "other"
	if *d.ContentType != "application/x-www-form-urlencoded" || s.Events[0] != "future-event" {
		t.Fatal("snapshot aliased")
	}
	after := s.Clone()
	after.Config = after.Config.Overlay(d)
	after.Events = []string{"push", "future-event"}
	if !WebhookPreserved(s, after, d) {
		t.Fatal("set ordering became drift")
	}
	for _, mutate := range []func(*WebhookState){func(s *WebhookState) { s.Config.PayloadURL = "https://drift.example" }, func(s *WebhookState) { s.Config.InsecureSSL = false }, func(s *WebhookState) { s.CreatedAt = "new" }, func(s *WebhookState) { s.Active = true }, func(s *WebhookState) { s.Events = []string{"push"} }} {
		bad := after.Clone()
		mutate(&bad)
		if WebhookPreserved(s, bad, d) {
			t.Fatal("omitted control drift ignored")
		}
	}
	exported, err := o.Declaration()
	if err != nil || exported[0].PayloadURL == nil || exported[0].ContentType == nil {
		t.Fatal(err)
	}
	s.Config.ContentType = *d.ContentType
	o.Values = []WebhookState{s}
	_, changes, err = PrepareWebhooks(o, []Webhook{d})
	if err != nil || len(changes) != 0 {
		t.Fatal("matching state changed", err)
	}
}
func TestWebhookReportClosedProjections(t *testing.T) {
	good := `{"before":{"id":9007199254740993,"content_type":"application/json"},"after":{"id":9007199254740993,"content_type":"application/x-www-form-urlencoded"}}`
	var c WebhookChange
	if err := json.Unmarshal([]byte(good), &c); err != nil || c.After.ID != 9007199254740993 {
		t.Fatal(err)
	}
	for _, bad := range []string{`{"before":null,"after":{}}`, `{"Before":{},"after":{}}`, `{"before":{},"after":{},"before":{}}`, `{"before":{"id":7,"content_type":null},"after":{"id":7,"content_type":"application/json"}}`, `{"before":{"id":7,"Content_type":"application/json"},"after":{"id":7,"content_type":"application/json"}}`, `{"before":{"id":7,"id":7,"content_type":"application/json"},"after":{"id":7,"content_type":"application/x-www-form-urlencoded"}}`, `{"before":{"id":7,"content_type":"application/json","secret":"PRIVATE"},"after":{"id":7,"content_type":"application/x-www-form-urlencoded"}}`} {
		if json.Unmarshal([]byte(bad), &c) == nil {
			t.Fatal("invalid report accepted", bad)
		}
	}
}
