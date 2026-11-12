package github

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/daiksud/gh-settings/internal/settings"
)

func TestDeleteActionsPolicyIdentityStateAcknowledgmentAndAbsence(t *testing.T) {
	state, err := eventPolicyDecode(t, eventPolicyJSON)
	if err != nil {
		t.Fatal(err)
	}
	change, err := settings.NewActionsEventPolicyDelete(*state.Policy)
	if err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"ok", "changed", "unsupported", "wrong-owner", "wrong-id", "denied", "status", "retained", "readback-failed", "partial-list"} {
		t.Run(mode, func(t *testing.T) {
			deletes, reads := 0, 0
			client := topicsClient(t, transportFunc(func(r *http.Request) (*http.Response, error) {
				if r.Method == "DELETE" {
					deletes++
					status := 204
					if mode == "status" {
						status = 200
					}
					return labelsResponse(r, status, ""), nil
				}
				if r.Method != "GET" {
					t.Fatal(r.Method)
				}
				if strings.HasSuffix(r.URL.Path, "/policies") {
					if mode == "readback-failed" {
						return labelsResponse(r, 503, `{}`), nil
					}
					if mode == "retained" {
						return labelsResponse(r, 200, `{"total_count":1,"policies":[`+eventPolicyJSON+`]}`), nil
					}
					if mode == "partial-list" {
						return labelsResponse(r, 200, `{"total_count":1,"policies":[]}`), nil
					}
					return labelsResponse(r, 200, `{"total_count":0,"policies":[]}`), nil
				}
				reads++
				body := eventPolicyJSON
				switch mode {
				case "changed":
					body = strings.Replace(body, `"enforcement":"active"`, `"enforcement":"disabled"`, 1)
				case "unsupported":
					body = strings.Replace(body, "restrict_action_events", "future_protection", 1)
				case "wrong-owner":
					body = strings.Replace(body, "owner/current", "other/current", 1)
				case "wrong-id":
					body = strings.Replace(body, `"id":7`, `"id":8`, 1)
				case "denied":
					return labelsResponse(r, 403, `{}`), nil
				}
				return labelsResponse(r, 200, body), nil
			}))
			id, err := client.WriteActionsEventPolicy(eventPolicyRepo, change)
			if mode == "ok" {
				if err != nil || id != 7 || deletes != 1 {
					t.Fatal(id, err, deletes)
				}
				return
			}
			if err == nil {
				t.Fatal("unexpected success", mode)
			}
			pre := mode == "changed" || mode == "unsupported" || mode == "wrong-owner" || mode == "wrong-id" || mode == "denied"
			if pre && deletes != 0 {
				t.Fatal("unsafe delete", mode)
			}
			if !pre && deletes != 1 {
				t.Fatal(fmt.Sprint(mode, reads, deletes, err))
			}
		})
	}
}
