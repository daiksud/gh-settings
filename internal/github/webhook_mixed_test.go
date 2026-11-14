package github

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/daiksud/gh-settings/internal/settings"
)

func TestWebhookMixedUpdateGuardsWholeInventory(t *testing.T) {
	for _, mode := range []string{"verified", "pre-drift", "fresh-selected-drift", "post-drift", "parent-guard", "matching"} {
		t.Run(mode, func(t *testing.T) {
			selected := webhookSelected(t)
			cfg, other := webhookCfg(), webhookCfg()
			other["url"] = "https://retained.example/omitted"
			old, err := decodeWebhook(webhookRepo, webhookRaw(webhookRow(9, other)))
			if err != nil {
				t.Fatal(err)
			}
			observed := settings.WebhookObservation{Complete: true, Target: selected.Target, Values: []settings.WebhookState{selected.Before, old}}
			selected.Expected = &observed
			if mode == "matching" {
				v := "application/json"
				selected.Desired.ContentType = &v
			}
			if mode == "parent-guard" {
				v := true
				selected.Desired.Active = &v
			}
			patches, lists, details := 0, 0, 0
			c := topicsClient(t, func(r *http.Request) (*http.Response, error) {
				reply := func(v any) (*http.Response, error) { return collaboratorsHTTPResponse(r, 200, webhookBody(v)), nil }
				switch r.Method + " " + r.URL.Path {
				case "GET /repos/owner/current":
					return collaboratorsHTTPResponse(r, 200, webhookTargetJSON), nil
				case "GET /repos/owner/current/hooks":
					lists++
					if mode == "pre-drift" || mode == "post-drift" && patches > 0 {
						other["url"] = "https://outside.example/PRIVATE"
					}
					return reply([]any{webhookRow(7, cfg), webhookRow(9, other)})
				case "GET /repos/owner/current/hooks/7":
					details++
					if mode == "fresh-selected-drift" && details > 1 {
						cfg["url"] = "https://outside.example/PRIVATE"
					}
					return reply(webhookRow(7, cfg))
				case "GET /repos/owner/current/hooks/7/config":
					return reply(cfg)
				case "GET /repos/owner/current/hooks/9":
					return reply(webhookRow(9, other))
				case "GET /repos/owner/current/hooks/9/config":
					return reply(other)
				case "PATCH /repos/owner/current/hooks/7/config":
					patches++
					cfg["content_type"] = "form"
					return reply(cfg)
				default:
					t.Fatal("unexpected route", r.Method, r.URL)
				}
				return nil, fmt.Errorf("unexpected fixture")
			})
			changed, err := c.ApplyWebhook(webhookRepo, selected)
			good := mode == "verified" || mode == "matching"
			if (err == nil) != good || patches > 1 {
				t.Fatal(err, patches)
			}
			if good {
				if changed != (mode == "verified") || lists < 1 {
					t.Fatal(changed, lists)
				}
			} else {
				pre := mode != "post-drift"
				var marker *securityPreWriteFailure
				if errors.As(err, &marker) != pre || changed == pre || strings.Contains(err.Error(), "PRIVATE") {
					t.Fatal("classification/privacy", changed, err)
				}
				if pre && patches != 0 {
					t.Fatal("drift wrote")
				}
			}
		})
	}
}
