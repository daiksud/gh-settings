package github

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func TestEnvironmentPatternsCompleteTypedPagination(t *testing.T) {
	for _, mode := range []string{"pages", "missing-type", "null", "partial", "duplicate-id", "duplicate-pair", "count-changed", "denied"} {
		t.Run(mode, func(t *testing.T) {
			calls := 0
			c := &EnvironmentsClient{rest: actionsClient(t, func(r *http.Request) (*http.Response, error) {
				calls++
				if r.Method != "GET" || !strings.HasSuffix(r.URL.Path, "/preview%2Ftest/deployment-branch-policies") && !strings.Contains(r.URL.EscapedPath(), "preview%2Ftest/deployment-branch-policies") {
					t.Fatal(r.Method, r.URL)
				}
				if mode == "denied" {
					return actionsResponse(r, 403, `{}`), nil
				}
				if mode == "null" {
					return actionsResponse(r, 200, `{"total_count":0,"branch_policies":null}`), nil
				}
				entries := []map[string]any{}
				total := 101
				if calls == 1 {
					for i := 1; i <= 100; i++ {
						entries = append(entries, map[string]any{"id": i, "type": "branch", "name": fmt.Sprintf("release-%d", i)})
					}
					if mode == "missing-type" {
						delete(entries[0], "type")
					}
					if mode == "partial" {
						entries = entries[:1]
					}
					if mode == "duplicate-id" {
						entries[1]["id"] = 1
					}
					if mode == "duplicate-pair" {
						entries[1]["name"] = "release-1"
					}
				} else {
					entries = append(entries, map[string]any{"id": 101, "type": "tag", "name": "release-1"})
					if mode == "count-changed" {
						total = 102
					}
				}
				body, _ := json.Marshal(map[string]any{"total_count": total, "branch_policies": entries})
				return actionsResponse(r, 200, string(body)), nil
			}).rest}
			states, err := c.readPatterns(repository.Repository{Owner: "owner", Name: "current"}, "preview/test")
			if (err == nil) != (mode == "pages") {
				t.Fatal(states, err)
			}
			if mode == "pages" && (len(states) != 101 || calls != 2 || states[100].Pattern.Type != "tag") {
				t.Fatal(len(states), calls)
			}
		})
	}
}

func TestEnvironmentPatternUpsertSequenceOmissionAndPartialFailure(t *testing.T) {
	for _, mode := range []string{"custom", "enter", "omitted", "empty", "create303", "delete403", "readback-mismatch", "identity-race", "omitted-api-changes"} {
		t.Run(mode, func(t *testing.T) {
			patterns := map[int64]settings.EnvironmentPattern{7: {Type: "branch", Name: "old"}, 8: {Type: "tag", Name: "same"}}
			custom := mode != "enter"
			writes := []string{}
			state := func() map[string]any {
				var policy any
				if custom {
					policy = map[string]bool{"protected_branches": false, "custom_branch_policies": true}
				}
				rules := []any{map[string]any{"type": "wait_timer", "wait_timer": 30}}
				if mode == "omitted-api-changes" && len(writes) > 0 {
					rules = []any{}
				}
				return map[string]any{"id": 1, "name": "production", "protection_rules": rules, "deployment_branch_policy": policy, "can_admins_bypass": false}
			}
			c := &EnvironmentsClient{rest: actionsClient(t, func(r *http.Request) (*http.Response, error) {
				respond := func(body any) (*http.Response, error) {
					data, _ := json.Marshal(body)
					return actionsResponse(r, 200, string(data)), nil
				}
				base := "/repos/owner/current/environments/production"
				if r.Method == "GET" {
					switch r.URL.Path {
					case "/repos/owner/current":
						return actionsResponse(r, 200, `{"id":71,"full_name":"owner/current","archived":false}`), nil
					case "/repos/owner/current/environments":
						return actionsResponse(r, 200, `{"total_count":1,"environments":[{"id":1,"name":"production"}]}`), nil
					case base:
						return respond(state())
					case base + "/deployment_protection_rules":
						return actionsResponse(r, 200, customEnvironmentEmpty), nil
					case base + "/deployment-branch-policies":
						entries := []any{}
						for id, p := range patterns {
							entries = append(entries, map[string]any{"id": id, "type": p.Type, "name": p.Name})
						}
						return respond(map[string]any{"total_count": len(entries), "branch_policies": entries})
					case base + "/deployment-branch-policies/7":
						name := "old"
						if mode == "identity-race" {
							name = "concurrent"
						}
						return respond(map[string]any{"id": 7, "type": "branch", "name": name})
					case base + "/deployment-branch-policies/8":
						return respond(map[string]any{"id": 8, "type": "tag", "name": "same"})
					}
				}
				writes = append(writes, r.Method+" "+r.URL.Path)
				if r.Method == "PUT" && r.URL.Path == base {
					var payload map[string]json.RawMessage
					json.NewDecoder(r.Body).Decode(&payload)
					if mode == "omitted" || mode == "omitted-api-changes" {
						if len(payload) != 0 {
							t.Fatal("omission was supplemented", payload)
						}
					} else {
						if len(payload) != 1 || payload["deployment_branch_policy"] == nil {
							t.Fatal("patterns or omitted values entered PUT", payload)
						}
						custom = true
					}
					return respond(state())
				}
				if r.Method == "POST" && r.URL.Path == base+"/deployment-branch-policies" {
					var pattern settings.EnvironmentPattern
					json.NewDecoder(r.Body).Decode(&pattern)
					if pattern.Type != "branch" || pattern.Name != "same" {
						t.Fatal(pattern)
					}
					if mode == "create303" {
						return actionsResponse(r, 303, `{}`), nil
					}
					if mode != "readback-mismatch" {
						patterns[9] = pattern
					}
					return respond(map[string]any{"id": 9, "type": pattern.Type, "name": pattern.Name})
				}
				if r.Method == "DELETE" && (strings.HasSuffix(r.URL.Path, "/7") || strings.HasSuffix(r.URL.Path, "/8")) {
					if mode == "delete403" {
						return actionsResponse(r, 403, `{}`), nil
					}
					if strings.HasSuffix(r.URL.Path, "/7") {
						delete(patterns, 7)
					} else {
						delete(patterns, 8)
					}
					return actionsResponse(r, 204, ""), nil
				}
				t.Fatal("unexpected request", r.Method, r.URL)
				return nil, nil
			}).rest}
			desired := settings.EnvironmentDeclaration{Name: "production"}
			if mode != "omitted" && mode != "omitted-api-changes" {
				desired.DeploymentBranchPolicy = &settings.EnvironmentBranchPolicy{CustomBranchPolicies: true}
				list := []settings.EnvironmentPattern{{Type: "branch", Name: "same"}, {Type: "tag", Name: "same"}}
				if mode == "empty" {
					list = []settings.EnvironmentPattern{}
				}
				desired.DeploymentBranchPatterns = &list
			}
			_, err := c.Update(repository.Repository{Owner: "owner", Name: "current"}, desired)
			bad := mode == "create303" || mode == "delete403" || mode == "readback-mismatch" || mode == "identity-race"
			if (err != nil) != bad {
				t.Fatal(writes, err)
			}
			if mode == "omitted" || mode == "omitted-api-changes" {
				if !reflect.DeepEqual(writes, []string{"PUT /repos/owner/current/environments/production"}) || len(patterns) != 2 {
					t.Fatal(writes, patterns)
				}
			}
			if mode == "custom" || mode == "enter" {
				if len(writes) != 3 || !strings.HasPrefix(writes[1], "POST ") || !strings.HasSuffix(writes[2], "/7") {
					t.Fatal(writes)
				}
			}
			if mode == "identity-race" && len(patterns) != 3 {
				t.Fatal("earlier creation was rolled back", patterns)
			}
		})
	}
}
