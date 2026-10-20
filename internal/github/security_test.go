package github

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func securityResponse(s, p bool) string {
	a, b := "disabled", "disabled"
	if s {
		a = "enabled"
	}
	if p {
		b = "enabled"
	}
	return fmt.Sprintf(`{"security_and_analysis":{"secret_scanning":{"status":%q},"secret_scanning_push_protection":{"status":%q},"future_security":{"status":"enabled"}}}`, a, b)
}
func TestSecretScanningCompleteReads(t *testing.T) {
	for _, visibility := range []string{"public", "private", "internal"} {
		for _, enabled := range []bool{false, true} {
			var fields map[string]any
			json.Unmarshal([]byte(securityResponse(enabled, false)), &fields)
			fields["visibility"] = visibility
			body, _ := json.Marshal(fields)
			client := &SecretScanningClient{rest: actionsClient(t, func(r *http.Request) (*http.Response, error) { return actionsResponse(r, 200, string(body)), nil }).rest}
			observed, err := client.Read(repository.Repository{Owner: "owner", Name: "current"})
			if err != nil {
				t.Fatal(visibility, err)
			}
			scanning, push, err := observed.Values()
			if err != nil || scanning != enabled || push {
				t.Fatal(visibility, scanning, push, err)
			}
		}
	}

	for _, body := range []string{`{}`, `null`, `{"security_and_analysis":null}`, `{"security_and_analysis":{}}`, `{"security_and_analysis":{"secret_scanning":{"status":"enabled"}}}`, `{"security_and_analysis":{"secret_scanning":{"status":null},"secret_scanning_push_protection":{"status":"disabled"}}}`, `{"security_and_analysis":{"secret_scanning":{"status":"unknown"},"secret_scanning_push_protection":{"status":"disabled"}}}`} {
		client := &SecretScanningClient{rest: actionsClient(t, func(r *http.Request) (*http.Response, error) {
			if r.Method != "GET" {
				t.Fatal("read mutated")
			}
			return actionsResponse(r, 200, body), nil
		}).rest}
		observed, err := client.Read(repository.Repository{Owner: "owner", Name: "current"})
		if err == nil {
			t.Fatal("incomplete state accepted", body)
		}
		if _, _, err := observed.Values(); err == nil {
			t.Fatal("failed read acquired")
		}
	}
	for _, status := range []int{301, 401, 403, 404, 422, 500} {
		calls := 0
		client := &SecretScanningClient{rest: actionsClient(t, func(r *http.Request) (*http.Response, error) {
			calls++
			response := actionsResponse(r, status, `{"message":"unavailable"}`)
			response.Header.Set("Location", "https://unexpected.invalid/repos/other/target")
			return response, nil
		}).rest}
		if _, err := client.Read(repository.Repository{}); err == nil || calls != 1 {
			t.Fatal(status, calls, err)
		}
	}
}
func TestSecretScanningNarrowPayloadReadbackAndFreshPreflight(t *testing.T) {
	for _, failure := range []string{"", "read", "write", "enforced", "hidden readback", "companion"} {
		t.Run(failure, func(t *testing.T) {
			s, p := true, false
			writes, reads := 0, 0
			client := &SecretScanningClient{rest: actionsClient(t, func(r *http.Request) (*http.Response, error) {
				if r.URL.Path != "/repos/owner/current" {
					t.Fatal("unrelated path", r.URL)
				}
				if r.Method == "GET" {
					reads++
					if failure == "read" {
						return actionsResponse(r, 403, `{}`), nil
					}
					if writes > 0 && failure == "hidden readback" {
						return actionsResponse(r, 200, `{}`), nil
					}
					return actionsResponse(r, 200, securityResponse(s, p)), nil
				}
				if r.Method != "PATCH" {
					t.Fatal("unexpected method", r.Method)
				}
				writes++
				var body map[string]any
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Fatal(err)
				}
				want := map[string]any{"security_and_analysis": map[string]any{"secret_scanning_push_protection": map[string]any{"status": "enabled"}}}
				if !reflect.DeepEqual(body, want) {
					t.Fatal("undeclared fields in payload", body)
				}
				if failure == "write" {
					return actionsResponse(r, 422, `{}`), nil
				}
				if failure != "enforced" {
					p = true
				}
				if failure == "companion" {
					s = false
				}
				return actionsResponse(r, 200, `{}`), nil
			}).rest}
			change, _ := settings.NewSecurityChange("security.secret_scanning_push_protection", false, true)
			wrote, err := client.Update(repository.Repository{Owner: "owner", Name: "current"}, change)
			if (err == nil) != (failure == "") || wrote != (failure == "") {
				t.Fatal(failure, wrote, err)
			}
			var beforeWrite interface{ ReadFailedBeforeWrite() }
			if errors.As(err, &beforeWrite) != (failure == "read") {
				t.Fatal("wrong write certainty", failure, err)
			}
			if failure == "read" && writes != 0 {
				t.Fatal("read failure wrote")
			}
			if failure == "" {
				if writes != 1 || reads != 2 {
					t.Fatal(writes, reads)
				}
				wrote, err = client.Update(repository.Repository{Owner: "owner", Name: "current"}, change)
				if wrote || err != nil || writes != 1 {
					t.Fatal("matching retry wrote", wrote, writes, err)
				}
			}
		})
	}
	for _, change := range []settings.SecurityChange{func() settings.SecurityChange {
		c, _ := settings.NewSecurityChange("security.secret_scanning", true, false)
		return c
	}(), func() settings.SecurityChange {
		c, _ := settings.NewSecurityChange("security.secret_scanning_push_protection", false, true)
		return c
	}()} {
		client := &SecretScanningClient{rest: actionsClient(t, func(r *http.Request) (*http.Response, error) {
			if r.Method != "GET" {
				t.Fatal("fresh prerequisite failure mutated")
			}
			return actionsResponse(r, 200, securityResponse(false, true)), nil
		}).rest}
		if _, err := client.Update(repository.Repository{}, change); err == nil {
			t.Fatal("invalid fresh prerequisite accepted", change)
		}
	}
}
