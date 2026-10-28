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

func TestImmutableReleasesRead404DecisionAndComplete200(t *testing.T) {
	for _, test := range []struct {
		status                int
		body                  string
		valid, enabled, known bool
	}{
		{200, `{"enabled":true,"enforced_by_owner":false}`, true, true, true},
		{200, `{"enabled":true,"enforced_by_owner":true}`, true, true, true},
		{200, `{"enabled":false,"enforced_by_owner":false}`, true, false, true},
		{404, `{"message":"Not Found"}`, true, false, false},
		{404, `not JSON`, true, false, false},
		{404, `{"message":"Resource not accessible by integration"}`, true, false, false},
		{200, `{"enabled":true,"enforced_by_owner":false}{}`, false, false, false},
		{200, `{}`, false, false, false},
		{200, `null`, false, false, false},
		{200, `{"enabled":true}`, false, false, false},
		{200, `{"enabled":null,"enforced_by_owner":false}`, false, false, false},
		{200, `{"enabled":true,"enforced_by_owner":null}`, false, false, false},
		{200, `{"enabled":"true","enforced_by_owner":false}`, false, false, false},
		{200, `{"Enabled":true,"enforced_by_owner":false}`, false, false, false},
		{200, `{"enabled":false,"enforced_by_owner":true}`, false, false, false},
		{204, ``, false, false, false},
		{401, `{}`, false, false, false},
		{403, `{}`, false, false, false},
		{409, `{}`, false, false, false},
		{500, `{}`, false, false, false},
		{301, `{}`, false, false, false},
	} {
		t.Run(fmt.Sprint(test.status, "/", test.body), func(t *testing.T) {
			calls := 0
			client := &ImmutableReleasesClient{rest: actionsClient(t, func(r *http.Request) (*http.Response, error) {
				calls++
				if r.Method != "GET" || r.URL.Path != "/repos/owner/current/immutable-releases" {
					t.Fatal(r.Method, r.URL)
				}
				response := actionsResponse(r, test.status, test.body)
				response.Header.Set("Location", "https://unexpected.invalid/other")
				return response, nil
			}).rest}
			state, err := client.Read(repository.Repository{Owner: "owner", Name: "current"})
			if (err == nil) != test.valid || calls != 1 {
				t.Fatal(state, err, calls)
			}
			enabled, _, known, err := state.Values()
			if test.valid && (err != nil || enabled != test.enabled || known != test.known) {
				t.Fatal(enabled, known, err)
			}
			if !test.valid && err == nil {
				t.Fatal("failed read acquired state")
			}
		})
	}
}

func TestImmutableReleasesFreshToggleAndUncertainty(t *testing.T) {
	for _, after := range []bool{false, true} {
		for _, mode := range []string{"success", "matching", "owner-enforced", "read403", "read-transport", "write409", "write404", "write-transport", "write200", "ignored", "readback403"} {
			t.Run(fmt.Sprint(after, "/", mode), func(t *testing.T) {
				enabled := !after
				if mode == "matching" {
					enabled = after
				}
				if mode == "owner-enforced" {
					enabled = true
				}
				writes, reads := 0, 0
				client := &ImmutableReleasesClient{rest: actionsClient(t, func(r *http.Request) (*http.Response, error) {
					if r.URL.Path != "/repos/owner/current/immutable-releases" {
						t.Fatal("individual release/tag/asset or unrelated endpoint", r.URL)
					}
					if r.Method == "GET" {
						reads++
						if mode == "read-transport" {
							return nil, errors.New("fixture transport failure")
						}
						if mode == "read403" || writes > 0 && mode == "readback403" {
							return actionsResponse(r, 403, `{}`), nil
						}
						if !enabled {
							return actionsResponse(r, 404, `{"message":"Not Found"}`), nil
						}
						return actionsResponse(r, 200, fmt.Sprintf(`{"enabled":true,"enforced_by_owner":%t}`, mode == "owner-enforced")), nil
					}
					want := "DELETE"
					if after {
						want = "PUT"
					}
					if r.Method != want {
						t.Fatal(r.Method, want)
					}
					if r.Body != nil {
						body, _ := io.ReadAll(r.Body)
						if len(body) > 0 {
							t.Fatal("toggle body", string(body))
						}
					}
					writes++
					if mode == "write-transport" {
						return nil, errors.New("fixture write transport")
					}
					if mode == "write409" {
						return actionsResponse(r, 409, `{}`), nil
					}
					if mode == "write404" {
						return actionsResponse(r, 404, `{}`), nil
					}
					if mode == "write200" {
						return actionsResponse(r, 200, `{}`), nil
					}
					if mode != "ignored" {
						enabled = after
					}
					return actionsResponse(r, 204, ""), nil
				}).rest}
				change, _ := settings.NewSecurityChange("security.immutable_releases", !after, after)
				wrote, err := client.Update(repository.Repository{Owner: "owner", Name: "current"}, change)
				good := mode == "success" || mode == "matching" || mode == "owner-enforced" && after
				if (err == nil) != good || wrote != (mode == "success") {
					t.Fatal(wrote, err)
				}
				pre := mode == "read403" || mode == "read-transport" || mode == "owner-enforced" && !after
				var certainty interface{ ReadFailedBeforeWrite() }
				if errors.As(err, &certainty) != pre {
					t.Fatal("incorrect failure certainty", err)
				}
				wantWrites := 1
				if pre || mode == "matching" || mode == "owner-enforced" {
					wantWrites = 0
				}
				if writes != wantWrites {
					t.Fatal("unexpected write/retry", writes, wantWrites)
				}
				if !good && !pre && !strings.Contains(err.Error(), "may already have applied") {
					t.Fatal(err)
				}
				if mode == "success" {
					wrote, err = client.Update(repository.Repository{Owner: "owner", Name: "current"}, change)
					if wrote || err != nil || writes != 1 {
						t.Fatal("repeat toggle wrote", wrote, err, writes)
					}
				}
			})
		}
	}
}

func TestImmutableReleasesRejectsOtherSecurityChanges(t *testing.T) {
	client := &ImmutableReleasesClient{rest: actionsClient(t, func(r *http.Request) (*http.Response, error) { t.Fatal("cross-family request", r.URL); return nil, nil }).rest}
	change, _ := settings.NewSecurityChange("security.secret_scanning", false, true)
	if _, err := client.Update(repository.Repository{}, change); err == nil {
		t.Fatal("accepted another family")
	}
}
