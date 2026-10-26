package github

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func TestEnvironmentUpsertWritesWholePolicyAndVerifiesWithoutNumericBinding(t *testing.T) {
	repo := repository.Repository{Owner: "owner", Name: "current"}
	for _, mode := range []string{"create", "update", "matching", "unsupported", "write422", "ignored", "readback403", "changed-incarnation"} {
		t.Run(mode, func(t *testing.T) {
			writes := 0
			exists := mode != "create"
			state := map[string]any{"id": 1, "name": "production", "protection_rules": []any{}, "deployment_branch_policy": nil}
			if mode != "matching" {
				state["protection_rules"] = []any{map[string]any{"type": "wait_timer", "wait_timer": 30}}
			}
			c := &EnvironmentsClient{rest: actionsClient(t, func(r *http.Request) (*http.Response, error) {
				respond := func(body any) (*http.Response, error) {
					data, err := json.Marshal(body)
					if err != nil {
						t.Fatal(err)
					}
					return actionsResponse(r, 200, string(data)), nil
				}
				if r.Method == "GET" {
					switch r.URL.Path {
					case "/repos/owner/current":
						return actionsResponse(r, 200, `{"id":71,"full_name":"owner/current","archived":false}`), nil
					case "/repos/owner/current/environments":
						if !exists {
							return actionsResponse(r, 200, `{"total_count":0,"environments":[]}`), nil
						}
						return actionsResponse(r, 200, `{"total_count":1,"environments":[{"id":1,"name":"production"}]}`), nil
					case "/repos/owner/current/environments/production":
						if mode == "readback403" && writes > 0 {
							return actionsResponse(r, 403, `{}`), nil
						}
						return respond(state)
					case "/repos/owner/current/environments/production/deployment_protection_rules":
						if mode == "unsupported" {
							return actionsResponse(r, 200, `{"total_count":1,"custom_deployment_protection_rules":[{"id":9}]}`), nil
						}
						return actionsResponse(r, 200, customEnvironmentEmpty), nil
					}
				}
				if r.Method != "PUT" || r.URL.Path != "/repos/owner/current/environments/production" {
					t.Fatal(r.Method, r.URL)
				}
				writes++
				var payload map[string]any
				if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
					t.Fatal(err)
				}
				want := map[string]any{"wait_timer": float64(0), "prevent_self_review": false, "reviewers": []any{}, "deployment_branch_policy": nil}
				if !reflect.DeepEqual(payload, want) {
					t.Fatal("incomplete or identity-bearing PUT", payload)
				}
				if mode == "write422" {
					return actionsResponse(r, 422, `{"message":"rejected"}`), nil
				}
				if mode != "ignored" {
					state["protection_rules"] = []any{}
				}
				if mode == "changed-incarnation" {
					state["id"] = 2
				}
				exists = true
				return respond(state)
			}).rest}
			changed, err := c.Update(repo, settings.EnvironmentDeclaration{Name: "production"})
			bad := mode == "unsupported" || mode == "write422" || mode == "ignored" || mode == "readback403"
			if (err != nil) != bad {
				t.Fatal(changed, err)
			}
			if mode == "unsupported" || mode == "matching" {
				if writes != 0 || changed {
					t.Fatal("preflight wrote", writes, changed)
				}
			} else if writes != 1 {
				t.Fatal("unexpected write count", writes)
			}
			if bad {
				var pre interface{ ReadFailedBeforeWrite() }
				if errors.As(err, &pre) != (mode == "unsupported") {
					t.Fatal("write certainty misclassified", err)
				}
			}
		})
	}
}

const emptyEnvironmentFixture = `{"id":1,"name":"production","protection_rules":[],"deployment_branch_policy":null}`
const customEnvironmentEmpty = `{"total_count":0,"custom_deployment_protection_rules":[]}`

func TestEnvironmentCompleteDisabledAndConfiguredProtections(t *testing.T) {
	for _, configured := range []bool{false, true} {
		var raw map[string]json.RawMessage
		if err := json.Unmarshal([]byte(emptyEnvironmentFixture), &raw); err != nil {
			t.Fatal(err)
		}
		if configured {
			raw["protection_rules"] = json.RawMessage(`[{"id":7,"type":"wait_timer","wait_timer":30},{"type":"required_reviewers","prevent_self_review":true,"reviewers":[{"type":"User","reviewer":{"id":9,"login":"reviewer","avatar_url":"https://example.invalid/image"}}]},{"type":"branch_policy"}]`)
			raw["deployment_branch_policy"] = json.RawMessage(`{"protected_branches":true,"custom_branch_policies":false}`)
		}
		state, err := decodeEnvironment(raw)
		if err != nil {
			t.Fatal(err)
		}
		p := state.Policy
		if configured {
			if *p.WaitTimer != 30 || !*p.PreventSelfReview || len(*p.Reviewers) != 1 || (*p.Reviewers)[0].ID != 9 || p.DeploymentBranchPolicy == nil {
				t.Fatal("lost protection", p)
			}
		} else if *p.WaitTimer != 0 || *p.PreventSelfReview || len(*p.Reviewers) != 0 || p.DeploymentBranchPolicy != nil {
			t.Fatal("disabled rules not represented", p)
		}
	}
}

func TestEnvironmentMissingAndUnknownAreNotDisabled(t *testing.T) {
	for _, key := range []string{"id", "name", "protection_rules", "deployment_branch_policy"} {
		for _, null := range []bool{true, false} {
			if key == "deployment_branch_policy" && null {
				continue
			}
			var raw map[string]json.RawMessage
			json.Unmarshal([]byte(emptyEnvironmentFixture), &raw)
			delete(raw, key)
			if null {
				raw[key] = json.RawMessage("null")
			}
			if _, err := decodeEnvironment(raw); err == nil {
				t.Fatal("accepted missing/null", key, null)
			}
		}
	}
	for _, rules := range []string{
		`[{"type":"wait_timer"}]`, `[{"type":"wait_timer","wait_timer":null}]`,
		`[{"type":"wait_timer","wait_timer":0},{"type":"wait_timer","wait_timer":1}]`,
		`[{"type":"required_reviewers","reviewers":[]}]`,
		`[{"type":"required_reviewers","prevent_self_review":false,"reviewers":null}]`,
		`[{"type":"required_reviewers","prevent_self_review":false,"reviewers":[{"type":"User","reviewer":{}}]}]`,
		`[{"type":"required_reviewers","prevent_self_review":false,"reviewers":[{"type":"User","reviewer":{"id":7,"type":"Organization"}}]}]`,
		`[{"type":"custom"}]`, `[{"type":"branch_policy","unknown":true}]`,
	} {
		var raw map[string]json.RawMessage
		json.Unmarshal([]byte(emptyEnvironmentFixture), &raw)
		raw["protection_rules"] = json.RawMessage(rules)
		if _, err := decodeEnvironment(raw); err == nil {
			t.Fatal("unsafe protection accepted", rules)
		}
	}
	for _, field := range []string{"can_admins_bypass", "future_setting"} {
		var raw map[string]json.RawMessage
		json.Unmarshal([]byte(emptyEnvironmentFixture), &raw)
		raw[field] = json.RawMessage("false")
		if _, err := decodeEnvironment(raw); err == nil {
			t.Fatal("unknown preservation assumed", field)
		}
	}
}

func TestEnvironmentSelectedReadsNeverAdoptUnavailableState(t *testing.T) {
	repo := repository.Repository{Owner: "owner", Name: "current"}
	for _, failure := range []string{"", "list403", "listNull", "partial", "detail404", "custom403", "customMissing", "customNull", "customEnabled"} {
		t.Run(failure, func(t *testing.T) {
			calls := 0
			c := &EnvironmentsClient{rest: actionsClient(t, func(r *http.Request) (*http.Response, error) {
				calls++
				if r.Method != "GET" {
					t.Fatal("mutated during read", r.Method)
				}
				switch r.URL.Path {
				case "/repos/owner/current/environments":
					if failure == "list403" {
						return actionsResponse(r, 403, `{}`), nil
					}
					if failure == "listNull" {
						return actionsResponse(r, 200, `{"total_count":0,"environments":null}`), nil
					}
					count := 2
					if failure == "partial" {
						count = 3
					}
					return actionsResponse(r, 200, fmt.Sprintf(`{"total_count":%d,"environments":[{"id":1,"name":"production"},{"id":2,"name":"unmanaged"}]}`, count)), nil
				case "/repos/owner/current/environments/production":
					if failure == "detail404" {
						return actionsResponse(r, 404, `{}`), nil
					}
					return actionsResponse(r, 200, emptyEnvironmentFixture), nil
				case "/repos/owner/current/environments/production/deployment_protection_rules":
					if failure == "custom403" {
						return actionsResponse(r, 403, `{}`), nil
					}
					body := customEnvironmentEmpty
					if failure == "customMissing" {
						body = `{}`
					}
					if failure == "customNull" {
						body = `{"total_count":0,"custom_deployment_protection_rules":null}`
					}
					if failure == "customEnabled" {
						body = `{"total_count":1,"custom_deployment_protection_rules":[{"id":8,"enabled":true}]}`
					}
					return actionsResponse(r, 200, body), nil
				default:
					t.Fatal("read unmanaged details or unrelated endpoint", r.URL)
					return nil, nil
				}
			}).rest}
			observation, err := c.Read(repo, []string{"PRODUCTION"})
			if (err == nil) != (failure == "") {
				t.Fatal(failure, err)
			}
			values, valueErr := observation.Values()
			if failure == "" {
				if valueErr != nil || len(values) != 1 || values[0].Policy.Name != "production" {
					t.Fatal(values, valueErr)
				}
			} else if valueErr == nil {
				t.Fatal("failed read acquired state")
			}
			before := calls
			if _, err := c.Read(repo, []string{}); err != nil || calls != before {
				t.Fatal("empty selection read remote", err)
			}
		})
	}
}

func TestEnvironmentPathEscapesNameAsOneSegment(t *testing.T) {
	repo := repository.Repository{Owner: "owner", Name: "current"}
	path := environmentPath(repo, "preview/a%2Fb")
	if !strings.HasSuffix(path, "/preview%2Fa%252Fb") {
		t.Fatal(path)
	}
}
