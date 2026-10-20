package github

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func TestPrivateReportingReadApplicabilityAndCompleteBoolean(t *testing.T) {
	for _, meta := range []string{`{"visibility":"public","security_and_analysis":null}`, `{"visibility":"private"}`, `{"visibility":"internal"}`, `{}`, `{"visibility":null}`, `{"Visibility":"public"}`, `{"visibility":true}`, `{"visibility":"unknown"}`} {
		for _, body := range []string{`{"enabled":false}`, `{"enabled":true}`, `{}`, `null`, `{"enabled":null}`, `{"enabled":"false"}`, `{"Enabled":true}`} {
			calls := 0
			client := &PrivateReportingClient{rest: actionsClient(t, func(r *http.Request) (*http.Response, error) {
				calls++
				if r.Method != "GET" {
					t.Fatal("read mutated")
				}
				switch r.URL.Path {
				case "/repos/owner/current":
					return actionsResponse(r, 200, meta), nil
				case "/repos/owner/current/private-vulnerability-reporting":
					return actionsResponse(r, 200, body), nil
				default:
					t.Fatal("unrelated endpoint", r.URL)
					return nil, nil
				}
			}).rest}
			observed, err := client.Read(repository.Repository{Owner: "owner", Name: "current"})
			valid := strings.Contains(meta, `"visibility":"public"`) && (body == `{"enabled":false}` || body == `{"enabled":true}`)
			if (err == nil) != valid {
				t.Fatal(meta, body, observed, err)
			}
			value, valueErr := observed.Value()
			if valid {
				if valueErr != nil || value != (body == `{"enabled":true}`) {
					t.Fatal(meta, body, value, valueErr)
				}
			} else if valueErr == nil {
				t.Fatal("failed read acquired state", meta, body)
			}
			wantCalls := 1
			if strings.Contains(meta, `"visibility":"public"`) {
				wantCalls = 2
			}
			if calls != wantCalls {
				t.Fatal(meta, calls)
			}
		}
	}
	for _, endpoint := range []string{"metadata", "status"} {
		for _, status := range []int{301, 403, 404, 422, 500} {
			client := &PrivateReportingClient{rest: actionsClient(t, func(r *http.Request) (*http.Response, error) {
				if endpoint == "metadata" || strings.HasSuffix(r.URL.Path, "/private-vulnerability-reporting") {
					response := actionsResponse(r, status, `{"message":"unavailable"}`)
					response.Header.Set("Location", "https://unexpected.invalid/other")
					return response, nil
				}
				return actionsResponse(r, 200, `{"visibility":"public"}`), nil
			}).rest}
			if _, err := client.Read(repository.Repository{Owner: "owner", Name: "current"}); err == nil {
				t.Fatal(endpoint, status)
			}
		}
	}
}
func TestPrivateReportingToggleAndCertainty(t *testing.T) {
	for _, after := range []bool{false, true} {
		for _, failure := range []string{"", "metadata", "status", "write", "verify boolean", "verify visibility", "enforced", "matching"} {
			t.Run(fmt.Sprint(after, "/", failure), func(t *testing.T) {
				enabled := !after
				if failure == "matching" {
					enabled = after
				}
				writes := 0
				client := &PrivateReportingClient{rest: actionsClient(t, func(r *http.Request) (*http.Response, error) {
					if r.URL.Path == "/repos/owner/current" {
						if r.Method != "GET" {
							t.Fatal("metadata mutated")
						}
						if failure == "metadata" {
							return actionsResponse(r, 403, `{}`), nil
						}
						if writes > 0 && failure == "verify visibility" {
							return actionsResponse(r, 200, `{"visibility":"private"}`), nil
						}
						return actionsResponse(r, 200, `{"visibility":"public"}`), nil
					}
					if r.URL.Path != "/repos/owner/current/private-vulnerability-reporting" {
						t.Fatal("advisory/report/unrelated endpoint", r.URL)
					}
					if r.Method == "GET" {
						if failure == "status" || writes > 0 && failure == "verify boolean" {
							return actionsResponse(r, 200, `{"enabled":null}`), nil
						}
						return actionsResponse(r, 200, fmt.Sprintf(`{"enabled":%t}`, enabled)), nil
					}
					want := "DELETE"
					if after {
						want = "PUT"
					}
					if r.Method != want {
						t.Fatal("wrong toggle method", r.Method, want)
					}
					if r.Body != nil {
						body, _ := io.ReadAll(r.Body)
						if len(body) != 0 {
							t.Fatal("unexpected request body", string(body))
						}
					}
					writes++
					if failure == "write" {
						return actionsResponse(r, 422, `{}`), nil
					}
					if failure != "enforced" {
						enabled = after
					}
					return actionsResponse(r, 204, ""), nil
				}).rest}
				change, _ := settings.NewSecurityChange("security.private_vulnerability_reporting", !after, after)
				wrote, err := client.Update(repository.Repository{Owner: "owner", Name: "current"}, change)
				good := failure == "" || failure == "matching"
				if (err == nil) != good || wrote != (failure == "") {
					t.Fatal(after, failure, wrote, err)
				}
				var pre interface{ ReadFailedBeforeWrite() }
				if errors.As(err, &pre) != (failure == "metadata" || failure == "status") {
					t.Fatal("wrong certainty", failure, err)
				}
				want := 1
				if failure == "metadata" || failure == "status" || failure == "matching" {
					want = 0
				}
				if writes != want {
					t.Fatal(failure, writes, want)
				}
				if failure == "" {
					wrote, err = client.Update(repository.Repository{Owner: "owner", Name: "current"}, change)
					if wrote || err != nil || writes != 1 {
						t.Fatal("matching update wrote", wrote, writes, err)
					}
				}
			})
		}
	}
}
func TestSecurityClientsRejectCrossFamilyChangesBeforeRequests(t *testing.T) {
	rest := actionsClient(t, func(r *http.Request) (*http.Response, error) {
		t.Fatal("cross-family change accessed API", r.Method, r.URL)
		return nil, nil
	}).rest
	secret, _ := settings.NewSecurityChange("security.secret_scanning", false, true)
	private, _ := settings.NewSecurityChange("security.private_vulnerability_reporting", false, true)
	if _, err := (&PrivateReportingClient{rest: rest}).Update(repository.Repository{}, secret); err == nil {
		t.Fatal("secret change accepted")
	}
	if _, err := (&SecretScanningClient{rest: rest}).Update(repository.Repository{}, private); err == nil {
		t.Fatal("PVR change accepted")
	}
}
