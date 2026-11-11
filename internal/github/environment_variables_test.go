package github

import (
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func TestEnvironmentVariableFreshUpsert(t *testing.T) {
	for _, tc := range []struct {
		name         string
		firstStatus  int
		first, after string
		writeStatus  int
		methods      string
		fail, pre    bool
	}{
		{"create", 404, `{}`, `{"name":"DISPLAY_NAME","value":"日本語"}`, 201, "GET GET POST GET", false, false},
		{"update", 200, `{"name":"DISPLAY_NAME","value":"old"}`, `{"name":"DISPLAY_NAME","value":"日本語"}`, 204, "GET GET PATCH GET", false, false},
		{"matching", 200, `{"name":"DISPLAY_NAME","value":"日本語"}`, ``, 204, "GET GET", false, false},
		{"preread denied", 403, `{}`, ``, 204, "GET GET", true, true},
		{"write denied", 404, `{}`, ``, 403, "GET GET POST", true, false},
		{"wrong write status", 404, `{}`, ``, 200, "GET GET POST", true, false},
		{"readback mismatch", 404, `{}`, `{"name":"DISPLAY_NAME","value":"old"}`, 201, "GET GET POST GET", true, false},
		{"readback malformed", 404, `{}`, `{"name":"DISPLAY_NAME","value":null}`, 201, "GET GET POST GET", true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var methods []string
			client := topicsClient(t, transportFunc(func(req *http.Request) (*http.Response, error) {
				methods = append(methods, req.Method)
				path := "/repos/owner/current/environments/Production%2FEU%20%231"
				if len(methods) == 1 {
					if req.Method != "GET" || req.URL.EscapedPath() != path {
						t.Fatalf("environment request: %s %s", req.Method, req.URL)
					}
					// Unknown protection is harmless: this path never owns or writes policy.
					return labelsResponse(req, 200, `{"id":42,"name":"production/EU #1","protection_rules":[{"type":"future"}]}`), nil
				}
				path += "/variables"
				if req.Method != "POST" {
					path += "/DISPLAY_NAME"
				}
				if req.URL.EscapedPath() != path {
					t.Fatalf("variable path: %s", req.URL)
				}
				if len(methods) == 2 {
					return labelsResponse(req, tc.firstStatus, tc.first), nil
				}
				if req.Method == "POST" || req.Method == "PATCH" {
					var body map[string]string
					if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
						t.Fatal(err)
					}
					want := map[string]string{"value": "日本語"}
					if req.Method == "POST" {
						want["name"] = "DISPLAY_NAME"
					}
					if !reflect.DeepEqual(body, want) {
						t.Fatalf("payload=%v", body)
					}
					return labelsResponse(req, tc.writeStatus, `{}`), nil
				}
				return labelsResponse(req, 200, tc.after), nil
			}))
			wrote, err := client.UpsertEnvironmentVariable(repository.Repository{Owner: "owner", Name: "current"}, "Production/EU #1", settings.RepositoryVariable{Name: "display_name", Value: "日本語"})
			var pre interface{ ReadFailedBeforeWrite() }
			if (err != nil) != tc.fail || errors.As(err, &pre) != tc.pre || wrote != (!tc.fail && tc.name != "matching") || strings.Join(methods, " ") != tc.methods {
				t.Fatalf("wrote=%v err=%v methods=%v", wrote, err, methods)
			}
		})
	}
}

func TestEnvironmentVariableIdentityFailuresStopBeforeVariableRequests(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status int
		body   string
	}{
		{"missing", 404, `{"message":"private response"}`},
		{"denied", 403, `{"message":"private response"}`},
		{"redirect", 302, `{}`},
		{"missing id", 200, `{"name":"prod"}`},
		{"null id", 200, `{"id":null,"name":"prod"}`},
		{"zero id", 200, `{"id":0,"name":"prod"}`},
		{"wrong id type", 200, `{"id":"42","name":"prod"}`},
		{"wrong name", 200, `{"id":42,"name":"other"}`},
		{"trailing", 200, `{"id":42,"name":"prod"} {}`},
	} {
		for _, write := range []bool{false, true} {
			t.Run(tc.name+map[bool]string{false: " read", true: " write"}[write], func(t *testing.T) {
				calls := 0
				client := topicsClient(t, transportFunc(func(req *http.Request) (*http.Response, error) {
					calls++
					if req.Method != "GET" || req.URL.Path != "/repos/owner/current/environments/prod" {
						t.Fatalf("unexpected request %s %s", req.Method, req.URL)
					}
					return labelsResponse(req, tc.status, tc.body), nil
				}))
				repo := repository.Repository{Owner: "owner", Name: "current"}
				var err error
				if write {
					_, err = client.UpsertEnvironmentVariable(repo, "prod", settings.RepositoryVariable{Name: "ITEM", Value: "desired"})
				} else {
					_, err = client.ReadEnvironmentVariable(repo, "prod", "ITEM")
				}
				var pre interface{ ReadFailedBeforeWrite() }
				if err == nil || calls != 1 || errors.As(err, &pre) != write || strings.Contains(err.Error(), "private response") {
					t.Fatalf("err=%v calls=%d", err, calls)
				}
			})
		}
	}
}

func TestEnvironmentVariableSelectedRead(t *testing.T) {
	for _, absent := range []bool{false, true} {
		calls := 0
		client := topicsClient(t, transportFunc(func(req *http.Request) (*http.Response, error) {
			calls++
			if req.Method != "GET" {
				t.Fatal(req.Method)
			}
			if calls == 1 {
				return labelsResponse(req, 200, `{"id":42,"name":"prod"}`), nil
			}
			if req.URL.Path != "/repos/owner/current/environments/prod/variables/ITEM" {
				t.Fatal(req.URL)
			}
			if absent {
				return labelsResponse(req, 404, `{}`), nil
			}
			return labelsResponse(req, 200, `{"name":"ITEM","value":"value"}`), nil
		}))
		observed, err := client.ReadEnvironmentVariable(repository.Repository{Owner: "owner", Name: "current"}, "prod", "item")
		value, valueErr := observed.Value()
		if err != nil || valueErr != nil || (value == nil) != absent || calls != 2 {
			t.Fatalf("value=%v err=%v valueErr=%v calls=%d", value, err, valueErr, calls)
		}
	}
}

func TestEnvironmentVariableInvalidIntentIsLocal(t *testing.T) {
	client := topicsClient(t, transportFunc(func(req *http.Request) (*http.Response, error) {
		t.Fatalf("unexpected request: %s", req.URL)
		return nil, nil
	}))
	repo := repository.Repository{Owner: "owner", Name: "current"}
	for _, tc := range []struct{ environment, name, value string }{{"", "ITEM", "value"}, {" prod", "ITEM", "value"}, {"prod", "GITHUB_ITEM", "value"}, {"prod", "ITEM", ""}} {
		_, err := client.UpsertEnvironmentVariable(repo, tc.environment, settings.RepositoryVariable{Name: tc.name, Value: tc.value})
		var pre interface{ ReadFailedBeforeWrite() }
		if err == nil || !errors.As(err, &pre) {
			t.Fatalf("invalid upsert err=%v", err)
		}
		if tc.value != "" {
			if _, err := client.ReadEnvironmentVariable(repo, tc.environment, tc.name); err == nil {
				t.Fatal("invalid read passed")
			}
		}
	}
}
