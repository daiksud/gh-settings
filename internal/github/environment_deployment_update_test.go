package github

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func TestNormalizedEnvironmentDeploymentUpdateUsesLiteralIntentAndRawConfirmation(t *testing.T) {
	for _, raw := range []bool{false, true} {
		for _, test := range []struct {
			name, mode, post                  string
			patterns, nameOnly, sink, wantErr bool
			wantWarnings                      int
		}{
			{"no restriction", "none", "canonical", false, false, true, false, 1},
			{"protected", "protected", "canonical", false, false, true, false, 1},
			{"selected", "selected", "canonical", false, false, true, false, 1},
			{"selected pattern CRUD", "selected", "canonical", true, false, true, false, 1},
			{"nil sink", "none", "canonical", false, false, false, false, 0},
			{"retained raw", "none", "ignored", false, false, true, true, 2},
			{"bad acknowledgment", "none", "bad-ack", false, false, true, true, 1},
			{"readback denied", "none", "read-denied", false, false, true, true, 1},
			{"readback malformed", "none", "read-malformed", false, false, true, true, 1},
			{"readback wrong name", "none", "wrong-name", false, false, true, true, 1},
			{"fresh custom denied", "none", "custom-denied", false, false, true, true, 1},
			{"fresh latent denied", "none", "latent-denied", false, false, true, true, 1},
			{"preflight denied", "none", "pre-denied", false, false, true, true, 0},
			{"preflight latent denied", "none", "pre-latent-denied", false, false, true, true, 0},
			{"repository identity race", "none", "identity-race", false, false, true, true, 0},
			{"patterns no explicit mode", "omit", "canonical", true, false, true, true, 0},
			{"pattern mode ignored", "selected", "ignored", true, false, true, true, 2},
			{"pattern create failure", "selected", "create-denied", true, false, true, true, 1},
			{"pattern delete failure", "selected", "delete-denied", true, false, true, true, 1},
			{"omitted mode to FF", "omit", "ff", false, false, true, false, 0},
			{"omitted mode to TT", "omit", "tt", false, false, true, false, 0},
			{"omitted mode to null", "omit", "null", false, false, true, false, 0},
			{"omitted mode to protected", "omit", "protected", false, false, true, false, 0},
			{"omitted mode to selected", "omit", "selected", false, false, true, false, 0},
			{"name-only API outcomes", "omit", "tt", false, true, true, false, 0},
		} {
			if !raw && (test.post == "latent-denied" || test.post == "pre-latent-denied") {
				continue
			}
			t.Run(fmt.Sprintf("%t/%s", raw, test.name), func(t *testing.T) {
				rawPair := map[string]any{"protected_branches": raw, "custom_branch_policies": raw}
				var actualMode any = rawPair
				patterns := map[int64]settings.EnvironmentPattern{7: {Type: "branch", Name: "old"}}
				var writes []string
				repoReads, postDetailReads := 0, 0
				postStarted := false
				type warning struct {
					name   string
					record settings.EnvironmentDeploymentNormalization
				}
				var warnings []warning
				intendedMode := any(nil)
				if test.mode == "protected" {
					intendedMode = map[string]any{"protected_branches": true, "custom_branch_policies": false}
				}
				if test.mode == "selected" {
					intendedMode = map[string]any{"protected_branches": false, "custom_branch_policies": true}
				}
				intendedPayload := map[string]any{}
				if !test.nameOnly {
					intendedPayload = map[string]any{"wait_timer": float64(0), "prevent_self_review": false, "can_admins_bypass": false, "reviewers": []any{}}
				}
				if test.mode != "omit" {
					intendedPayload["deployment_branch_policy"] = intendedMode
				}
				detail := func(mode any) map[string]any {
					rules := []any{}
					bypass := false
					if test.nameOnly && postStarted {
						rules = []any{map[string]any{"type": "wait_timer", "wait_timer": 30}}
						bypass = true
					}
					return map[string]any{"id": 1, "name": "production", "protection_rules": rules, "deployment_branch_policy": mode, "can_admins_bypass": bypass}
				}
				respond := func(req *http.Request, status int, body any) (*http.Response, error) {
					data, _ := json.Marshal(body)
					return actionsResponse(req, status, string(data)), nil
				}
				client := &EnvironmentsClient{rest: actionsClient(t, func(req *http.Request) (*http.Response, error) {
					base := "/repos/owner/current/environments/production"
					if req.URL.Path == "/repos/owner/current" {
						repoReads++
						id := 71
						if test.post == "identity-race" && repoReads == 2 {
							id = 72
						}
						return respond(req, 200, map[string]any{"id": id, "full_name": "owner/current", "archived": false})
					}
					if req.URL.Path == "/repos/owner/current/environments" {
						return actionsResponse(req, 200, `{"total_count":1,"environments":[{"id":1,"name":"production"}]}`), nil
					}
					if req.URL.Path == base {
						if req.Method == "GET" {
							if postStarted {
								postDetailReads++
								if test.post == "read-denied" {
									return actionsResponse(req, 403, `{}`), nil
								}
							}
							if !postStarted && test.post == "pre-denied" {
								return actionsResponse(req, 403, `{}`), nil
							}
							body := detail(actualMode)
							if postStarted && test.post == "read-malformed" {
								body["can_admins_bypass"] = nil
							}
							if postStarted && test.post == "wrong-name" {
								body["name"] = "other"
							}
							return respond(req, 200, body)
						}
						if req.Method != "PUT" {
							t.Fatal("unexpected lifecycle write", req.Method)
						}
						var payload map[string]any
						if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
							t.Fatal(err)
						}
						if !reflect.DeepEqual(payload, intendedPayload) {
							t.Fatal("PUT differs from independent literal intent; copied/defaulted omitted field", payload, intendedPayload)
						}
						if test.sink && test.mode != "omit" && len(warnings) != 1 {
							t.Fatal("prewrite provenance missing", warnings)
						}
						writes = append(writes, "PUT environment")
						postStarted = true
						switch test.post {
						case "ignored", "custom-denied", "latent-denied": // Retain the original raw pair while acknowledging canonical intent.
						case "ff":
							actualMode = map[string]any{"protected_branches": false, "custom_branch_policies": false}
						case "tt":
							actualMode = map[string]any{"protected_branches": true, "custom_branch_policies": true}
						case "null":
							actualMode = nil
						case "protected":
							actualMode = map[string]any{"protected_branches": true, "custom_branch_policies": false}
						case "selected":
							actualMode = map[string]any{"protected_branches": false, "custom_branch_policies": true}
						default:
							actualMode = intendedMode
						}
						ackMode := actualMode
						if test.mode != "omit" {
							ackMode = intendedMode
						}
						if test.post == "bad-ack" {
							ackMode = rawPair
						}
						return respond(req, 200, detail(ackMode))
					}
					if req.URL.Path == base+"/deployment_protection_rules" {
						if postStarted && test.post == "custom-denied" {
							return actionsResponse(req, 403, `{}`), nil
						}
						return actionsResponse(req, 200, customEnvironmentEmpty), nil
					}
					if req.URL.Path == base+"/deployment-branch-policies" {
						if req.Method == "GET" {
							if !postStarted && test.post == "pre-latent-denied" || postStarted && test.post == "latent-denied" {
								return actionsResponse(req, 403, `{}`), nil
							}
							mode, ok := actualMode.(map[string]any)
							if !ok || mode["custom_branch_policies"] != true {
								t.Fatal("patterns acquired from unsupported applicability", actualMode)
							}
							entries := []any{}
							ids := []int{}
							for id := range patterns {
								ids = append(ids, int(id))
							}
							sort.Ints(ids)
							for _, id := range ids {
								p := patterns[int64(id)]
								entries = append(entries, map[string]any{"id": id, "type": p.Type, "name": p.Name})
							}
							return respond(req, 200, map[string]any{"total_count": len(entries), "branch_policies": entries})
						}
						if req.Method != "POST" {
							t.Fatal(req.Method)
						}
						writes = append(writes, "POST pattern")
						var payload map[string]any
						json.NewDecoder(req.Body).Decode(&payload)
						if !reflect.DeepEqual(payload, map[string]any{"type": "branch", "name": "wanted"}) {
							t.Fatal("pattern differs from literal intent", payload)
						}
						if test.post == "create-denied" {
							return actionsResponse(req, 403, `{}`), nil
						}
						patterns[8] = settings.EnvironmentPattern{Type: "branch", Name: "wanted"}
						return actionsResponse(req, 200, `{"id":8,"type":"branch","name":"wanted"}`), nil
					}
					if req.URL.Path == base+"/deployment-branch-policies/7" {
						if req.Method == "GET" {
							return actionsResponse(req, 200, `{"id":7,"type":"branch","name":"old"}`), nil
						}
						if req.Method != "DELETE" {
							t.Fatal(req.Method)
						}
						writes = append(writes, "DELETE pattern")
						if test.post == "delete-denied" {
							return actionsResponse(req, 403, `{}`), nil
						}
						delete(patterns, 7)
						return actionsResponse(req, 204, ""), nil
					}
					t.Fatal("unexpected continuation", req.Method, req.URL)
					return nil, nil
				}).rest}
				if test.sink {
					client.NormalizationWarning = func(name string, record settings.EnvironmentDeploymentNormalization) {
						warnings = append(warnings, warning{name, record})
					}
				}
				zero, off, reviewers := 0, false, []settings.EnvironmentReviewer{}
				desired := settings.EnvironmentDeclaration{Name: "PRODUCTION"}
				if !test.nameOnly {
					desired.WaitTimer, desired.PreventSelfReview, desired.CanAdminsBypass, desired.Reviewers = &zero, &off, &off, &reviewers
				}
				if test.mode != "omit" {
					desired.DeploymentBranchPolicyDeclared = true
					if test.mode == "protected" {
						desired.DeploymentBranchPolicy = &settings.EnvironmentBranchPolicy{ProtectedBranches: true}
					}
					if test.mode == "selected" {
						desired.DeploymentBranchPolicy = &settings.EnvironmentBranchPolicy{CustomBranchPolicies: true}
					}
				}
				if test.patterns {
					desired.DeploymentBranchPatterns = &[]settings.EnvironmentPattern{{Type: "branch", Name: "wanted"}}
				}
				changed, err := client.Update(repository.Repository{Owner: "owner", Name: "current"}, desired)
				if (err != nil) != test.wantErr || changed == test.wantErr {
					t.Fatal("certainty/outcome", changed, err, writes)
				}
				expectedWrites := []string{"PUT environment"}
				if test.post == "pre-denied" || test.post == "pre-latent-denied" || test.post == "identity-race" || test.mode == "omit" && test.patterns {
					expectedWrites = nil
				}
				if test.patterns && test.mode == "selected" && test.post != "ignored" {
					expectedWrites = append(expectedWrites, "POST pattern")
					if test.post != "create-denied" {
						expectedWrites = append(expectedWrites, "DELETE pattern")
					}
				}
				if !reflect.DeepEqual(writes, expectedWrites) || len(warnings) != test.wantWarnings {
					t.Fatal("write scope/order or callback count", writes, expectedWrites, warnings, test.wantWarnings, err)
				}
				wantRecord := settings.EnvironmentDeploymentNormalization{Path: settings.EnvironmentDeploymentModePath, RawPolicy: settings.EnvironmentBranchPolicy{ProtectedBranches: raw, CustomBranchPolicies: raw}, Fallback: settings.DefaultEnvironmentDeploymentChoice}
				for _, w := range warnings {
					if w.name != "production" || w.record != wantRecord {
						t.Fatal("warning context or raw pair changed", warnings)
					}
				}
				if err != nil && len(writes) > 0 && !strings.Contains(err.Error(), "write may have applied") && !strings.Contains(err.Error(), "earlier writes remain applied") {
					t.Fatal("partial effects hidden", err)
				}
				if test.post == "bad-ack" && postDetailReads != 0 {
					t.Fatal("bad acknowledgment continued to fresh read", postDetailReads)
				}
				if test.post == "create-denied" && len(patterns) != 1 || test.post == "delete-denied" && len(patterns) != 2 {
					t.Fatal("partial CRUD was rolled back or continued", patterns)
				}
				if !test.wantErr && test.mode != "omit" {
					oldWrites, oldWarnings := len(writes), len(warnings)
					repeated, err := client.Update(repository.Repository{Owner: "owner", Name: "current"}, desired)
					if err != nil || repeated || len(writes) != oldWrites || len(warnings) != oldWarnings {
						t.Fatal("canonical exact repeat not a no-op", repeated, err, writes, warnings)
					}
				}
			})
		}
	}
}
