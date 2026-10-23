package github

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
)

var forkingRepo = repository.Repository{Host: "github.com", Owner: "owner", Name: "current"}

func forkingMetadata(allowed bool) string {
	return fmt.Sprintf(`{"id":42,"full_name":"owner/current","private":true,"visibility":"private","fork":false,"archived":false,"allow_forking":%t,"owner":{"id":7,"login":"owner","type":"Organization"}}`, allowed)
}

const forkingOrg = `{"id":7,"login":"owner","members_can_fork_private_repositories":true}`

func TestForkingRequiresPositiveApplicabilityAndOrganizationPolicy(t *testing.T) {
	for _, tc := range []struct {
		name, repo, org string
		status          int
		ok              bool
	}{
		{"false is observed", forkingMetadata(false), forkingOrg, 200, true},
		{"true is observed", forkingMetadata(true), forkingOrg, 200, true},
		{"public", strings.Replace(forkingMetadata(false), `"private":true,"visibility":"private"`, `"private":false,"visibility":"public"`, 1), forkingOrg, 200, false},
		{"internal", strings.Replace(forkingMetadata(false), `"visibility":"private"`, `"visibility":"internal"`, 1), forkingOrg, 200, false},
		{"personal", strings.Replace(forkingMetadata(false), `"Organization"`, `"User"`, 1), forkingOrg, 200, false},
		{"fork", strings.Replace(forkingMetadata(false), `"fork":false`, `"fork":true`, 1), forkingOrg, 200, false},
		{"archived", strings.Replace(forkingMetadata(false), `"archived":false`, `"archived":true`, 1), forkingOrg, 200, false},
		{"missing allowed", strings.Replace(forkingMetadata(false), `"allow_forking":false,`, ``, 1), forkingOrg, 200, false},
		{"null allowed", strings.Replace(forkingMetadata(false), `"allow_forking":false`, `"allow_forking":null`, 1), forkingOrg, 200, false},
		{"wrong target", strings.Replace(forkingMetadata(false), `owner/current`, `owner/other`, 1), forkingOrg, 200, false},
		{"wrong owner", strings.Replace(forkingMetadata(false), `"login":"owner"`, `"login":"other"`, 1), forkingOrg, 200, false},
		{"null private", strings.Replace(forkingMetadata(false), `"private":true`, `"private":null`, 1), forkingOrg, 200, false},
		{"missing fork", strings.Replace(forkingMetadata(false), `"fork":false,`, ``, 1), forkingOrg, 200, false},
		{"missing archive", strings.Replace(forkingMetadata(false), `"archived":false,`, ``, 1), forkingOrg, 200, false},
		{"null id", strings.Replace(forkingMetadata(false), `"id":42`, `"id":null`, 1), forkingOrg, 200, false},
		{"fraction id", strings.Replace(forkingMetadata(false), `"id":42`, `"id":42.5`, 1), forkingOrg, 200, false},
		{"wrong organization", forkingMetadata(false), strings.Replace(forkingOrg, `"id":7`, `"id":8`, 1), 200, false},
		{"hidden organization policy", forkingMetadata(false), `{"id":7,"login":"owner"}`, 200, false},
		{"disabled organization policy", forkingMetadata(false), strings.Replace(forkingOrg, `true`, `false`, 1), 200, false},
		{"null organization policy", forkingMetadata(false), strings.Replace(forkingOrg, `true`, `null`, 1), 200, false},
		{"denied", forkingMetadata(false), forkingOrg, 403, false},
		{"not found", forkingMetadata(false), forkingOrg, 404, false},
		{"redirect", forkingMetadata(false), forkingOrg, 301, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rest := writeTestClient(t, func(r *http.Request) (*http.Response, error) {
				if r.Method != "GET" {
					t.Fatal("read mutated", r.Method)
				}
				body := tc.repo
				if r.URL.Path == "/orgs/owner" {
					body = tc.org
				} else if r.URL.Path != "/repos/owner/current" {
					t.Fatal(r.URL)
				}
				return archiveHTTP(r, tc.status, body), nil
			})
			c := ForkingClient{rest.rest}
			o, err := c.Read(forkingRepo)
			if (err == nil) != tc.ok {
				t.Fatal(o, err)
			}
			if tc.ok {
				v, err := o.Value()
				if err != nil || v != (tc.name == "true is observed") {
					t.Fatal(v, err)
				}
			}
			for _, desired := range []bool{true, false} {
				if tc.ok {
					continue
				}
				_, err = c.Update(forkingRepo, desired)
				var pre *forkingPreWriteFailure
				if !errors.As(err, &pre) {
					t.Fatal("unobserved/inapplicable state was not a pre-write failure", err)
				}
			}
		})
	}
}
func TestForkingUpdateFreshNarrowIdempotentAndVerified(t *testing.T) {
	for _, desired := range []bool{false, true} {
		t.Run(fmt.Sprint(desired), func(t *testing.T) {
			current := !desired
			writes, orgReads := 0, 0
			rest := writeTestClient(t, func(r *http.Request) (*http.Response, error) {
				if r.URL.Path == "/orgs/owner" {
					if r.Method != "GET" {
						t.Fatal("organization mutation")
					}
					orgReads++
					return archiveHTTP(r, 200, forkingOrg), nil
				}
				if r.Method == "PATCH" {
					writes++
					var payload map[string]any
					err := json.NewDecoder(r.Body).Decode(&payload)
					if err != nil || len(payload) != 1 || payload["allow_forking"] != desired {
						t.Fatal(payload, err)
					}
					current = desired
				} else if r.Method != "GET" {
					t.Fatal(r.Method)
				}
				return archiveHTTP(r, 200, forkingMetadata(current)), nil
			})
			c := ForkingClient{rest.rest}
			wrote, err := c.Update(forkingRepo, desired)
			if err != nil || !wrote || writes != 1 || orgReads != 3 {
				t.Fatal(wrote, err, writes, orgReads)
			}
			wrote, err = c.Update(forkingRepo, desired)
			if err != nil || wrote || writes != 1 || orgReads != 4 {
				t.Fatal(wrote, err, writes, orgReads)
			}
		})
	}
}
func TestForkingStopsOnConcurrentOrUncertainState(t *testing.T) {
	for _, mode := range []string{"identity during read", "policy before write", "state before write", "patch denied", "patch wrong identity", "patch ignored", "readback wrong state", "readback policy denied"} {
		t.Run(mode, func(t *testing.T) {
			reads, orgReads, writes := 0, 0, 0
			rest := writeTestClient(t, func(r *http.Request) (*http.Response, error) {
				status := 200
				body := forkingMetadata(writes > 0)
				if r.URL.Path == "/orgs/owner" {
					orgReads++
					body = forkingOrg
					if (mode == "policy before write" && orgReads == 2) || (mode == "readback policy denied" && writes > 0) {
						status = 403
					}
				} else if r.Method == "PATCH" {
					writes++
					body = forkingMetadata(true)
					switch mode {
					case "patch denied":
						status = 403
					case "patch wrong identity":
						body = strings.Replace(body, `"id":42`, `"id":43`, 1)
					case "patch ignored":
						body = forkingMetadata(false)
					}
				} else {
					reads++
					if mode == "identity during read" && reads == 2 {
						body = strings.Replace(body, `"id":42`, `"id":43`, 1)
					}
					if mode == "state before write" && reads >= 3 {
						body = forkingMetadata(true)
					}
					if mode == "readback wrong state" && writes > 0 {
						body = forkingMetadata(false)
					}
				}
				return archiveHTTP(r, status, body), nil
			})
			c := ForkingClient{rest.rest}
			wrote, err := c.Update(forkingRepo, true)
			if err == nil || wrote {
				t.Fatal("unconfirmed success", wrote, err)
			}
			prewrite := strings.Contains(mode, "before write") || mode == "identity during read"
			var pre *forkingPreWriteFailure
			if errors.As(err, &pre) != prewrite || (prewrite && writes != 0) || (!prewrite && writes != 1) {
				t.Fatal(err, writes)
			}
			if !prewrite && !strings.Contains(err.Error(), "may have applied") {
				t.Fatal(err)
			}
		})
	}
}
func TestForkingReadOrganizationDeniedDoesNotBecomeFalse(t *testing.T) {
	rest := writeTestClient(t, func(r *http.Request) (*http.Response, error) {
		if r.URL.Path == "/orgs/owner" {
			return &http.Response{StatusCode: 404, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(`{"message":"not found"}`)), Request: r}, nil
		}
		return archiveHTTP(r, 200, forkingMetadata(false)), nil
	})
	c := ForkingClient{rest.rest}
	if _, err := c.Read(forkingRepo); err == nil {
		t.Fatal("ambiguous organization 404 became false")
	}
}
