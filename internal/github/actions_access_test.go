package github

import (
	"errors"
	"fmt"
	"net/http"
	"testing"
)

func TestActionsAccessStrictApplicabilityAndRead(t *testing.T) {
	for _, metadata := range []string{`{}`, `null`, `{"full_name":"owner/current"}`, `{"full_name":"owner/current","visibility":null}`, `{"full_name":"owner/current","visibility":"public"}`, `{"full_name":"owner/current","visibility":"internal"}`, `{"full_name":"owner/current","visibility":"private"}`, `{"full_name":"owner/current","visibility":"private","owner":null}`, `{"full_name":"owner/current","visibility":"private","owner":{"type":"Bot"}}`, `{"full_name":"owner/current","visibility":"private","owner":{"Type":"Organization"}}`} {
		client := &ActionsAccessClient{rest: actionsClient(t, func(r *http.Request) (*http.Response, error) {
			if r.URL.Path != "/repos/owner/current" || r.Method != "GET" {
				t.Fatal("invalid applicability proceeded", r.Method, r.URL)
			}
			return actionsResponse(r, 200, metadata), nil
		}).rest}
		if _, err := client.Read(generalTestRepo); err == nil {
			t.Fatal("accepted", metadata)
		}
	}
	for _, body := range []string{`{}`, `null`, `{"access_level":null}`, `{"access_level":false}`, `{"access_level":"enterprise"}`, `{"access_level":"unknown"}`, `{"Access_level":"none"}`, `{"access_level":"user"}`} {
		client := &ActionsAccessClient{rest: actionsClient(t, func(r *http.Request) (*http.Response, error) {
			if r.Method != "GET" {
				t.Fatal("read mutated")
			}
			if r.URL.Path == "/repos/owner/current" {
				return actionsResponse(r, 200, generalTestMetadata), nil
			}
			if r.URL.Path != "/repos/owner/current/actions/permissions/access" {
				t.Fatal(r.URL)
			}
			return actionsResponse(r, 200, body), nil
		}).rest}
		if _, err := client.Read(generalTestRepo); err == nil {
			t.Fatal("accepted", body)
		}
	}
}
func TestActionsGeneralVisibilityAndOwnerConstraints(t *testing.T) {
	for _, kind := range []string{"retention", "access"} {
		client := actionsClient(t, func(r *http.Request) (*http.Response, error) {
			if r.Method != "GET" {
				t.Fatal("inapplicable desired mutated", r.Method)
			}
			if r.URL.Path == "/repos/owner/current" {
				visibility := "private"
				if kind == "retention" {
					visibility = "public"
				}
				return actionsResponse(r, 200, fmt.Sprintf(`{"full_name":"owner/current","visibility":"%s","owner":{"type":"User"}}`, visibility)), nil
			}
			if kind == "retention" {
				return actionsResponse(r, 200, `{"max_cache_retention_days":7}`), nil
			}
			return actionsResponse(r, 200, `{"access_level":"none"}`), nil
		}).rest
		var err error
		if kind == "retention" {
			_, err = (&CacheRetentionClient{rest: client}).Update(generalTestRepo, 91)
		} else {
			_, err = (&ActionsAccessClient{rest: client}).Update(generalTestRepo, "organization")
		}
		var pre interface{ ReadFailedBeforeWrite() }
		if err == nil || !errors.As(err, &pre) {
			t.Fatal(kind, err)
		}
	}
}
func TestActionsGeneralHTTPFailuresDoNotFollowRedirects(t *testing.T) {
	for _, kind := range []string{"retention", "storage", "access"} {
		for _, method := range []string{"GET", "PUT"} {
			for _, status := range []int{301, 403, 404, 422, 500} {
				writes := 0
				rest := actionsClient(t, func(r *http.Request) (*http.Response, error) {
					if r.URL.Host != "api.github.com" {
						t.Fatal("followed redirect", r.URL)
					}
					if r.Method == "PUT" {
						writes++
					}
					if r.Method == method {
						response := actionsResponse(r, status, `{"message":"unavailable"}`)
						response.Header.Set("Location", "https://unexpected.invalid/other")
						return response, nil
					}
					if r.URL.Path == "/repos/owner/current" {
						return actionsResponse(r, 200, generalTestMetadata), nil
					}
					body := `{"max_cache_retention_days":7}`
					if kind == "storage" {
						body = `{"max_cache_size_gb":7}`
					}
					if kind == "access" {
						body = `{"access_level":"none"}`
					}
					return actionsResponse(r, 200, body), nil
				}).rest
				_, err := generalUpdate(kind, rest)()
				var pre interface{ ReadFailedBeforeWrite() }
				if err == nil || errors.As(err, &pre) != (method == "GET") || method == "GET" && writes != 0 {
					t.Fatal(kind, method, status, writes, err)
				}
			}
		}
	}
}

func TestActionsAccessUserOwnerAndPostWriteOwnerChange(t *testing.T) {
	for _, change := range []bool{false, true} {
		writes := 0
		client := &ActionsAccessClient{rest: actionsClient(t, func(r *http.Request) (*http.Response, error) {
			if r.URL.Path == "/repos/owner/current" {
				owner := "User"
				if writes > 0 && change {
					owner = "Organization"
				}
				return actionsResponse(r, 200, fmt.Sprintf(`{"full_name":"OWNER/CURRENT","visibility":"private","owner":{"type":"%s"}}`, owner)), nil
			}
			if r.URL.Path != "/repos/owner/current/actions/permissions/access" {
				t.Fatal(r.URL)
			}
			if r.Method == "PUT" {
				writes++
				return actionsResponse(r, 204, ""), nil
			}
			level := "none"
			if writes > 0 {
				level = "user"
			}
			return actionsResponse(r, 200, fmt.Sprintf(`{"access_level":"%s"}`, level)), nil
		}).rest}
		wrote, err := client.Update(generalTestRepo, "user")
		if wrote == change || (err != nil) != change || writes != 1 {
			t.Fatal(change, wrote, writes, err)
		}
		var pre interface{ ReadFailedBeforeWrite() }
		if errors.As(err, &pre) {
			t.Fatal("postwrite applicability failure reported no effects", err)
		}
	}
}
