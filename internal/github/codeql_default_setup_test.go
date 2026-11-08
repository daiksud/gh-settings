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

func TestCodeQLDefaultSetupReadRequiresCompletePersonalPublicState(t *testing.T) {
	for _, meta := range []string{`{"visibility":"public","owner":{"type":"User"}}`, `{"visibility":"private","owner":{"type":"User"}}`, `{"visibility":"public","owner":{"type":"Organization"}}`, `{}`, `{"visibility":null,"owner":{"type":"User"}}`} {
		for _, body := range []string{`{"state":"configured"}`, `{"state":"not-configured"}`, `{}`, `null`, `{"state":null}`, `{"state":true}`, `{"state":"pending"}`, `{"State":"configured"}`} {
			client := &CodeQLDefaultSetupClient{rest: actionsClient(t, func(r *http.Request) (*http.Response, error) {
				if r.Method != "GET" {
					t.Fatal("read mutated")
				}
				if r.URL.Path == "/repos/owner/current" {
					return actionsResponse(r, 200, meta), nil
				}
				if r.URL.Path != "/repos/owner/current/code-scanning/default-setup" {
					t.Fatal(r.URL)
				}
				return actionsResponse(r, 200, body), nil
			}).rest}
			o, err := client.Read(repository.Repository{Owner: "owner", Name: "current"})
			valid := strings.Contains(meta, `"public"`) && strings.Contains(meta, `"User"`) && (body == `{"state":"configured"}` || body == `{"state":"not-configured"}`)
			if (err == nil) != valid {
				t.Fatal(meta, body, err)
			}
			value, valueErr := o.Value()
			if valid && (valueErr != nil || value != (body == `{"state":"configured"}`)) || !valid && valueErr == nil {
				t.Fatal(o, err, valueErr)
			}
		}
	}
}

func TestCodeQLDefaultSetupWritesOnlyStateAndConfirmsEnablement(t *testing.T) {
	for _, after := range []bool{true, false} {
		for _, mode := range []string{"ok", "accepted", "matching", "denied", "read-denied", "malformed", "pending", "verify-failed", "redirect"} {
			writes := 0
			enabled := !after
			if mode == "matching" {
				enabled = after
			}
			c := &CodeQLDefaultSetupClient{rest: actionsClient(t, func(r *http.Request) (*http.Response, error) {
				if r.URL.Path == "/repos/owner/current" {
					return actionsResponse(r, 200, `{"visibility":"public","owner":{"type":"User"}}`), nil
				}
				if r.URL.Path != "/repos/owner/current/code-scanning/default-setup" {
					t.Fatal(r.URL)
				}
				if r.Method == "GET" {
					if mode == "read-denied" || writes > 0 && mode == "verify-failed" {
						return actionsResponse(r, 403, `{}`), nil
					}
					if mode == "malformed" {
						return actionsResponse(r, 200, `{"state":null}`), nil
					}
					state := "not-configured"
					if enabled {
						state = "configured"
					}
					return actionsResponse(r, 200, `{"state":"`+state+`","languages":["go"],"query_suite":"extended","threat_model":"remote","runner_type":"labeled","runner_label":"retained"}`), nil
				}
				writes++
				body, _ := io.ReadAll(r.Body)
				var got map[string]any
				json.Unmarshal(body, &got)
				want := "not-configured"
				if after {
					want = "configured"
				}
				if r.Method != "PATCH" || len(got) != 1 || got["state"] != want {
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
					return actionsResponse(r, 202, `{"run_id":42,"run_url":"https://api.github.com/repos/owner/current/actions/runs/42"}`), nil
				}
				return actionsResponse(r, 200, `{}`), nil
			}).rest}
			change, _ := settings.NewSecurityChange("security.codeql_default_setup", !after, after)
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
