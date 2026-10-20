package github

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
)

func TestRetentionReadsRequireBothIntegers(t *testing.T) {
	for _, test := range []struct {
		body  string
		valid bool
	}{{`{"days":90,"maximum_allowed_days":90}`, true}, {`{"days":90,"maximum_allowed_days":14}`, true}, {`{"days":400,"maximum_allowed_days":400}`, true}, {`{}`, false}, {`null`, false}, {`{"days":90}`, false}, {`{"days":null,"maximum_allowed_days":90}`, false}, {`{"days":90,"maximum_allowed_days":null}`, false}, {`{"days":90,"maximum_allowed_days":0}`, false}, {`{"days":0,"maximum_allowed_days":90}`, false}, {`{"days":"90","maximum_allowed_days":90}`, false}, {`{"days":1.5,"maximum_allowed_days":90}`, false}, {`{"Days":90,"maximum_allowed_days":90}`, false}} {
		client := &RetentionClient{rest: actionsClient(t, func(r *http.Request) (*http.Response, error) {
			if r.Method != "GET" || r.URL.Path != "/repos/owner/current/actions/permissions/artifact-and-log-retention" {
				t.Fatal("unrelated request", r.Method, r.URL)
			}
			return actionsResponse(r, 200, test.body), nil
		}).rest}
		o, err := client.Read(repository.Repository{Owner: "owner", Name: "current"})
		if (err == nil) != test.valid {
			t.Fatal(test, err)
		}
		if !test.valid {
			if _, _, err := o.Values(); err == nil {
				t.Fatal("failed read acquired state")
			}
		}
	}
}
func TestRetentionFreshCapNarrowPUTAndReadback(t *testing.T) {
	for _, failure := range []string{"", "read", "cap", "write", "enforced", "missing maximum", "lower maximum", "changed maximum", "repair over cap"} {
		t.Run(failure, func(t *testing.T) {
			days, maximum := 90, 90
			desired := 30
			writes := 0
			if failure == "cap" {
				maximum = 14
			}
			if failure == "repair over cap" {
				maximum = 14
				desired = 7
			}
			client := &RetentionClient{rest: actionsClient(t, func(r *http.Request) (*http.Response, error) {
				if r.URL.Path != "/repos/owner/current/actions/permissions/artifact-and-log-retention" {
					t.Fatal("unrelated endpoint", r.URL)
				}
				if r.Method == "GET" {
					if failure == "read" {
						return actionsResponse(r, 403, `{}`), nil
					}
					if writes > 0 && failure == "missing maximum" {
						return actionsResponse(r, 200, fmt.Sprintf(`{"days":%d}`, days)), nil
					}
					return actionsResponse(r, 200, fmt.Sprintf(`{"days":%d,"maximum_allowed_days":%d}`, days, maximum)), nil
				}
				if r.Method != "PUT" {
					t.Fatal("unexpected method", r.Method)
				}
				writes++
				var body map[string]any
				json.NewDecoder(r.Body).Decode(&body)
				if !reflect.DeepEqual(body, map[string]any{"days": float64(desired)}) {
					t.Fatal("readonly or unrelated fields sent", body)
				}
				if failure == "write" {
					return actionsResponse(r, 422, `{}`), nil
				}
				if failure != "enforced" {
					days = desired
				}
				if failure == "changed maximum" {
					maximum = 60
				}
				if failure == "lower maximum" {
					maximum = 14
				}
				return actionsResponse(r, 204, ""), nil
			}).rest}
			wrote, err := client.Update(repository.Repository{Owner: "owner", Name: "current"}, desired)
			good := failure == "" || failure == "changed maximum" || failure == "repair over cap"
			if (err == nil) != good || wrote != good {
				t.Fatal(failure, wrote, err)
			}
			var pre interface{ ReadFailedBeforeWrite() }
			if errors.As(err, &pre) != (failure == "read" || failure == "cap") {
				t.Fatal("wrong certainty", failure, err)
			}
			want := 1
			if failure == "read" || failure == "cap" {
				want = 0
			}
			if writes != want {
				t.Fatal(failure, writes)
			}
			if good {
				wrote, err = client.Update(repository.Repository{Owner: "owner", Name: "current"}, desired)
				if wrote || err != nil || writes != 1 {
					t.Fatal("matching update wrote", wrote, writes, err)
				}
			}
		})
	}
	client := &RetentionClient{rest: actionsClient(t, func(*http.Request) (*http.Response, error) {
		t.Fatal("invalid local days accessed API")
		return nil, nil
	}).rest}
	if _, err := client.Update(repository.Repository{}, 0); err == nil {
		t.Fatal("zero accepted")
	}
	for _, method := range []string{"GET", "PUT"} {
		for _, status := range []int{301, 403, 404, 422, 500} {
			writes := 0
			client := &RetentionClient{rest: actionsClient(t, func(r *http.Request) (*http.Response, error) {
				if r.Method == "PUT" {
					writes++
				}
				if r.Method == method {
					response := actionsResponse(r, status, `{"message":"unavailable"}`)
					response.Header.Set("Location", "https://unexpected.invalid/other")
					return response, nil
				}
				return actionsResponse(r, 200, `{"days":90,"maximum_allowed_days":90}`), nil
			}).rest}
			_, err := client.Update(repository.Repository{}, 30)
			if err == nil || method == "GET" && writes != 0 {
				t.Fatal(method, status, writes, err)
			}
		}
	}
}
