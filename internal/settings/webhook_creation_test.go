package settings

import (
	"encoding/json"
	"strings"
	"testing"
)

func webhookCreateIntent() Webhook {
	active := false
	events := []string{"push", "pull_request"}
	return Webhook{PayloadURL: webhookPtr("https://new.example/hook"), ContentType: webhookPtr("application/json"), Active: &active, Events: &events}
}
func webhookCreateInventory() WebhookObservation {
	return WebhookObservation{Complete: true, Target: WebhookTarget{ID: 1, OwnerID: 2, FullName: "owner/current"}, Values: []WebhookState{{ID: 7, CreatedAt: "stamp", Config: WebhookConfig{PayloadURL: "https://old.example/hook", ContentType: "application/json", InsecureSSL: true}, Active: false, Events: []string{"push"}}}}
}
func TestWebhookCreationGuardsAndCollisionPreflight(t *testing.T) {
	for _, mode := range []string{"create", "empty-events", "opaque-events", "existing-url", "inactive-url", "unlisted-url", "disjoint-events", "moving-away", "partial-inventory", "duplicate-create", "bound-declared-url", "parent-active", "parent-events", "matching-guards", "reordered-guards"} {
		t.Run(mode, func(t *testing.T) {
			o := webhookCreateInventory()
			d := webhookCreateIntent()
			bound := Webhook{ID: 7, ContentType: webhookPtr("application/x-www-form-urlencoded")}
			desired := []Webhook{bound, d}
			good := mode == "create" || mode == "empty-events" || mode == "opaque-events" || mode == "matching-guards" || mode == "reordered-guards"
			switch mode {
			case "empty-events":
				e := []string{}
				desired[1].Events = &e
			case "opaque-events":
				e := []string{"*", "future_event"}
				desired[1].Events = &e
			case "existing-url", "inactive-url":
				o.Values[0].Config.PayloadURL = *d.PayloadURL
			case "unlisted-url":
				o.Values[0].Config.PayloadURL = *d.PayloadURL
				desired = []Webhook{d}
			case "disjoint-events":
				o.Values[0].Config.PayloadURL = *d.PayloadURL
				o.Values[0].Events = []string{"issues"}
			case "moving-away":
				o.Values[0].Config.PayloadURL = *d.PayloadURL
				desired[0].PayloadURL = webhookPtr("https://elsewhere.example")
			case "partial-inventory":
				o.Complete = false
			case "duplicate-create":
				desired = append(desired, d)
			case "bound-declared-url":
				desired[0].PayloadURL = d.PayloadURL
			case "parent-active":
				a := true
				desired[0].Active = &a
			case "parent-events":
				e := []string{}
				desired[0].Events = &e
			case "matching-guards":
				a := false
				e := []string{"push"}
				desired[0].Active = &a
				desired[0].Events = &e
			case "reordered-guards":
				o.Values[0].Events = []string{"push", "issues"}
				e := []string{"issues", "push"}
				desired[0].Events = &e
			}
			selections, changes, err := PrepareWebhooks(o, desired)
			if (err == nil) != good {
				t.Fatal(err)
			}
			if !good {
				if selections != nil || changes != nil {
					t.Fatal("partial preparation escaped")
				}
				return
			}
			if len(changes) != 2 || changes[1].Before != nil || changes[1].After.ID != 0 || ValidateWebhookChanges(changes) != nil {
				t.Fatal(changes)
			}
			data, err := json.Marshal(changes)
			if err != nil {
				t.Fatal(err)
			}
			var read []WebhookChange
			if json.Unmarshal(data, &read) != nil {
				t.Fatal(string(data))
			}
			if strings.Contains(string(data), "insecure_ssl") || strings.Contains(string(data), "created_at") {
				t.Fatal("private snapshot escaped")
			}
		})
	}
}
func TestWebhookInventoryTracksOnlyVerifiedOwnChanges(t *testing.T) {
	initial := webhookCreateInventory()
	d := Webhook{ID: 7, ContentType: webhookPtr("application/x-www-form-urlencoded")}
	updated, err := initial.WithUpdated(d)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Values[0].Config.ContentType != *d.ContentType || !updated.Values[0].Config.InsecureSSL || initial.Values[0].Config.ContentType != "application/json" {
		t.Fatal("overlay/alias")
	}
	create := webhookCreateIntent()
	state := WebhookState{ID: 8, CreatedAt: "new", Config: WebhookConfig{PayloadURL: *create.PayloadURL, ContentType: *create.ContentType}, Events: *create.Events}
	next, err := updated.WithCreated(state)
	if err != nil || len(next.Values) != 2 || !WebhookCreationMatches(state, create) {
		t.Fatal(err)
	}
	if _, err = next.WithCreated(state); err == nil {
		t.Fatal("existing identity adopted")
	}
	for _, mutate := range []func(*WebhookObservation){func(o *WebhookObservation) { o.Complete = false }, func(o *WebhookObservation) { o.Target.OwnerID++ }, func(o *WebhookObservation) { o.Values = o.Values[1:] }, func(o *WebhookObservation) { o.Values[0].Config.InsecureSSL = false }, func(o *WebhookObservation) { o.Values[0].Active = true }, func(o *WebhookObservation) { o.Values[0].CreatedAt = "other" }, func(o *WebhookObservation) { o.Values[0].Events = []string{} }, func(o *WebhookObservation) { o.Values[0].Config.PayloadURL = "https://outside.example" }} {
		bad := next.Clone()
		mutate(&bad)
		if VerifyWebhookInventory(next, bad) == nil {
			t.Fatal("outside drift adopted")
		}
	}
	reordered := next.Clone()
	reordered.Values[0], reordered.Values[1] = reordered.Values[1], reordered.Values[0]
	reordered.Values[0].Events = []string{"pull_request", "push"}
	if VerifyWebhookInventory(next, reordered) != nil {
		t.Fatal("set order is drift")
	}
}
func TestWebhookCreationReportRejectsInvalidUnion(t *testing.T) {
	good := `{"after":{"payload_url":"https://example.com","content_type":"application/json","active":false,"events":[]}}`
	var c WebhookChange
	if json.Unmarshal([]byte(good), &c) != nil {
		t.Fatal("create report rejected")
	}
	for _, bad := range []string{
		`{"before":{"id":7,"active":false}}`,
		strings.Replace(good, `"after":{`, `"after":{"id":0,`, 1),
		strings.Replace(good, `"active":false`, `"active":null`, 1),
		strings.Replace(good, `"events":[]`, `"events":["push","push"]`, 1),
		strings.Replace(good, `"events":[]`, `"events":["bad event"]`, 1),
		strings.Replace(good, `"events":[]`, `"events":[null]`, 1),
		strings.Replace(good, `"events":[]`, `"Events":[]`, 1),
		strings.Replace(good, `"events":[]`, `"events":[],"events":[]`, 1),
		`{"before":{"id":7,"active":false,"content_type":"application/json"},"after":{"id":7,"active":true,"content_type":"application/x-www-form-urlencoded"}}`,
	} {
		if json.Unmarshal([]byte(bad), &c) == nil {
			t.Fatal("invalid report accepted", bad)
		}
	}
}
