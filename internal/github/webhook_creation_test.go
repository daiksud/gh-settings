package github

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/daiksud/gh-settings/internal/settings"
)

func webhookCreationSelection() (settings.WebhookObservation, settings.Webhook) {
	url, mime, active, events := "https://new.example/hook", "application/json", false, []string{"pull_request", "push"}
	return settings.WebhookObservation{Complete: true, Target: settings.WebhookTarget{ID: 1, OwnerID: 2, FullName: "owner/current"}, Values: []settings.WebhookState{{ID: 7, CreatedAt: "2026-11-14T00:00:00Z", Config: settings.WebhookConfig{PayloadURL: "https://old.example/private-omitted", ContentType: "application/json", InsecureSSL: true}, Events: []string{"future_event", "push"}}}}, settings.Webhook{PayloadURL: &url, ContentType: &mime, Active: &active, Events: &events}
}
func TestWebhookCreationExactPayloadRecoveryAndReadback(t *testing.T) {
	modes := []string{"success", "ssl-enabled", "ssl-disabled", "empty-events", "opaque-events", "active", "form", "secret-changing", "location-ignored", "event-order", "pre-denied", "pre-target", "pre-addition", "pre-collision", "archived", "transport", "http", "wrong-status", "malformed", "duplicate-id", "duplicate-config", "null-id", "fraction-id", "old-id", "missing-config", "unknown-config", "ssl-ack", "active-ack", "events-ack", "unexpected-link", "detail-denied", "config-denied", "created-drift", "target-drift", "retained-drift", "outside-addition", "missing-new"}
	for _, mode := range modes {
		t.Run(mode, func(t *testing.T) {
			expected, desired := webhookCreationSelection()
			if mode == "ssl-enabled" || mode == "ssl-disabled" {
				v := mode == "ssl-enabled"
				desired.SSLVerification = &v
			}
			if mode == "empty-events" {
				v := []string{}
				desired.Events = &v
			}
			if mode == "opaque-events" {
				v := []string{"*", "future_event"}
				desired.Events = &v
			}
			if mode == "active" {
				v := true
				desired.Active = &v
			}
			if mode == "form" {
				v := "application/x-www-form-urlencoded"
				desired.ContentType = &v
			}
			if mode == "pre-collision" {
				v := "https://old.example/private-omitted"
				desired.PayloadURL = &v
			}
			if mode == "archived" {
				expected.Target.Archived = true
			}
			posts, lists, newDetails, newConfigs := 0, 0, 0, 0
			oldCfg := webhookCfg()
			var added map[string]any
			c := topicsClient(t, func(req *http.Request) (*http.Response, error) {
				if req.URL.Host != "api.github.com" {
					t.Fatal("destination request")
				}
				reply := func(status int, v any) (*http.Response, error) {
					return collaboratorsHTTPResponse(req, status, webhookBody(v)), nil
				}
				if req.Method == "POST" {
					if req.URL.Path != "/repos/owner/current/hooks" {
						t.Fatal("forbidden POST", req.URL)
					}
					posts++
					data, _ := io.ReadAll(req.Body)
					var payload map[string]any
					json.Unmarshal(data, &payload)
					events := make([]any, len(*desired.Events))
					for i, e := range *desired.Events {
						events[i] = e
					}
					native := "json"
					if mode == "form" {
						native = "form"
					}
					ssl := "0"
					if desired.SSLVerification != nil && !*desired.SSLVerification {
						ssl = "1"
					}
					want := map[string]any{"name": "web", "active": *desired.Active, "events": events, "config": map[string]any{"url": *desired.PayloadURL, "content_type": native, "insecure_ssl": ssl}}
					if !reflect.DeepEqual(payload, want) {
						t.Fatalf("creation payload %s", data)
					}
					if mode == "transport" {
						return nil, errors.New("PRIVATE_CREATION_TRANSPORT")
					}
					if mode == "http" {
						return reply(422, map[string]any{"id": 99, "message": "PRIVATE_HTTP"})
					}
					cfg := map[string]any{"url": *desired.PayloadURL, "content_type": native, "insecure_ssl": ssl, "secret": "PRIVATE_ACK_SECRET"}
					added = webhookRow(8, cfg)
					added["active"] = *desired.Active
					added["events"] = *desired.Events
					ack := webhookRaw(added)
					switch mode {
					case "wrong-status":
						return reply(200, added)
					case "malformed":
						return collaboratorsHTTPResponse(req, 201, `{"id":8,"PRIVATE_BROKEN`), nil
					case "duplicate-id":
						body := strings.Replace(webhookBody(added), `"id":8`, `"id":8,"id":9`, 1)
						return collaboratorsHTTPResponse(req, 201, body), nil
					case "duplicate-config":
						body := strings.Replace(webhookBody(added), `"content_type":"json"`, `"content_type":"json","content_type":"form"`, 1)
						return collaboratorsHTTPResponse(req, 201, body), nil
					case "null-id":
						ack["id"] = json.RawMessage("null")
					case "fraction-id":
						ack["id"] = json.RawMessage("8.1")
					case "old-id":
						ack["id"] = json.RawMessage("7")
					case "missing-config":
						delete(ack, "config")
					case "unknown-config":
						bad := map[string]any{}
						for k, v := range cfg {
							bad[k] = v
						}
						bad["PRIVATE_UNKNOWN"] = true
						ack["config"], _ = json.Marshal(bad)
					case "ssl-ack":
						cfg["insecure_ssl"] = "1"
						ack["config"], _ = json.Marshal(cfg)
					case "active-ack":
						ack["active"] = json.RawMessage("true")
					case "events-ack":
						ack["events"] = json.RawMessage(`[]`)
					}
					r := collaboratorsHTTPResponse(req, 201, webhookBody(ack))
					if mode == "unexpected-link" {
						r.Header.Set("Link", `<https://destination.invalid>; rel="next"`)
					}
					if mode == "location-ignored" {
						r.Header.Set("Location", "https://destination.invalid/PRIVATE")
					}
					return r, nil
				}
				if req.Method != "GET" {
					t.Fatal("forbidden mutation", req.Method, req.URL)
				}
				switch req.URL.Path {
				case "/repos/owner/current":
					target := webhookTargetJSON
					if mode == "archived" {
						target = strings.Replace(target, `"archived":false`, `"archived":true`, 1)
					}
					if mode == "pre-target" || mode == "target-drift" && posts > 0 {
						target = strings.Replace(target, `"id":1`, `"id":9`, 1)
					}
					return collaboratorsHTTPResponse(req, 200, target), nil
				case "/repos/owner/current/hooks":
					lists++
					if mode == "pre-denied" {
						return reply(403, map[string]any{"message": "PRIVATE_DENIED"})
					}
					if mode == "retained-drift" && posts > 0 {
						oldCfg["url"] = "https://outside.example/PRIVATE"
					}
					values := []any{webhookRow(7, oldCfg)}
					if added != nil && mode != "missing-new" {
						values = append(values, added)
					}
					if mode == "pre-addition" || mode == "outside-addition" && posts > 0 {
						values = append(values, webhookRow(9, webhookCfg()))
					}
					return reply(200, values)
				case "/repos/owner/current/hooks/7":
					return reply(200, webhookRow(7, oldCfg))
				case "/repos/owner/current/hooks/7/config":
					return reply(200, oldCfg)
				case "/repos/owner/current/hooks/9":
					return reply(200, webhookRow(9, webhookCfg()))
				case "/repos/owner/current/hooks/9/config":
					return reply(200, webhookCfg())
				case "/repos/owner/current/hooks/8":
					newDetails++
					if mode == "detail-denied" {
						return reply(403, map[string]any{"message": "PRIVATE_READBACK"})
					}
					if mode == "created-drift" {
						added["created_at"] = "2026-11-14T01:00:00Z"
					}
					if mode == "event-order" {
						added["events"] = []string{"push", "pull_request"}
					}
					if mode == "secret-changing" {
						added["config"].(map[string]any)["secret"] = "PRIVATE_DIFFERENT_SECRET"
					}
					return reply(200, added)
				case "/repos/owner/current/hooks/8/config":
					newConfigs++
					if mode == "config-denied" {
						return reply(403, map[string]any{"message": "PRIVATE_CONFIG"})
					}
					return reply(200, added["config"])
				default:
					t.Fatal("unexpected or destination route", req.URL)
				}
				return nil, fmt.Errorf("unexpected fixture")
			})
			result, err := c.CreateWebhook(webhookRepo, expected, desired)
			good := mode == "ssl-enabled" || mode == "ssl-disabled" || mode == "success" || mode == "empty-events" || mode == "opaque-events" || mode == "active" || mode == "form" || mode == "secret-changing" || mode == "location-ignored" || mode == "event-order"
			pre := strings.HasPrefix(mode, "pre-") || mode == "archived"
			if (err == nil) != good || posts > 1 || result.Attempted == pre {
				t.Fatal(err, result, posts)
			}
			if pre && posts != 0 {
				t.Fatal("preflight wrote")
			}
			var marker *securityPreWriteFailure
			if err != nil && (errors.As(err, &marker) != pre || strings.Contains(err.Error(), "PRIVATE")) {
				t.Fatal("classification/privacy", err)
			}
			recoverable := !pre && mode != "transport" && mode != "http" && mode != "wrong-status" && mode != "malformed" && mode != "duplicate-id" && mode != "null-id" && mode != "fraction-id" && mode != "old-id"
			if (result.RecoveryID == 8) != recoverable {
				t.Fatal("unsafe/missing recovery", mode, result.RecoveryID)
			}
			if good {
				if result.Observation.ValidateComplete() != nil || lists < 2 || newDetails < 2 || newConfigs < 2 {
					t.Fatal("ACK used as verification", result, lists, newDetails, newConfigs)
				}
			} else if result.Observation.Values != nil {
				t.Fatal("failed readback claimed verified observation")
			}
		})
	}
}

func TestWebhookExportAcquisitionRequiresAllPublicControls(t *testing.T) {
	observed, _ := webhookCreationSelection()
	full, err := observed.Declaration()
	if err != nil {
		t.Fatal(err)
	}
	if err = CheckExportAcquisition(settings.Declaration{Webhooks: &full}); err != nil {
		t.Fatal(err)
	}
	for _, drop := range []string{"id", "active", "events"} {
		partial := []settings.Webhook{full[0].Clone()}
		switch drop {
		case "id":
			partial[0].ID = 0
		case "active":
			partial[0].Active = nil
		case "events":
			partial[0].Events = nil
		}
		if CheckExportAcquisition(settings.Declaration{Webhooks: &partial}) == nil {
			t.Fatal("incomplete observation adopted", drop)
		}
		if _, err := UIFromExportDeclaration(settings.Declaration{Webhooks: &partial}); err != nil {
			t.Fatal("valid retained intent rejected", drop, err)
		}
	}
}
