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

// This fixture intentionally replaces every built-in on PUT. Omitted request
// members reset instead of inheriting state, exposing accidental partial sends.
type preservationEnvironment struct {
	timer           int
	prevent, bypass bool
	reviewers       []settings.EnvironmentReviewer
	mode            *settings.EnvironmentBranchPolicy
	patterns        map[int64]settings.EnvironmentPattern
}

func (s preservationEnvironment) detail() map[string]any {
	rules := []any{}
	if s.timer != 0 {
		rules = append(rules, map[string]any{"type": "wait_timer", "wait_timer": s.timer})
	}
	if len(s.reviewers) > 0 {
		reviewers := []any{}
		for _, r := range s.reviewers {
			reviewers = append(reviewers, map[string]any{"type": r.Type, "reviewer": map[string]any{"id": r.ID, "type": r.Type}})
		}
		rules = append(rules, map[string]any{"type": "required_reviewers", "prevent_self_review": s.prevent, "reviewers": reviewers})
	}
	return map[string]any{"id": 1, "name": "production", "protection_rules": rules, "can_admins_bypass": s.bypass, "deployment_branch_policy": s.mode}
}
func (s preservationEnvironment) payload() map[string]any {
	return map[string]any{"wait_timer": s.timer, "prevent_self_review": s.prevent, "can_admins_bypass": s.bypass, "reviewers": s.reviewers, "deployment_branch_policy": s.mode}
}
func TestEnvironmentPartialUpdatePreservesLatestCompanions(t *testing.T) {
	for _, control := range []string{"timer", "self-review", "bypass", "reviewers", "mode", "timer-with-empty-controls"} {
		t.Run(control, func(t *testing.T) {
			state := preservationEnvironment{30, true, true, []settings.EnvironmentReviewer{{Type: "User", ID: 7}}, &settings.EnvironmentBranchPolicy{ProtectedBranches: true}, nil}
			want := state
			requested := settings.EnvironmentDeclaration{Name: "PRODUCTION"}
			zero, off := 0, false
			switch control {
			case "timer":
				requested.WaitTimer = &zero
				want.timer = 0
			case "self-review":
				requested.PreventSelfReview = &off
				want.prevent = false
			case "bypass":
				requested.CanAdminsBypass = &off
				want.bypass = false
			case "reviewers":
				values := []settings.EnvironmentReviewer{{Type: "Team", ID: 9}}
				requested.Reviewers = &values
				want.reviewers = values
			case "mode":
				requested.DeploymentBranchPolicyDeclared = true
				want.mode = nil
			case "timer-with-empty-controls":
				state.prevent = false
				state.bypass = false
				state.reviewers = []settings.EnvironmentReviewer{}
				state.mode = nil
				want = state
				want.timer = 0
				requested.WaitTimer = &zero
			}
			writes, reads := 0, 0
			client := &EnvironmentsClient{rest: actionsClient(t, func(req *http.Request) (*http.Response, error) {
				respond := func(v any) (*http.Response, error) {
					data, _ := json.Marshal(v)
					return actionsResponse(req, 200, string(data)), nil
				}
				base := "/repos/owner/current/environments/production"
				switch req.URL.Path {
				case "/repos/owner/current":
					return actionsResponse(req, 200, `{"id":71,"full_name":"owner/current","archived":false}`), nil
				case "/repos/owner/current/environments":
					return actionsResponse(req, 200, `{"total_count":1,"environments":[{"id":1,"name":"production"}]}`), nil
				case base + "/deployment_protection_rules":
					return actionsResponse(req, 200, customEnvironmentEmpty), nil
				case base:
					if req.Method == "GET" {
						reads++
						return respond(state.detail())
					}
					if req.Method != "PUT" {
						t.Fatal(req.Method)
					}
					var got map[string]any
					json.NewDecoder(req.Body).Decode(&got)
					data, _ := json.Marshal(want.payload())
					var expected map[string]any
					json.Unmarshal(data, &expected)
					if !reflect.DeepEqual(got, expected) {
						t.Fatalf("partial intent lost companions: got %v want %v", got, expected)
					}
					var sent settings.EnvironmentDeclaration
					data, _ = json.Marshal(got)
					json.Unmarshal(data, &sent)
					state = preservationEnvironment{*sent.WaitTimer, *sent.PreventSelfReview, *sent.CanAdminsBypass, *sent.Reviewers, sent.DeploymentBranchPolicy, nil}
					writes++
					return respond(state.detail())
				}
				t.Fatal(req.Method, req.URL)
				return nil, nil
			}).rest}
			repo := repository.Repository{Owner: "owner", Name: "current"}
			// An earlier plan must not own companions from its snapshot.
			if _, err := client.Read(repo, []string{"production"}); err != nil {
				t.Fatal(err)
			}
			if control != "reviewers" && control != "timer-with-empty-controls" {
				state.reviewers = []settings.EnvironmentReviewer{{Type: "User", ID: 11}}
				want.reviewers = state.reviewers
			}
			changed, err := client.Update(repo, requested)
			if err != nil || !changed || writes != 1 || reads != 3 || !reflect.DeepEqual(state, want) {
				t.Fatal(changed, writes, reads, state, want, err)
			}
			changed, err = client.Update(repo, requested)
			if err != nil || changed || writes != 1 || reads != 4 {
				t.Fatal("repeat mutated", changed, writes, reads, err)
			}
			if requested.Name != "PRODUCTION" || requested.DeploymentBranchPatterns != nil {
				t.Fatal("intent mutated", requested)
			}
		})
	}
}

func TestEnvironmentPreservationRejectsCompanionDriftAndRetainsRawPatterns(t *testing.T) {
	for _, corruption := range []string{"none", "timer", "self-review", "reviewers", "mode", "pattern-id", "pattern-type", "pattern-name", "pattern-missing", "read-denied", "ack-bypass"} {
		t.Run(corruption, func(t *testing.T) {
			state := preservationEnvironment{30, true, true, []settings.EnvironmentReviewer{{Type: "User", ID: 7}}, &settings.EnvironmentBranchPolicy{CustomBranchPolicies: true}, map[int64]settings.EnvironmentPattern{7: {Type: "future", Name: "main"}, 8: {Type: "branch", Name: "main"}}}
			writes, warnings := 0, 0
			client := &EnvironmentsClient{rest: actionsClient(t, func(req *http.Request) (*http.Response, error) {
				respond := func(v any) (*http.Response, error) {
					data, _ := json.Marshal(v)
					return actionsResponse(req, 200, string(data)), nil
				}
				base := "/repos/owner/current/environments/production"
				switch req.URL.Path {
				case "/repos/owner/current":
					return actionsResponse(req, 200, `{"id":71,"full_name":"owner/current","archived":false}`), nil
				case "/repos/owner/current/environments":
					return actionsResponse(req, 200, `{"total_count":1,"environments":[{"id":1,"name":"production"}]}`), nil
				case base + "/deployment_protection_rules":
					return actionsResponse(req, 200, customEnvironmentEmpty), nil
				case base + "/deployment-branch-policies":
					if req.Method != "GET" {
						t.Fatal("unowned pattern mutation", req.Method)
					}
					entries := []any{}
					for id, p := range state.patterns {
						if writes > 0 && id == 7 {
							switch corruption {
							case "pattern-id":
								id = 9
							case "pattern-type":
								p.Type = "other"
							case "pattern-name":
								p.Name = "changed"
							case "pattern-missing":
								continue
							}
						}
						entries = append(entries, map[string]any{"id": id, "type": p.Type, "name": p.Name})
					}
					return respond(map[string]any{"total_count": len(entries), "branch_policies": entries})
				case base:
					if req.Method == "PUT" {
						var payload map[string]any
						json.NewDecoder(req.Body).Decode(&payload)
						if len(payload) != 5 || payload["can_admins_bypass"] != false {
							t.Fatal(payload)
						}
						state.bypass = false
						writes++
						ack := state
						if corruption == "ack-bypass" {
							ack.bypass = true
						}
						return respond(ack.detail())
					}
					if req.Method != "GET" {
						t.Fatal(req.Method)
					}
					actual := state
					if writes > 0 {
						switch corruption {
						case "timer":
							actual.timer = 0
						case "self-review":
							actual.prevent = false
						case "reviewers":
							actual.reviewers = []settings.EnvironmentReviewer{{Type: "User", ID: 9}}
						case "mode":
							actual.mode = nil
						case "read-denied":
							return actionsResponse(req, 403, `{}`), nil
						}
					}
					return respond(actual.detail())
				}
				t.Fatal(req.Method, req.URL)
				return nil, nil
			}).rest, NormalizationWarning: func(string, settings.EnvironmentDeploymentNormalization) { warnings++ }, RefTypeNormalizationWarning: func(string, settings.EnvironmentPatternState) { warnings++ }}
			off := false
			changed, err := client.Update(repository.Repository{Owner: "owner", Name: "current"}, settings.EnvironmentDeclaration{Name: "production", CanAdminsBypass: &off})
			var pre interface{ ReadFailedBeforeWrite() }
			if (err != nil) != (corruption != "none") || changed != (corruption == "none") || writes != 1 || warnings != 0 || errors.As(err, &pre) {
				t.Fatal(changed, writes, warnings, err)
			}
			if err != nil && !strings.Contains(err.Error(), "write may have applied") {
				t.Fatal("uncertain state hidden", err)
			}
			var effects interface{ EnvironmentWritesCompleted() }
			if errors.As(err, &effects) != strings.HasPrefix(corruption, "pattern-") {
				t.Fatal("verified policy progress differs from uncertain acknowledgment/readback", err)
			}
		})
	}
}

func TestEnvironmentOmittedUnreplayableModeStopsOnlyNeededPolicyWrites(t *testing.T) {
	for _, raw := range []bool{false, true} {
		for _, changed := range []bool{false, true} {
			t.Run(fmt.Sprintf("%t/%t", raw, changed), func(t *testing.T) {
				client := &EnvironmentsClient{rest: actionsClient(t, func(req *http.Request) (*http.Response, error) {
					if req.Method != "GET" {
						t.Fatal("unsafe raw policy update", req.Method)
					}
					switch req.URL.Path {
					case "/repos/owner/current":
						return actionsResponse(req, 200, `{"id":71,"full_name":"owner/current","archived":false}`), nil
					case "/repos/owner/current/environments":
						return actionsResponse(req, 200, `{"total_count":1,"environments":[{"id":1,"name":"production"}]}`), nil
					case "/repos/owner/current/environments/production":
						return actionsResponse(req, 200, fmt.Sprintf(`{"id":1,"name":"production","can_admins_bypass":false,"protection_rules":[],"deployment_branch_policy":{"protected_branches":%t,"custom_branch_policies":%t}}`, raw, raw)), nil
					case "/repos/owner/current/environments/production/deployment_protection_rules":
						return actionsResponse(req, 200, customEnvironmentEmpty), nil
					case "/repos/owner/current/environments/production/deployment-branch-policies":
						return actionsResponse(req, 200, `{"total_count":0,"branch_policies":[]}`), nil
					}
					t.Fatal(req.URL)
					return nil, nil
				}).rest}
				timer := 0
				if changed {
					timer = 5
				}
				wrote, err := client.Update(repository.Repository{Owner: "owner", Name: "current"}, settings.EnvironmentDeclaration{Name: "production", WaitTimer: &timer})
				var pre interface{ ReadFailedBeforeWrite() }
				if wrote || (err != nil) != changed || errors.As(err, &pre) != changed {
					t.Fatal(wrote, err)
				}
			})
		}
	}
}

func TestEnvironmentPatternOnlyUpdatesVerifyBuiltinsAndPrewriteFailures(t *testing.T) {
	for _, outcome := range []string{"success", "timer drift", "bypass drift", "reviewer drift", "identity read fails", "identity changes"} {
		t.Run(outcome, func(t *testing.T) {
			state := preservationEnvironment{30, true, true, []settings.EnvironmentReviewer{{Type: "User", ID: 7}}, &settings.EnvironmentBranchPolicy{CustomBranchPolicies: true}, nil}
			writes := 0
			client := &EnvironmentsClient{rest: actionsClient(t, func(req *http.Request) (*http.Response, error) {
				respond := func(v any) (*http.Response, error) {
					data, _ := json.Marshal(v)
					return actionsResponse(req, 200, string(data)), nil
				}
				base := "/repos/owner/current/environments/production"
				switch req.URL.Path {
				case "/repos/owner/current":
					return actionsResponse(req, 200, `{"id":71,"full_name":"owner/current","archived":false}`), nil
				case "/repos/owner/current/environments":
					return actionsResponse(req, 200, `{"total_count":1,"environments":[{"id":1,"name":"production"}]}`), nil
				case base:
					if req.Method != "GET" {
						t.Fatal("pattern-only change sent redundant policy write")
					}
					actual := state
					if writes > 0 {
						switch outcome {
						case "timer drift":
							actual.timer = 0
						case "bypass drift":
							actual.bypass = false
						case "reviewer drift":
							actual.reviewers = []settings.EnvironmentReviewer{{Type: "User", ID: 9}}
						}
					}
					return respond(actual.detail())
				case base + "/deployment_protection_rules":
					return actionsResponse(req, 200, customEnvironmentEmpty), nil
				case base + "/deployment-branch-policies":
					if req.Method != "GET" {
						t.Fatal(req.Method)
					}
					if writes > 0 {
						return actionsResponse(req, 200, `{"total_count":0,"branch_policies":[]}`), nil
					}
					return actionsResponse(req, 200, `{"total_count":1,"branch_policies":[{"id":7,"type":"branch","name":"obsolete"}]}`), nil
				case base + "/deployment-branch-policies/7":
					if req.Method == "GET" {
						if outcome == "identity read fails" {
							return actionsResponse(req, 403, `{}`), nil
						}
						name := "obsolete"
						if outcome == "identity changes" {
							name = "different"
						}
						return respond(map[string]any{"id": 7, "type": "branch", "name": name})
					}
					if req.Method != "DELETE" {
						t.Fatal(req.Method)
					}
					writes++
					return actionsResponse(req, 204, ""), nil
				}
				t.Fatal(req.Method, req.URL)
				return nil, nil
			}).rest}
			empty := []settings.EnvironmentPattern{}
			wrote, err := client.Update(repository.Repository{Owner: "owner", Name: "current"}, settings.EnvironmentDeclaration{Name: "production", DeploymentBranchPatterns: &empty})
			prewrite := strings.HasPrefix(outcome, "identity")
			var pre interface{ ReadFailedBeforeWrite() }
			if (err != nil) != (outcome != "success") || wrote != (outcome == "success") || errors.As(err, &pre) != prewrite || writes != map[bool]int{true: 0, false: 1}[prewrite] {
				t.Fatal(wrote, writes, err)
			}
		})
	}
}
