package github

import (
	"encoding/json"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func actionsPointer[T any](v T) *T { return &v }
func actionsClient(t *testing.T, handle transportFunc) *ActionsPolicyClient {
	t.Helper()
	rest, err := api.NewRESTClient(api.ClientOptions{Host: "github.com", AuthToken: "dummy-token", LogIgnoreEnv: true, Transport: handle, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }})
	if err != nil {
		t.Fatal(err)
	}
	return &ActionsPolicyClient{rest: rest}
}
func actionsResponse(req *http.Request, status int, body string) *http.Response {
	return &http.Response{Request: req, StatusCode: status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}
}

func TestActionsReadIsCompleteAndSelectedIsConditional(t *testing.T) {
	base := `{"enabled":false,"allowed_actions":"all","sha_pinning_required":false,"selected_actions_url":"https://untrusted.invalid"}`
	for _, body := range []string{base, strings.Replace(base, `"all"`, `"local_only"`, 1), strings.Replace(base, `"all"`, `"selected"`, 1), `{}`, `null`, strings.Replace(base, `"enabled":false`, `"enabled":null`, 1), strings.Replace(base, `"sha_pinning_required":false`, `"sha_pinning_required":"false"`, 1), strings.Replace(base, `"all"`, `"unknown"`, 1)} {
		calls := []string{}
		client := actionsClient(t, func(req *http.Request) (*http.Response, error) {
			calls = append(calls, req.URL.Path)
			if req.Method != "GET" {
				t.Fatal("read wrote", req.Method)
			}
			if strings.HasSuffix(req.URL.Path, "/selected-actions") {
				return actionsResponse(req, 200, `{"github_owned_allowed":false,"verified_allowed":false,"patterns_allowed":[]}`), nil
			}
			return actionsResponse(req, 200, body), nil
		})
		state, err := client.ReadActionsPolicy(repository.Repository{Owner: "owner", Name: "current"}, true)
		valid := strings.Contains(body, `"allowed_actions":"all"`) || strings.Contains(body, `"allowed_actions":"local_only"`) || strings.Contains(body, `"allowed_actions":"selected"`)
		valid = valid && !strings.Contains(body, "null") && !strings.Contains(body, `"sha_pinning_required":"false"`)
		if (err == nil) != valid {
			t.Fatalf("body=%s state=%v error=%v", body, state, err)
		}
		if valid && state.AllowedActions == "selected" {
			if len(calls) != 2 || state.Selected == nil || state.Selected.PatternsAllowed == nil {
				t.Fatal(calls, state)
			}
		} else if len(calls) != 1 || state.Selected != nil {
			t.Fatal(calls, state)
		}
		if err != nil && !reflect.DeepEqual(state, settings.ActionsPolicyState{}) {
			t.Fatal("partial read escaped", state)
		}
	}
	for _, selected := range []string{`{}`, `{"github_owned_allowed":null,"verified_allowed":false,"patterns_allowed":[]}`, `{"github_owned_allowed":false,"verified_allowed":false,"patterns_allowed":null}`, `{"github_owned_allowed":false,"verified_allowed":false,"patterns_allowed":[null]}`, `{"github_owned_allowed":false,"verified_allowed":false,"patterns_allowed":[false]}`} {
		client := actionsClient(t, func(req *http.Request) (*http.Response, error) {
			if strings.HasSuffix(req.URL.Path, "selected-actions") {
				return actionsResponse(req, 200, selected), nil
			}
			return actionsResponse(req, 200, strings.Replace(base, `"all"`, `"selected"`, 1)), nil
		})
		state, err := client.ReadActionsPolicy(repository.Repository{Owner: "owner", Name: "current"}, true)
		if err == nil || !reflect.DeepEqual(state, settings.ActionsPolicyState{}) {
			t.Fatal(selected, state, err)
		}
	}
}

func TestActionsApplyPreservesCompanionsAndOrdersWrites(t *testing.T) {
	for _, test := range []struct {
		name, policy string
		desired      settings.ActionsPolicyDeclaration
		want         []map[string]any
	}{
		{"disable", "all", settings.ActionsPolicyDeclaration{Enabled: actionsPointer(false)}, []map[string]any{{"enabled": false}}},
		{"sha only", "local_only", settings.ActionsPolicyDeclaration{SHAPinningRequired: actionsPointer(false)}, []map[string]any{{"enabled": false, "sha_pinning_required": false}}},
		{"local policy", "all", settings.ActionsPolicyDeclaration{AllowedActions: actionsPointer("local_only")}, []map[string]any{{"enabled": false, "allowed_actions": "local_only"}}},
		{"selected partial", "selected", settings.ActionsPolicyDeclaration{SelectedActions: settings.SelectedActionsDeclaration{VerifiedAllowed: actionsPointer(false)}}, []map[string]any{{"github_owned_allowed": true, "verified_allowed": false, "patterns_allowed": []any{"org/*", "!org/bad@*", "org/*"}}}},
		{"transition", "all", settings.ActionsPolicyDeclaration{AllowedActions: actionsPointer("selected"), SelectedActions: settings.SelectedActionsDeclaration{GitHubOwnedAllowed: actionsPointer(false), VerifiedAllowed: actionsPointer(false), PatternsAllowed: actionsPointer([]string{})}}, []map[string]any{{"enabled": false, "allowed_actions": "selected"}, {"github_owned_allowed": false, "verified_allowed": false, "patterns_allowed": []any{}}}},
		{"enable after selection", "all", settings.ActionsPolicyDeclaration{Enabled: actionsPointer(true), AllowedActions: actionsPointer("selected"), SelectedActions: settings.SelectedActionsDeclaration{GitHubOwnedAllowed: actionsPointer(false), VerifiedAllowed: actionsPointer(false), PatternsAllowed: actionsPointer([]string{})}}, []map[string]any{{"enabled": false, "allowed_actions": "selected"}, {"github_owned_allowed": false, "verified_allowed": false, "patterns_allowed": []any{}}, {"enabled": true}}},
	} {
		t.Run(test.name, func(t *testing.T) {
			base := map[string]any{"enabled": false, "allowed_actions": test.policy, "sha_pinning_required": true}
			selected := map[string]any{"github_owned_allowed": true, "verified_allowed": true, "patterns_allowed": []any{"org/*", "!org/bad@*", "org/*"}}
			writes := []map[string]any{}
			operations := []string{}
			client := actionsClient(t, func(req *http.Request) (*http.Response, error) {
				isSelected := strings.HasSuffix(req.URL.Path, "/selected-actions")
				target := base
				if isSelected {
					target = selected
					if test.name == "enable after selection" && req.Method == "PUT" && base["enabled"] != false {
						t.Fatal("enabled before installing selection")
					}
					if base["allowed_actions"] != "selected" {
						t.Fatal("inactive selected request")
					}
				}
				operations = append(operations, req.Method+" "+req.URL.Path)
				if req.Method == "PUT" {
					var payload map[string]any
					if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
						t.Fatal(err)
					}
					writes = append(writes, payload)
					for k, v := range payload {
						target[k] = v
					}
					return actionsResponse(req, 204, ""), nil
				}
				if req.Method != "GET" {
					t.Fatal("unexpected method", req.Method)
				}
				body, _ := json.Marshal(target)
				return actionsResponse(req, 200, string(body)), nil
			})
			if err := client.ApplyActionsPolicy(repository.Repository{Owner: "owner", Name: "current"}, test.desired); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(writes, test.want) {
				t.Fatalf("writes=%v want=%v", writes, test.want)
			}
			if test.name == "transition" && !reflect.DeepEqual(operations, []string{"GET /repos/owner/current/actions/permissions", "PUT /repos/owner/current/actions/permissions", "PUT /repos/owner/current/actions/permissions/selected-actions", "GET /repos/owner/current/actions/permissions", "GET /repos/owner/current/actions/permissions/selected-actions"}) {
				t.Fatal(operations)
			}
		})
	}
}

func TestActionsApplyPreflightAndFailures(t *testing.T) {
	for _, d := range []settings.ActionsPolicyDeclaration{{AllowedActions: actionsPointer("selected")}, {SelectedActions: settings.SelectedActionsDeclaration{VerifiedAllowed: actionsPointer(false)}}} {
		calls := 0
		client := actionsClient(t, func(req *http.Request) (*http.Response, error) {
			calls++
			if req.Method != "GET" {
				t.Fatal("preflight wrote")
			}
			return actionsResponse(req, 200, `{"enabled":false,"allowed_actions":"all","sha_pinning_required":false}`), nil
		})
		if err := client.ApplyActionsPolicy(repository.Repository{}, d); err == nil || calls != 1 {
			t.Fatal(err, calls)
		}
	}
	for _, status := range []int{200, 202, 301, 403, 404, 409, 422} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			writes := 0
			client := actionsClient(t, func(req *http.Request) (*http.Response, error) {
				if req.Method == "PUT" {
					writes++
					return actionsResponse(req, status, `{}`), nil
				}
				return actionsResponse(req, 200, `{"enabled":false,"allowed_actions":"all","sha_pinning_required":true}`), nil
			})
			err := client.ApplyActionsPolicy(repository.Repository{}, settings.ActionsPolicyDeclaration{Enabled: actionsPointer(true)})
			if err == nil || writes != 1 || !strings.Contains(err.Error(), "may already have applied") {
				t.Fatal(err, writes)
			}
		})
	}
	client := actionsClient(t, func(*http.Request) (*http.Response, error) { t.Fatal("invalid/unmanaged invoked API"); return nil, nil })
	if err := client.ApplyActionsPolicy(repository.Repository{}, settings.ActionsPolicyDeclaration{}); err != nil {
		t.Fatal(err)
	}
	if err := client.ApplyActionsPolicy(repository.Repository{}, settings.ActionsPolicyDeclaration{AllowedActions: actionsPointer("unknown")}); err == nil {
		t.Fatal("invalid policy accepted")
	}
	// A successful write followed by a mismatching readback is not success or rollback.
	client = actionsClient(t, func(req *http.Request) (*http.Response, error) {
		if req.Method == "PUT" {
			return actionsResponse(req, 204, ""), nil
		}
		return actionsResponse(req, 200, `{"enabled":false,"allowed_actions":"all","sha_pinning_required":true}`), nil
	})
	if err := client.ApplyActionsPolicy(repository.Repository{}, settings.ActionsPolicyDeclaration{Enabled: actionsPointer(true)}); err == nil || !strings.Contains(err.Error(), "may already have applied") {
		t.Fatal(err)
	}
}

func TestActionsSelectedFailureNeverEnablesOrInventsState(t *testing.T) {
	for _, method := range []string{"GET", "PUT"} {
		t.Run(method, func(t *testing.T) {
			policy := "selected"
			if method == "PUT" {
				policy = "all"
			}
			writes := 0
			client := actionsClient(t, func(req *http.Request) (*http.Response, error) {
				if strings.HasSuffix(req.URL.Path, "/selected-actions") {
					if req.Method != method {
						t.Fatal("unexpected selected request", req.Method)
					}
					return actionsResponse(req, 403, `{"message":"organization restriction"}`), nil
				}
				if req.Method == "PUT" {
					writes++
					var body map[string]any
					json.NewDecoder(req.Body).Decode(&body)
					if body["enabled"] != false {
						t.Fatal("failure path enabled Actions", body)
					}
					return actionsResponse(req, 204, ""), nil
				}
				return actionsResponse(req, 200, `{"enabled":false,"allowed_actions":"`+policy+`","sha_pinning_required":true}`), nil
			})
			d := settings.ActionsPolicyDeclaration{Enabled: actionsPointer(true), AllowedActions: actionsPointer("selected"), SelectedActions: settings.SelectedActionsDeclaration{GitHubOwnedAllowed: actionsPointer(false), VerifiedAllowed: actionsPointer(false), PatternsAllowed: actionsPointer([]string{})}}
			err := client.ApplyActionsPolicy(repository.Repository{Owner: "owner", Name: "current"}, d)
			want := 0
			if method == "PUT" {
				want = 1
			}
			if err == nil || writes != want {
				t.Fatal(writes, err)
			}
		})
	}
}
