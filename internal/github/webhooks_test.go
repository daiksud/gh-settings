package github

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

var webhookRepo = repository.Repository{Host: "github.com", Owner: "owner", Name: "current"}

const webhookTargetJSON = `{"id":1,"full_name":"owner/current","private":false,"visibility":"public","archived":false,"owner":{"id":2,"login":"owner","type":"User"}}`

func webhookCfg() map[string]any {
	return map[string]any{"url": "https://old.example/private-omitted", "content_type": "json", "insecure_ssl": "1", "secret": "PRIVATE_SECRET_CANARY"}
}
func webhookRow(id int64, cfg map[string]any) map[string]any {
	return map[string]any{"id": id, "name": "web", "type": "Repository", "url": fmt.Sprintf("https://api.github.com/repos/owner/current/hooks/%d", id), "created_at": "2026-11-14T00:00:00Z", "config": cfg, "active": false, "events": []string{"future_event", "push"}, "last_response": map[string]any{"message": "PRIVATE_DELIVERY_CANARY"}, "test_url": "https://destination.invalid/forbidden"}
}
func webhookRaw(v any) map[string]json.RawMessage {
	b, _ := json.Marshal(v)
	var r map[string]json.RawMessage
	json.Unmarshal(b, &r)
	return r
}
func webhookBody(v any) string { b, _ := json.Marshal(v); return string(b) }
func webhookSelected(t *testing.T) settings.WebhookSelection {
	t.Helper()
	s, e := decodeWebhook(webhookRepo, webhookRaw(webhookRow(7, webhookCfg())))
	if e != nil {
		t.Fatal(e)
	}
	want := "application/x-www-form-urlencoded"
	return settings.WebhookSelection{Target: settings.WebhookTarget{ID: 1, OwnerID: 2, FullName: "owner/current"}, Before: s, Desired: settings.Webhook{ID: 7, ContentType: &want}}
}
func TestWebhookPartialPatchAndVerification(t *testing.T) {
	modes := []string{"content", "url", "both", "noop", "archived-noop", "archived-change", "pre-read-error", "HTTP-error", "transport-error", "bad-json", "empty-ack", "null-ack", "missing-ack", "unknown-ack", "ack-drift", "readback-denied", "selected-matching-drift", "omitted-url-drift", "ssl-drift", "active-drift", "events-drift", "created-drift", "target-drift", "pre-target-drift", "secret-missing", "secret-changing", "event-order", "fresh-companions", "fresh-match", "fresh-created-drift"}
	for _, mode := range modes {
		t.Run(mode, func(t *testing.T) {
			selected := webhookSelected(t)
			cfg := webhookCfg()
			patches := 0
			targetReads := 0
			requests := []string{}
			currentParent := webhookRow(7, cfg)
			url := "https://NEW.example:8443/a%2Fb?case=%2f"
			switch mode {
			case "url":
				selected.Desired.ContentType = nil
				selected.Desired.PayloadURL = &url
			case "both":
				selected.Desired.PayloadURL = &url
			case "selected-matching-drift":
				v := cfg["url"].(string)
				selected.Desired.PayloadURL = &v
			case "noop", "archived-noop":
				v := "application/json"
				selected.Desired.ContentType = &v
			case "fresh-match":
				cfg["content_type"] = "form"
			case "fresh-companions":
				cfg["url"] = "https://fresh.example/omitted"
				cfg["insecure_ssl"] = 0
				currentParent["active"] = true
				currentParent["events"] = []string{"issues"}
			case "fresh-created-drift":
				currentParent["created_at"] = "2026-11-14T01:00:00Z"
			}
			if strings.HasPrefix(mode, "archived") {
				selected.Target.Archived = true
			}
			c := topicsClient(t, func(req *http.Request) (*http.Response, error) {
				requests = append(requests, req.Method+" "+req.URL.Path)
				if req.Method == "PATCH" {
					patches++
					if req.URL.Path != "/repos/owner/current/hooks/7/config" {
						t.Fatal("parent mutation")
					}
					data, _ := io.ReadAll(req.Body)
					var payload map[string]any
					json.Unmarshal(data, &payload)
					want := map[string]any{}
					if selected.Desired.PayloadURL != nil && *selected.Desired.PayloadURL != cfg["url"] {
						want["url"] = *selected.Desired.PayloadURL
					}
					if selected.Desired.ContentType != nil && cfg["content_type"] != "form" {
						want["content_type"] = "form"
					}
					if !reflect.DeepEqual(payload, want) || len(payload) == 0 || strings.Contains(string(data), "PRIVATE") {
						t.Fatalf("payload %s expected %v", data, want)
					}
					if mode == "transport-error" {
						return nil, errors.New("PRIVATE_TRANSPORT_CANARY")
					}
					if mode == "HTTP-error" {
						return collaboratorsHTTPResponse(req, 403, `{"message":"PRIVATE_ERROR_CANARY"}`), nil
					}
					for k, v := range payload {
						cfg[k] = v
					}
					ack := map[string]any{}
					for k, v := range cfg {
						ack[k] = v
					}
					switch mode {
					case "empty-ack":
						return collaboratorsHTTPResponse(req, 200, ""), nil
					case "bad-json":
						return collaboratorsHTTPResponse(req, 200, `{"PRIVATE_MALFORMED`), nil
					case "null-ack":
						return collaboratorsHTTPResponse(req, 200, "null"), nil
					case "missing-ack":
						delete(ack, "url")
					case "unknown-ack":
						ack["PRIVATE_UNKNOWN_CANARY"] = true
					case "ack-drift":
						ack["content_type"] = "json"
					case "secret-missing":
						delete(ack, "secret")
						delete(cfg, "secret")
					case "secret-changing":
						ack["secret"] = "MASK_CHANGED"
						cfg["secret"] = "DIFFERENT_PRIVATE_SECRET"
					}
					return collaboratorsHTTPResponse(req, 200, webhookBody(ack)), nil
				}
				if req.Method != "GET" {
					t.Fatal("forbidden operation", req.Method)
				}
				if req.URL.Path == "/repos/owner/current" {
					targetReads++
					target := webhookTargetJSON
					if strings.HasPrefix(mode, "archived") {
						target = strings.Replace(target, `"archived":false`, `"archived":true`, 1)
					}
					if mode == "target-drift" && patches > 0 || mode == "pre-target-drift" && targetReads == 2 {
						target = strings.Replace(target, `"id":1`, `"id":9`, 1)
					}
					return collaboratorsHTTPResponse(req, 200, target), nil
				}
				if mode == "pre-read-error" && patches == 0 || mode == "readback-denied" && patches > 0 {
					return collaboratorsHTTPResponse(req, 403, `{"message":"PRIVATE_UPSTREAM_CANARY"}`), nil
				}
				if patches > 0 {
					switch mode {
					case "selected-matching-drift", "omitted-url-drift":
						cfg["url"] = "https://drift.example"
					case "ssl-drift":
						cfg["insecure_ssl"] = "0"
					case "active-drift":
						currentParent["active"] = true
					case "events-drift":
						currentParent["events"] = []string{"issues"}
					case "created-drift":
						currentParent["created_at"] = "2026-11-14T02:00:00Z"
					case "event-order":
						currentParent["events"] = []string{"push", "future_event"}
					}
				}
				switch req.URL.Path {
				case "/repos/owner/current/hooks/7":
					return collaboratorsHTTPResponse(req, 200, webhookBody(currentParent)), nil
				case "/repos/owner/current/hooks/7/config":
					return collaboratorsHTTPResponse(req, 200, webhookBody(cfg)), nil
				default:
					t.Fatal("destination or unselected route", req.URL)
					return nil, nil
				}
			})
			wrote, err := c.ApplyWebhook(webhookRepo, selected)
			success := mode == "content" || mode == "url" || mode == "both" || mode == "noop" || mode == "archived-noop" || mode == "secret-missing" || mode == "secret-changing" || mode == "event-order" || mode == "fresh-companions" || mode == "fresh-match"
			if (err == nil) != success {
				t.Fatal(mode, wrote, err, requests)
			}
			if err != nil && strings.Contains(err.Error(), "PRIVATE") {
				t.Fatal("private diagnostic", err)
			}
			pre := mode == "archived-change" || mode == "pre-read-error" || mode == "pre-target-drift" || mode == "fresh-created-drift"
			if pre {
				var marker interface{ ReadFailedBeforeWrite() }
				if !errors.As(err, &marker) || patches != 0 || wrote {
					t.Fatal("prewrite failure classification", err, patches)
				}
			} else if mode == "noop" || mode == "archived-noop" || mode == "fresh-match" {
				if patches != 0 || wrote {
					t.Fatal("matching write")
				}
			} else {
				if patches != 1 || !wrote {
					t.Fatal("missing attempted evidence", patches, wrote)
				}
			}
		})
	}
}
func TestWebhookCompleteAcquisitionAndUnknownPrivacy(t *testing.T) {
	for _, mode := range []string{"selected", "export", "empty", "null-list", "duplicate-id", "foreign-url", "target-private", "target-org", "target-incomplete", "config-conflict", "unknown-config", "missing-url", "null-secret", "number-secret", "number-ssl", "bool-ssl", "missing-events", "duplicate-events", "duplicate-json", "renamed-created", "denied-config", "target-drift"} {
		t.Run(mode, func(t *testing.T) {
			cfg := webhookCfg()
			row := webhookRow(7, cfg)
			target := webhookTargetJSON
			reads := 0
			switch mode {
			case "foreign-url":
				row["url"] = "https://elsewhere.example/PRIVATE"
			case "target-private":
				target = strings.Replace(target, `"private":false`, `"private":true`, 1)
			case "target-org":
				target = strings.Replace(target, `"User"`, `"Organization"`, 1)
			case "target-incomplete":
				target = strings.Replace(target, `"visibility":"public",`, "", 1)
			case "unknown-config":
				cfg["PRIVATE_UNKNOWN_MEMBER"] = true
			case "missing-url":
				delete(cfg, "url")
			case "null-secret":
				cfg["secret"] = nil
			case "number-secret":
				cfg["secret"] = 42
			case "number-ssl":
				cfg["insecure_ssl"] = 1
			case "bool-ssl":
				cfg["insecure_ssl"] = true
			case "missing-events":
				delete(row, "events")
			case "duplicate-events":
				row["events"] = []string{"push", "push"}
			}
			c := topicsClient(t, func(req *http.Request) (*http.Response, error) {
				if req.Method != "GET" {
					t.Fatal("write during read")
				}
				switch req.URL.Path {
				case "/repos/owner/current":
					reads++
					if mode == "target-drift" && reads == 2 {
						target = strings.Replace(target, `"id":1`, `"id":9`, 1)
					}
					return collaboratorsHTTPResponse(req, 200, target), nil
				case "/repos/owner/current/hooks":
					if mode == "null-list" {
						return collaboratorsHTTPResponse(req, 200, "null"), nil
					}
					rows := []any{row}
					if mode == "empty" {
						rows = []any{}
					}
					if mode == "duplicate-id" {
						rows = append(rows, row)
					}
					return collaboratorsHTTPResponse(req, 200, webhookBody(rows)), nil
				case "/repos/owner/current/hooks/7":
					if mode == "renamed-created" {
						row["created_at"] = "2026-11-14T01:00:00Z"
					}
					return collaboratorsHTTPResponse(req, 200, webhookBody(row)), nil
				case "/repos/owner/current/hooks/7/config":
					if mode == "denied-config" {
						return collaboratorsHTTPResponse(req, 403, `{"message":"PRIVATE_CONFIG_ERROR"}`), nil
					}
					if mode == "config-conflict" {
						cfg["content_type"] = "form"
					}
					if mode == "duplicate-json" {
						return collaboratorsHTTPResponse(req, 200, `{"url":"https://x.example","url":"https://y.example","content_type":"json","insecure_ssl":"0"}`), nil
					}
					return collaboratorsHTTPResponse(req, 200, webhookBody(cfg)), nil
				default:
					t.Fatal("unexpected route", req.URL)
					return nil, nil
				}
			})
			var ids []int64
			if mode == "selected" {
				ids = []int64{7}
			}
			o, e := c.ReadWebhooks(webhookRepo, ids)
			good := mode == "selected" || mode == "export" || mode == "empty" || mode == "number-ssl"
			if (e == nil) != good {
				t.Fatal(mode, e)
			}
			if e != nil && strings.Contains(e.Error(), "PRIVATE") {
				t.Fatal(e)
			}
			if good {
				values, e := o.Declaration()
				if e != nil {
					t.Fatal(e)
				}
				b, _ := json.Marshal(values)
				if strings.Contains(string(b), "secret") || strings.Contains(string(b), "PRIVATE") {
					t.Fatal("private export", string(b))
				}
			}
		})
	}
}
func TestWebhookFixedRoutePagination(t *testing.T) {
	for _, mode := range []string{"complete", "later-denied", "repeat", "short-next"} {
		t.Run(mode, func(t *testing.T) {
			pages, details := 0, 0
			c := topicsClient(t, func(req *http.Request) (*http.Response, error) {
				if req.URL.Path == "/repos/owner/current" {
					return collaboratorsHTTPResponse(req, 200, webhookTargetJSON), nil
				}
				if req.URL.Path == "/repos/owner/current/hooks" {
					pages++
					if pages == 2 && mode == "later-denied" {
						return collaboratorsHTTPResponse(req, 403, `{"message":"PRIVATE"}`), nil
					}
					rows := []any{}
					if pages == 1 || mode == "repeat" {
						for i := int64(1); i <= 100; i++ {
							rows = append(rows, webhookRow(i, webhookCfg()))
						}
					}
					if mode == "short-next" {
						rows = rows[:1]
					}
					response := collaboratorsHTTPResponse(req, 200, webhookBody(rows))
					if pages == 1 {
						response.Header.Set("Link", `<https://destination.invalid/private>; rel="next"`)
					}
					return response, nil
				}
				details++
				if strings.HasSuffix(req.URL.Path, "/config") {
					return collaboratorsHTTPResponse(req, 200, webhookBody(webhookCfg())), nil
				}
				var id int64
				fmt.Sscanf(strings.TrimPrefix(req.URL.Path, "/repos/owner/current/hooks/"), "%d", &id)
				if id <= 0 {
					t.Fatal("followed link", req.URL)
				}
				return collaboratorsHTTPResponse(req, 200, webhookBody(webhookRow(id, webhookCfg()))), nil
			})
			o, e := c.ReadWebhooks(webhookRepo, nil)
			if mode == "complete" {
				if e != nil || len(o.Values) != 100 || pages != 2 || details != 200 {
					t.Fatal(e, pages, details)
				}
			} else if e == nil || details != 0 {
				t.Fatal("incomplete list accepted", e, details)
			}
		})
	}
}
func TestWebhookSSLNumericRepresentationsAndExportCompleteness(t *testing.T) {
	for _, raw := range []string{`0`, `0.0`, `0e20`, `0e999999999999999999999`, `-0`, `1`, `1.0`, `1e0`, `"0"`, `"1"`, `"\u0030"`, `"\u0031"`, ` 1 `} {
		cfg := webhookRaw(webhookCfg())
		cfg["insecure_ssl"] = json.RawMessage(raw)
		if _, e := decodeWebhookConfig(cfg); e != nil {
			t.Fatal(raw, e)
		}
	}
	for _, raw := range []string{`true`, `null`, `2`, `"1.0"`, `0.9999999999999999999999`, `1e-9999`, `1e99999999999999999`} {
		cfg := webhookRaw(webhookCfg())
		cfg["insecure_ssl"] = json.RawMessage(raw)
		if _, e := decodeWebhookConfig(cfg); e == nil {
			t.Fatal("unsupported SSL accepted", raw)
		}
	}
	content := "application/json"
	partial := []settings.Webhook{{ID: 7, ContentType: &content}}
	if e := CheckExportAcquisition(settings.Declaration{Webhooks: &partial}); e == nil {
		t.Fatal("incomplete acquired export accepted")
	}
	if _, e := UIFromExportDeclaration(settings.Declaration{Webhooks: &partial}); e != nil {
		t.Fatal("valid retained partial intent rejected", e)
	}
}

func TestWebhookExportBindsTargetBeforeInventory(t *testing.T) {
	for _, empty := range []bool{true, false} {
		t.Run(fmt.Sprint("empty=", empty), func(t *testing.T) {
			id := 1
			requests := []string{}
			c := topicsClient(t, func(req *http.Request) (*http.Response, error) {
				requests = append(requests, req.URL.Path)
				switch req.URL.Path {
				case "/repos/owner/current":
					return collaboratorsHTTPResponse(req, 200, strings.Replace(webhookTargetJSON, `"id":1`, fmt.Sprintf(`"id":%d`, id), 1)), nil
				case "/repos/owner/current/hooks":
					id = 9
					rows := []any{}
					if !empty {
						rows = append(rows, webhookRow(7, webhookCfg()))
					}
					return collaboratorsHTTPResponse(req, 200, webhookBody(rows)), nil
				case "/repos/owner/current/hooks/7":
					return collaboratorsHTTPResponse(req, 200, webhookBody(webhookRow(7, webhookCfg()))), nil
				case "/repos/owner/current/hooks/7/config":
					return collaboratorsHTTPResponse(req, 200, webhookBody(webhookCfg())), nil
				default:
					t.Fatal("unexpected route", req.URL)
					return nil, nil
				}
			})
			if _, err := c.ReadWebhooks(webhookRepo, nil); err == nil {
				t.Fatal("old inventory accepted under replacement identity", requests)
			}
			if requests[0] != "/repos/owner/current" {
				t.Fatal("inventory preceded target binding", requests)
			}
		})
	}
}

func TestWebhookShortPageChecksAllLinkRelations(t *testing.T) {
	cases := []struct {
		name    string
		headers []string
		valid   bool
	}{
		{"unquoted", []string{`<https://foreign.example/PRIVATE>; rel=next`}, false},
		{"quoted list", []string{`<https://foreign.example/PRIVATE>; rel="prev next"`}, false},
		{"parameter case", []string{`<https://foreign.example/PRIVATE>; REL = "Next"`}, false},
		{"separate headers", []string{`<https://foreign.example/PRIVATE>; rel=prev`, `<https://foreign.example/PRIVATE>; rel=next`}, false},
		{"comma URI and title", []string{`<https://foreign.example/PRIVATE,a>; title="x,y"; rel=next`}, false},
		{"malformed", []string{`<PRIVATE_UNCLOSED; rel=next`}, false},
		{"duplicate relation", []string{`<https://foreign.example/PRIVATE>; rel=prev; rel=next`}, false},
		{"terminal", []string{`<https://foreign.example/PRIVATE,a>; rel=prev; title="x,y", <https://foreign.example/PRIVATE>; rel="first last"`}, true},
		{"extended relation overrides next", []string{`<https://foreign.example/PRIVATE>; rel=next; rel*=utf-8''prev`}, false},
		{"extended relation before next", []string{`<https://foreign.example/PRIVATE>; rel*=utf-8''prev; rel=next`}, false},
		{"extended relation case", []string{`<https://foreign.example/PRIVATE>; rel=next; REL* = utf-8''prev`}, false},
		{"continued relation overrides next", []string{`<https://foreign.example/PRIVATE>; rel=next; rel*0*=utf-8''pre; rel*1*=v`}, false},
		{"extended relation alone", []string{`<https://foreign.example/PRIVATE>; rel*=utf-8''prev`}, false},
		{"quoted extension text", []string{`<https://foreign.example/PRIVATE>; rel=prev; title=";rel*=utf-8''next"`}, true},
		{"unrelated extension", []string{`<https://foreign.example/PRIVATE>; rel=prev; title*=utf-8''example`}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			pages, details := 0, 0
			c := topicsClient(t, func(req *http.Request) (*http.Response, error) {
				switch req.URL.Path {
				case "/repos/owner/current":
					return collaboratorsHTTPResponse(req, 200, webhookTargetJSON), nil
				case "/repos/owner/current/hooks":
					pages++
					if req.URL.Query().Get("page") != "1" {
						t.Fatal("retried unsupported short page")
					}
					r := collaboratorsHTTPResponse(req, 200, webhookBody([]any{webhookRow(7, webhookCfg())}))
					for _, h := range tc.headers {
						r.Header.Add("Link", h)
					}
					return r, nil
				case "/repos/owner/current/hooks/7":
					details++
					return collaboratorsHTTPResponse(req, 200, webhookBody(webhookRow(7, webhookCfg()))), nil
				case "/repos/owner/current/hooks/7/config":
					details++
					return collaboratorsHTTPResponse(req, 200, webhookBody(webhookCfg())), nil
				default:
					t.Fatal("followed response URL", req.URL)
					return nil, nil
				}
			})
			_, err := c.ReadWebhooks(webhookRepo, nil)
			if (err == nil) != tc.valid {
				t.Fatal("incorrect completeness decision", err)
			}
			if err != nil && (details != 0 || strings.Contains(err.Error(), "PRIVATE")) {
				t.Fatal("incomplete inventory used or private link leaked", details, err)
			}
			if pages != 1 {
				t.Fatal(pages)
			}
		})
	}
}
