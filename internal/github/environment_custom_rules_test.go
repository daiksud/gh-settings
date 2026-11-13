package github

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func appJSON(id int64) map[string]any {
	return map[string]any{"id": id, "node_id": fmt.Sprint("app", id), "slug": fmt.Sprint("custom-", id), "integration_url": fmt.Sprint("https://api.github.com/apps/custom-", id)}
}
func ruleJSON(id int64) map[string]any {
	return map[string]any{"id": id + 100, "node_id": fmt.Sprint("rule", id), "enabled": true, "app": appJSON(id)}
}

func TestEnvironmentCustomRuleReconciliationAndEffects(t *testing.T) {
	for _, mode := range []string{"add", "matching", "replace", "clear", "omitted", "raw-mode", "capacity", "unavailable", "wrong-ack", "wrong-status", "ignored", "readback-failed", "delete-failed", "replacement-drift", "parent-drift", "create-unavailable"} {
		t.Run(mode, func(t *testing.T) {
			repo := repository.Repository{Owner: "owner", Name: "current"}
			path := "/repos/owner/current/environments/production"
			rules := []map[string]any{ruleJSON(1)}
			want := []int64{1, 2}
			writes := []string{}
			availability := 0
			customReads := 0
			exists := true
			state := map[string]any{"id": 7, "name": "production", "protection_rules": []any{}, "deployment_branch_policy": nil, "can_admins_bypass": false}
			intent := settings.EnvironmentDeclaration{Name: "production", CustomRules: &want}
			switch mode {
			case "matching":
				want = []int64{1}
			case "replace", "delete-failed":
				want = []int64{2}
			case "clear":
				want = []int64{}
			case "omitted", "parent-drift":
				intent.CustomRules = nil
				timer := 5
				intent.WaitTimer = &timer
			case "raw-mode":
				state["deployment_branch_policy"] = map[string]bool{"protected_branches": false, "custom_branch_policies": false}
			case "capacity":
				rules = []map[string]any{}
				for id := int64(1); id <= 6; id++ {
					rules = append(rules, ruleJSON(id))
				}
				want = []int64{2, 3, 4, 5, 6, 7}
				timer := 5
				intent.WaitTimer = &timer
			case "create-unavailable":
				exists = false
				rules = []map[string]any{}
			}
			c := &EnvironmentsClient{rest: actionsClient(t, func(r *http.Request) (*http.Response, error) {
				respond := func(status int, value any) (*http.Response, error) {
					data, _ := json.Marshal(value)
					return actionsResponse(r, status, string(data)), nil
				}
				if r.Method == "GET" {
					switch r.URL.Path {
					case "/repos/owner/current":
						return actionsResponse(r, 200, `{"id":71,"full_name":"owner/current","archived":false}`), nil
					case "/repos/owner/current/environments":
						if !exists {
							return actionsResponse(r, 200, `{"total_count":0,"environments":[]}`), nil
						}
						return actionsResponse(r, 200, `{"total_count":1,"environments":[{"id":7,"name":"production"}]}`), nil
					case path:
						return respond(200, state)
					case path + "/deployment_protection_rules":
						customReads++
						if mode == "replacement-drift" && customReads >= 3 {
							rules[0]["id"] = int64(999)
						}
						return respond(200, map[string]any{"total_count": len(rules), "custom_deployment_protection_rules": rules})
					case path + "/deployment_protection_rules/apps":
						availability++
						if mode == "unavailable" || mode == "create-unavailable" {
							return actionsResponse(r, 403, `{}`), nil
						}
						return respond(200, map[string]any{"total_count": 1, "available_custom_deployment_protection_rule_integrations": []any{appJSON(2)}})
					case path + "/deployment_protection_rules/102":
						if mode == "readback-failed" {
							return actionsResponse(r, 403, `{}`), nil
						}
						return respond(200, ruleJSON(2))
					}
				}
				if r.Method == "PUT" && r.URL.Path == path {
					writes = append(writes, "parent")
					var payload map[string]any
					if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
						t.Fatal(err)
					}
					if _, ok := payload["custom_rules"]; ok {
						t.Fatal("App IDs leaked into parent payload", payload)
					}
					if timer, ok := payload["wait_timer"]; ok {
						state["protection_rules"] = []any{map[string]any{"type": "wait_timer", "wait_timer": timer}}
					}
					exists = true
					if mode == "parent-drift" {
						rules[0]["id"] = int64(999)
					}
					return respond(200, state)
				}
				if r.Method == "POST" && r.URL.Path == path+"/deployment_protection_rules" {
					writes = append(writes, "add")
					var payload map[string]int64
					if err := json.NewDecoder(r.Body).Decode(&payload); err != nil || !reflect.DeepEqual(payload, map[string]int64{"integration_id": 2}) {
						t.Fatal(payload, err)
					}
					if mode != "ignored" {
						rules = append(rules, ruleJSON(2))
					}
					ack := ruleJSON(2)
					if mode == "wrong-ack" {
						ack["app"] = appJSON(3)
					}
					status := 201
					if mode == "wrong-status" {
						status = 200
					}
					return respond(status, ack)
				}
				if r.Method == "DELETE" && r.URL.Path == path+"/deployment_protection_rules/101" {
					writes = append(writes, "remove")
					if mode == "delete-failed" {
						return actionsResponse(r, 403, `{}`), nil
					}
					rules = slices.DeleteFunc(rules, func(rule map[string]any) bool { return rule["id"] == int64(101) })
					return actionsResponse(r, 204, ""), nil
				}
				t.Fatal("unrelated endpoint", r.Method, r.URL)
				return nil, nil
			}).rest}
			wrote, err := c.Update(repo, intent)
			bad := slices.Contains([]string{"capacity", "unavailable", "wrong-ack", "wrong-status", "ignored", "readback-failed", "delete-failed", "replacement-drift", "parent-drift", "create-unavailable"}, mode)
			if (err != nil) != bad {
				t.Fatal(wrote, writes, err)
			}
			wantWrites := []string{"add"}
			switch mode {
			case "matching", "capacity", "unavailable", "replacement-drift":
				wantWrites = []string{}
			case "replace", "delete-failed":
				wantWrites = []string{"add", "remove"}
			case "clear":
				wantWrites = []string{"remove"}
			case "omitted", "parent-drift", "create-unavailable":
				wantWrites = []string{"parent"}
			}
			if !reflect.DeepEqual(writes, wantWrites) {
				t.Fatal(writes, wantWrites, err)
			}
			if (mode == "matching" || mode == "omitted" || mode == "clear" || mode == "parent-drift") && availability != 0 {
				t.Fatal("unneeded availability", availability)
			}
			if bad {
				var pre interface{ ReadFailedBeforeWrite() }
				var effects interface{ EnvironmentWritesCompleted() }
				if errors.As(err, &pre) != (len(writes) == 0) {
					t.Fatal("incorrect prewrite certainty", err)
				}
				if errors.As(err, &effects) != slices.Contains([]string{"delete-failed", "parent-drift", "create-unavailable"}, mode) {
					t.Fatal("incorrect completed effects", err)
				}
				if slices.Contains([]string{"wrong-ack", "wrong-status", "ignored", "readback-failed"}, mode) && !strings.Contains(err.Error(), "binding rule ID 102") {
					t.Fatal("lost recovery ID", err)
				}
			} else if wrote != (len(writes) > 0) {
				t.Fatal("wrong no-op evidence", wrote, writes)
			}
		})
	}
}

func TestEnvironmentAvailableAppsCompletePagination(t *testing.T) {
	for _, mode := range []string{"complete", "array", "changed-count", "duplicate", "short", "null", "late-denied"} {
		t.Run(mode, func(t *testing.T) {
			pages := 0
			c := &EnvironmentsClient{rest: actionsClient(t, func(r *http.Request) (*http.Response, error) {
				pages++
				if r.URL.Query().Get("per_page") != "100" || r.URL.Query().Get("page") != strconv.Itoa(pages) {
					t.Fatal(r.URL)
				}
				if mode == "array" {
					return actionsResponse(r, 200, `[]`), nil
				}
				if mode == "null" {
					return actionsResponse(r, 200, `{"total_count":0,"available_custom_deployment_protection_rule_integrations":null}`), nil
				}
				if mode == "late-denied" && pages == 2 {
					return actionsResponse(r, 403, `{}`), nil
				}
				apps := []any{}
				if pages == 1 {
					for id := int64(1); id <= 100; id++ {
						apps = append(apps, appJSON(id))
					}
					if mode == "short" {
						apps = apps[:99]
					}
				} else {
					id := int64(101)
					if mode == "duplicate" {
						id = 1
					}
					apps = append(apps, appJSON(id))
				}
				total := 101
				if mode == "changed-count" && pages == 2 {
					total = 102
				}
				data, _ := json.Marshal(map[string]any{"total_count": total, "available_custom_deployment_protection_rule_integrations": apps})
				return actionsResponse(r, 200, string(data)), nil
			}).rest}
			got, err := c.availableCustomApps(repository.Repository{Owner: "owner", Name: "current"}, "production")
			if (err == nil) != (mode == "complete") || (err == nil && len(got) != 101) || (err != nil && got != nil) {
				t.Fatal(len(got), pages, err)
			}
		})
	}
}

func TestEnvironmentCustomRuleReadsRejectIncompleteOrUnknownState(t *testing.T) {
	for _, mode := range []string{"complete", "missing", "null", "count", "duplicate", "disabled", "unknown", "missing-app", "missing-node", "array", "status"} {
		t.Run(mode, func(t *testing.T) {
			rule := ruleJSON(1)
			entries := []any{rule}
			count := 1
			status := 200
			switch mode {
			case "duplicate":
				entries = append(entries, rule)
				count = 2
			case "count":
				count = 2
			case "disabled":
				rule["enabled"] = false
			case "unknown":
				rule["secret"] = "PRIVATE_VALUE_CANARY"
			case "missing-app":
				delete(rule, "app")
			case "missing-node":
				delete(rule, "node_id")
			case "status":
				status = 202
			}
			body := map[string]any{"total_count": count, "custom_deployment_protection_rules": entries}
			if mode == "null" {
				body["custom_deployment_protection_rules"] = nil
			}
			if mode == "missing" {
				delete(body, "custom_deployment_protection_rules")
			}
			data, _ := json.Marshal(body)
			if mode == "array" {
				data = []byte(`[]`)
			}
			c := &EnvironmentsClient{rest: actionsClient(t, func(r *http.Request) (*http.Response, error) { return actionsResponse(r, status, string(data)), nil }).rest}
			got, err := c.readCustomRules(repository.Repository{Owner: "owner", Name: "current"}, "production")
			if (err == nil) != (mode == "complete") || (err != nil && got != nil) {
				t.Fatal(got, err)
			}
			if err != nil && strings.Contains(err.Error(), "PRIVATE_VALUE_CANARY") {
				t.Fatal("raw evidence leaked", err)
			}
		})
	}
}

func TestEnvironmentPruneCustomGateRemainsIndependent(t *testing.T) {
	for _, late := range []bool{false, true} {
		t.Run(fmt.Sprint(late), func(t *testing.T) {
			reads := 0
			children := 0
			c := &EnvironmentsClient{rest: actionsClient(t, func(r *http.Request) (*http.Response, error) {
				if r.Method != "GET" {
					t.Fatal("prune preflight mutated", r.Method)
				}
				switch r.URL.Path {
				case "/repos/owner/current":
					return actionsResponse(r, 200, `{"id":71,"full_name":"owner/current","archived":false}`), nil
				case "/repos/owner/current/environments":
					return actionsResponse(r, 200, `{"total_count":1,"environments":[{"id":1,"name":"production"}]}`), nil
				case "/repos/owner/current/environments/production":
					return actionsResponse(r, 200, emptyEnvironmentFixture), nil
				case "/repos/owner/current/environments/production/deployment_protection_rules":
					reads++
					if late && reads == 1 {
						return actionsResponse(r, 200, customEnvironmentEmpty), nil
					}
					data, _ := json.Marshal(map[string]any{"total_count": 1, "custom_deployment_protection_rules": []any{ruleJSON(1)}})
					return actionsResponse(r, 200, string(data)), nil
				case "/repos/owner/current/environments/production/variables":
					children++
					return actionsResponse(r, 200, `{"total_count":0,"variables":[]}`), nil
				case "/repos/owner/current/environments/production/secrets":
					children++
					return actionsResponse(r, 200, `{"total_count":0,"secrets":[]}`), nil
				default:
					t.Fatal(r.URL)
					return nil, nil
				}
			}).rest}
			if _, err := c.ReadEnvironmentPrune(repository.Repository{Owner: "owner", Name: "current"}, nil); err == nil {
				t.Fatal("custom protections admitted to parent deletion")
			}
			if !late && children != 0 {
				t.Fatal("nonempty custom gate did not stop acquisition")
			}
		})
	}
}

func TestEnvironmentCustomRulesPreserveRawPatternsAcrossStages(t *testing.T) {
	for _, mode := range []string{"app-preserves-unknown-pattern", "app-changes-pattern", "patterns-preserve-app", "pattern-stage-app-drift"} {
		t.Run(mode, func(t *testing.T) {
			path := "/repos/owner/current/environments/production"
			rules := []any{ruleJSON(1)}
			patterns := []any{map[string]any{"id": 41, "name": "main", "type": "future"}}
			ids := []int64{1, 2}
			desired := settings.EnvironmentDeclaration{Name: "production", CustomRules: &ids}
			patternOnly := strings.HasPrefix(mode, "pattern")
			if patternOnly {
				patterns = []any{}
				desired.CustomRules = nil
				p := []settings.EnvironmentPattern{{Type: "branch", Name: "main"}, {Type: "tag", Name: "v*"}}
				desired.DeploymentBranchPatterns = &p
			}
			writes := []string{}
			c := &EnvironmentsClient{rest: actionsClient(t, func(r *http.Request) (*http.Response, error) {
				respond := func(status int, body any) (*http.Response, error) {
					data, _ := json.Marshal(body)
					return actionsResponse(r, status, string(data)), nil
				}
				if r.Method == "GET" {
					switch r.URL.Path {
					case "/repos/owner/current":
						return actionsResponse(r, 200, `{"id":71,"full_name":"owner/current","archived":false}`), nil
					case "/repos/owner/current/environments":
						return actionsResponse(r, 200, `{"total_count":1,"environments":[{"id":1,"name":"production"}]}`), nil
					case path:
						return actionsResponse(r, 200, `{"id":1,"name":"production","protection_rules":[],"deployment_branch_policy":{"protected_branches":false,"custom_branch_policies":true},"can_admins_bypass":false}`), nil
					case path + "/deployment_protection_rules":
						return respond(200, map[string]any{"total_count": len(rules), "custom_deployment_protection_rules": rules})
					case path + "/deployment_protection_rules/apps":
						return respond(200, map[string]any{"total_count": 1, "available_custom_deployment_protection_rule_integrations": []any{appJSON(2)}})
					case path + "/deployment_protection_rules/102":
						return respond(200, ruleJSON(2))
					case path + "/deployment-branch-policies":
						return respond(200, map[string]any{"total_count": len(patterns), "branch_policies": patterns})
					}
				}
				if r.Method == "POST" && r.URL.Path == path+"/deployment_protection_rules" {
					writes = append(writes, "app")
					rules = append(rules, ruleJSON(2))
					if mode == "app-changes-pattern" {
						patterns[0].(map[string]any)["id"] = 42
					}
					return respond(201, ruleJSON(2))
				}
				if r.Method == "POST" && r.URL.Path == path+"/deployment-branch-policies" {
					writes = append(writes, "pattern")
					var body map[string]any
					if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
						t.Fatal(err)
					}
					body["id"] = len(patterns) + 1
					patterns = append(patterns, body)
					if mode == "pattern-stage-app-drift" {
						rules[0].(map[string]any)["id"] = 999
					}
					return respond(200, body)
				}
				t.Fatal("unrelated mutation", r.Method, r.URL)
				return nil, nil
			}).rest}
			changed, err := c.Update(repository.Repository{Owner: "owner", Name: "current"}, desired)
			bad := mode == "app-changes-pattern" || mode == "pattern-stage-app-drift"
			if (err != nil) != bad || (!bad && !changed) {
				t.Fatal(writes, changed, err)
			}
			want := "app"
			if patternOnly {
				want = "pattern,pattern"
			}
			if mode == "pattern-stage-app-drift" {
				want = "pattern"
			}
			if strings.Join(writes, ",") != want {
				t.Fatal(writes, want, err)
			}
		})
	}
}
