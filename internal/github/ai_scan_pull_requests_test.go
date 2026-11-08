package github

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func TestAIScanPullRequestsReadRequiresCompletePersonalPublicState(t *testing.T) {
	for _, meta := range []string{`{"visibility":"public","owner":{"type":"User"}}`, `{"visibility":"private","owner":{"type":"User"}}`, `{"visibility":"public","owner":{"type":"Organization"}}`, `{}`, `{"visibility":null,"owner":{"type":"User"}}`} {
		for _, body := range []string{`{"pr_scan":"enabled"}`, `{"pr_scan":"disabled"}`, `{}`, `null`, `{"pr_scan":null}`, `{"pr_scan":true}`, `{"pr_scan":"pending"}`, `{"Pr_scan":"enabled"}`} {
			client := &AIScanPullRequestsClient{rest: actionsClient(t, func(r *http.Request) (*http.Response, error) {
				if r.Method != "GET" {
					t.Fatal("read mutated")
				}
				if r.URL.Path == "/repos/owner/current" {
					return actionsResponse(r, 200, meta), nil
				}
				if r.URL.Path != "/repos/owner/current/code-scanning/ai-scan" {
					t.Fatal(r.URL)
				}
				return actionsResponse(r, 200, body), nil
			}).rest}
			o, err := client.Read(repository.Repository{Owner: "owner", Name: "current"})
			valid := strings.Contains(meta, `"public"`) && strings.Contains(meta, `"User"`) && (body == `{"pr_scan":"enabled"}` || body == `{"pr_scan":"disabled"}`)
			if (err == nil) != valid {
				t.Fatal(meta, body, err)
			}
			value, valueErr := o.Value()
			if valid && (valueErr != nil || value != (body == `{"pr_scan":"enabled"}`)) || !valid && valueErr == nil {
				t.Fatal(o, err, valueErr)
			}
		}
	}
}

func TestAIScanPullRequestsWritesOnlyStateAndConfirmsEnablement(t *testing.T) {
	for _, after := range []bool{true, false} {
		for _, mode := range []string{"ok", "accepted", "matching", "denied", "read-denied", "malformed", "pending", "verify-failed", "redirect"} {
			writes := 0
			enabled := !after
			if mode == "matching" {
				enabled = after
			}
			c := &AIScanPullRequestsClient{rest: actionsClient(t, func(r *http.Request) (*http.Response, error) {
				if r.URL.Path == "/repos/owner/current" {
					return actionsResponse(r, 200, `{"visibility":"public","owner":{"type":"User"}}`), nil
				}
				if r.URL.Path != "/repos/owner/current/code-scanning/ai-scan" {
					t.Fatal(r.URL)
				}
				if r.Method == "GET" {
					if mode == "read-denied" || writes > 0 && mode == "verify-failed" {
						return actionsResponse(r, 403, `{}`), nil
					}
					if mode == "malformed" {
						return actionsResponse(r, 200, `{"pr_scan":null}`), nil
					}
					state := "disabled"
					if enabled {
						state = "enabled"
					}
					return actionsResponse(r, 200, `{"pr_scan":"`+state+`","languages":["go"],"query_suite":"extended","threat_model":"remote","runner_type":"labeled","runner_label":"retained"}`), nil
				}
				writes++
				body, _ := io.ReadAll(r.Body)
				var got map[string]any
				json.Unmarshal(body, &got)
				want := "disabled"
				if after {
					want = "enabled"
				}
				if r.Method != "PATCH" || len(got) != 1 || got["pr_scan"] != want {
					t.Fatal(r.Method, string(body))
				}
				if mode == "denied" {
					return actionsResponse(r, 422, `{}`), nil
				}
				if mode == "redirect" {
					resp := actionsResponse(r, 307, `{}`)
					resp.Header.Set("Location", "https://other.invalid/")
					return resp, nil
				}
				if mode != "pending" {
					enabled = after
				}
				if mode == "accepted" {
					return actionsResponse(r, 200, `{"run_id":42,"run_url":"https://api.github.com/repos/owner/current/actions/runs/42"}`), nil
				}
				return actionsResponse(r, 200, `{}`), nil
			}).rest}
			change, _ := settings.NewSecurityChange("security.ai_scan_pull_requests", !after, after)
			changed, err := c.Update(repository.Repository{Owner: "owner", Name: "current"}, change)
			success := mode == "ok" || mode == "accepted" || mode == "matching"
			if (err == nil) != success || changed != (success && mode != "matching") {
				t.Fatal(after, mode, changed, err)
			}
			var pre *securityPreWriteFailure
			if errors.As(err, &pre) != (mode == "read-denied" || mode == "malformed") {
				t.Fatal(mode, err)
			}
			if writes > 1 || (mode == "matching" || mode == "read-denied" || mode == "malformed") && writes != 0 {
				t.Fatal(mode, writes)
			}
		}
	}
}
