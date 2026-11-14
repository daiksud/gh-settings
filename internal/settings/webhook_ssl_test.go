package settings

import (
	"encoding/json"
	"testing"
)

func TestWebhookSSLIntentComparisonAndExport(t *testing.T) {
	for _, verify := range []bool{false, true} {
		o := webhookCreateInventory()
		o.Values[0].Config.InsecureSSL = verify
		d := Webhook{ID: 7, SSLVerification: &verify}
		selected, changes, err := PrepareWebhooks(o, []Webhook{d})
		if err != nil || len(changes) != 1 || *changes[0].Before.SSLVerification == verify || !selected[0].Before.Config.Overlay(d).Matches(d) {
			t.Fatal(changes, err)
		}
		next, err := o.WithUpdated(d)
		if err != nil || next.Values[0].Config.InsecureSSL == verify || o.Values[0].Config.InsecureSSL != verify {
			t.Fatal(next, err)
		}
		_, changes, err = PrepareWebhooks(next, []Webhook{d})
		if err != nil || len(changes) != 0 {
			t.Fatal(changes, err)
		}
		values, err := next.Declaration()
		if err != nil || values[0].SSLVerification == nil || *values[0].SSLVerification != verify {
			t.Fatal(values, err)
		}
		data, err := json.Marshal(WebhookChange{Before: &Webhook{ID: 7, SSLVerification: ptrBool(!verify)}, After: &d})
		var change WebhookChange
		if err != nil || json.Unmarshal(data, &change) != nil || *change.After.SSLVerification != verify {
			t.Fatal(string(data), err)
		}
	}
	for _, raw := range []string{`{"after":{"id":7,"ssl_verification":null}}`, `{"after":{"id":7,"ssl_verification":"false"}}`, `{"after":{"id":7,"ssl_verification":false,"ssl_verification":true}}`} {
		var value WebhookChange
		if json.Unmarshal([]byte(raw), &value) == nil {
			t.Fatal("invalid report", raw)
		}
	}
}
func ptrBool(v bool) *bool { return &v }

func TestWebhookCreationSSLDefaultAndSelection(t *testing.T) {
	d := webhookCreateIntent()
	s := WebhookState{ID: 8, CreatedAt: "stamp", Config: WebhookConfig{PayloadURL: *d.PayloadURL, ContentType: *d.ContentType}, Active: *d.Active, Events: *d.Events}
	if !WebhookCreationMatches(s, d) {
		t.Fatal("omitted SSL must require verified default")
	}
	s.Config.InsecureSSL = true
	if WebhookCreationMatches(s, d) {
		t.Fatal("default accepted disabled SSL")
	}
	d.SSLVerification = ptrBool(false)
	if !WebhookCreationMatches(s, d) {
		t.Fatal("explicit disabled SSL rejected")
	}
	d.SSLVerification = ptrBool(true)
	if WebhookCreationMatches(s, d) {
		t.Fatal("explicit enabled SSL lost")
	}
}
