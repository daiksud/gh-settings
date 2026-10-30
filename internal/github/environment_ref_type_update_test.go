package github

import (
	"encoding/json"
	"errors"
	"maps"
	"net/http"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func TestEnvironmentRefTypeUpdatePreservesDeclaredPatternIntent(t *testing.T) {
	for _, tc := range []struct {
		name                string
		initial             map[int64]settings.EnvironmentPattern
		desiredPatterns     []settings.EnvironmentPattern
		omitPatterns        bool
		timer               int
		wantPOST            map[string]any
		wantHistory         []string
		wantRemaining       map[int64]settings.EnvironmentPattern
		wantWarningCount    int
		uncanonicalReadback bool
	}{
		{
			name:             "fallback equality still replaces raw identity",
			initial:          map[int64]settings.EnvironmentPattern{7: {Type: "future", Name: "main"}},
			desiredPatterns:  []settings.EnvironmentPattern{{Type: "branch", Name: "main"}},
			wantPOST:         map[string]any{"type": "branch", "name": "main"},
			wantHistory:      []string{"PUT", "POST branch/main", "GET future/main/7", "DELETE 7"},
			wantRemaining:    map[int64]settings.EnvironmentPattern{8: {Type: "branch", Name: "main"}},
			wantWarningCount: 2,
		},
		{
			name:             "fresh unsupported identity cannot confirm fallback equality",
			initial:          map[int64]settings.EnvironmentPattern{7: {Type: "future", Name: "main"}},
			desiredPatterns:  []settings.EnvironmentPattern{{Type: "branch", Name: "main"}},
			wantPOST:         map[string]any{"type": "branch", "name": "main"},
			wantHistory:      []string{"PUT", "POST branch/main", "GET future/main/7", "DELETE 7"},
			wantRemaining:    map[int64]settings.EnvironmentPattern{8: {Type: "future", Name: "main"}},
			wantWarningCount: 3, uncanonicalReadback: true,
		},
		{
			name:             "canonical companion retains its own ID",
			initial:          map[int64]settings.EnvironmentPattern{7: {Type: "future", Name: "main"}, 8: {Type: "branch", Name: "main"}},
			desiredPatterns:  []settings.EnvironmentPattern{{Type: "branch", Name: "main"}},
			wantHistory:      []string{"PUT", "GET future/main/7", "DELETE 7"},
			wantRemaining:    map[int64]settings.EnvironmentPattern{8: {Type: "branch", Name: "main"}},
			wantWarningCount: 2,
		},
		{
			name:             "tag replacement does not adopt the raw ID",
			initial:          map[int64]settings.EnvironmentPattern{7: {Type: "future", Name: "main"}},
			desiredPatterns:  []settings.EnvironmentPattern{{Type: "tag", Name: "main"}},
			wantPOST:         map[string]any{"type": "tag", "name": "main"},
			wantHistory:      []string{"PUT", "POST tag/main", "GET future/main/7", "DELETE 7"},
			wantRemaining:    map[int64]settings.EnvironmentPattern{8: {Type: "tag", Name: "main"}},
			wantWarningCount: 2,
		},
		{
			name:             "explicit empty clears the raw pattern",
			initial:          map[int64]settings.EnvironmentPattern{7: {Type: "future", Name: "main"}},
			desiredPatterns:  []settings.EnvironmentPattern{},
			wantHistory:      []string{"PUT", "GET future/main/7", "DELETE 7"},
			wantRemaining:    map[int64]settings.EnvironmentPattern{},
			wantWarningCount: 2,
		},
		{
			name:         "omitted list remains private during builtin update",
			initial:      map[int64]settings.EnvironmentPattern{7: {Type: "future", Name: "main"}},
			omitPatterns: true, timer: 30,
			wantHistory:   []string{"PUT"},
			wantRemaining: map[int64]settings.EnvironmentPattern{7: {Type: "future", Name: "main"}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			patterns := maps.Clone(tc.initial)
			var history []string
			var warnings []settings.EnvironmentPatternState
			putDone, finalRead := false, false
			currentTimer := 0
			client := &EnvironmentsClient{rest: actionsClient(t, func(req *http.Request) (*http.Response, error) {
				respond := func(body any) (*http.Response, error) {
					data, _ := json.Marshal(body)
					return actionsResponse(req, 200, string(data)), nil
				}
				base := "/repos/owner/current/environments/production"
				switch req.URL.Path {
				case "/repos/owner/current":
					if req.Method != "GET" {
						t.Fatal(req.Method)
					}
					return actionsResponse(req, 200, `{"id":71,"full_name":"owner/current","archived":false}`), nil
				case "/repos/owner/current/environments":
					if req.Method != "GET" {
						t.Fatal(req.Method)
					}
					return actionsResponse(req, 200, `{"total_count":1,"environments":[{"id":1,"name":"production"}]}`), nil
				case base:
					if req.Method == "PUT" {
						var payload map[string]any
						if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
							t.Fatal(err)
						}
						want := map[string]any{"wait_timer": float64(tc.timer), "prevent_self_review": false, "can_admins_bypass": false, "reviewers": []any{}, "deployment_branch_policy": map[string]any{"protected_branches": false, "custom_branch_policies": true}}
						if !reflect.DeepEqual(payload, want) || len(warnings) != tc.wantWarningCount/2 {
							t.Fatal("literal PUT or prewrite warning", payload, warnings)
						}
						history = append(history, "PUT")
						putDone = true
						currentTimer = tc.timer
					} else if req.Method != "GET" {
						t.Fatal(req.Method)
					}
					rules := []any{}
					if currentTimer != 0 {
						rules = append(rules, map[string]any{"type": "wait_timer", "wait_timer": currentTimer})
					}
					return respond(map[string]any{"id": 1, "name": "production", "protection_rules": rules, "can_admins_bypass": false, "deployment_branch_policy": map[string]bool{"protected_branches": false, "custom_branch_policies": true}})
				case base + "/deployment_protection_rules":
					if req.Method != "GET" {
						t.Fatal(req.Method)
					}
					return actionsResponse(req, 200, customEnvironmentEmpty), nil
				case base + "/deployment-branch-policies":
					if req.Method == "GET" {
						ids := []int64{}
						for id := range patterns {
							ids = append(ids, id)
						}
						sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
						entries := []any{}
						for _, id := range ids {
							p := patterns[id]
							entries = append(entries, map[string]any{"id": id, "type": p.Type, "name": p.Name})
						}
						if putDone && reflect.DeepEqual(patterns, tc.wantRemaining) {
							finalRead = true
						}
						return respond(map[string]any{"total_count": len(entries), "branch_policies": entries})
					}
					if req.Method != "POST" {
						t.Fatal(req.Method)
					}
					var payload map[string]any
					if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
						t.Fatal(err)
					}
					if tc.wantPOST == nil || !reflect.DeepEqual(payload, tc.wantPOST) || len(warnings) != 2 {
						t.Fatal("literal POST or fresh warning", payload, warnings)
					}
					patterns[8] = settings.EnvironmentPattern{Type: payload["type"].(string), Name: payload["name"].(string)}
					history = append(history, "POST "+patterns[8].Type+"/"+patterns[8].Name)
					return respond(map[string]any{"id": 8, "type": patterns[8].Type, "name": patterns[8].Name})
				case base + "/deployment-branch-policies/7":
					if req.Method == "GET" {
						history = append(history, "GET future/main/7")
						return respond(map[string]any{"id": 7, "type": "future", "name": "main"})
					}
					if req.Method != "DELETE" {
						t.Fatal(req.Method)
					}
					history = append(history, "DELETE 7")
					delete(patterns, 7)
					if tc.uncanonicalReadback {
						patterns[8] = settings.EnvironmentPattern{Type: "future", Name: "main"}
					}
					return actionsResponse(req, 204, ""), nil
				default:
					t.Fatal("unexpected request", req.Method, req.URL)
					return nil, nil
				}
			}).rest, RefTypeNormalizationWarning: func(name string, pattern settings.EnvironmentPatternState) {
				if name != "production" {
					t.Fatal(name)
				}
				warnings = append(warnings, pattern)
			}}
			off, reviewers := false, []settings.EnvironmentReviewer{}
			var desiredPatterns *[]settings.EnvironmentPattern
			if !tc.omitPatterns {
				desiredPatterns = &tc.desiredPatterns
			}
			desired := settings.EnvironmentDeclaration{Name: "production", WaitTimer: &tc.timer, PreventSelfReview: &off, CanAdminsBypass: &off, Reviewers: &reviewers, DeploymentBranchPolicyDeclared: true, DeploymentBranchPolicy: &settings.EnvironmentBranchPolicy{CustomBranchPolicies: true}, DeploymentBranchPatterns: desiredPatterns}
			written, err := client.Update(repository.Repository{Owner: "owner", Name: "current"}, desired)
			if (err != nil) != tc.uncanonicalReadback || written == tc.uncanonicalReadback || !finalRead || !reflect.DeepEqual(history, tc.wantHistory) || !reflect.DeepEqual(patterns, tc.wantRemaining) {
				t.Fatal("fallback equality hid raw replacement or verification", written, history, patterns, finalRead, err)
			}
			wantWarning := settings.EnvironmentPatternState{ID: 7, Pattern: settings.EnvironmentPattern{Type: "branch", Name: "main"}, RefTypeNormalization: settings.ObservationNormalization{Path: settings.EnvironmentRefTypePath, RawValue: "future", Fallback: "branch"}}
			var wantWarnings []settings.EnvironmentPatternState
			for range tc.wantWarningCount {
				wantWarnings = append(wantWarnings, wantWarning)
			}
			if tc.uncanonicalReadback {
				wantWarnings[2].ID = 8
			}
			if !reflect.DeepEqual(warnings, wantWarnings) {
				t.Fatal("fresh scoped raw warnings missing or changed", warnings)
			}
			if tc.uncanonicalReadback {
				var pre interface{ ReadFailedBeforeWrite() }
				if !strings.Contains(err.Error(), "verify environment readback (write may have applied)") || errors.As(err, &pre) {
					t.Fatal("final failure hid prior writes", err)
				}
				return
			}
			written, err = client.Update(repository.Repository{Owner: "owner", Name: "current"}, desired)
			if err != nil || written || !reflect.DeepEqual(history, tc.wantHistory) || !reflect.DeepEqual(warnings, wantWarnings) {
				t.Fatal("canonical repeat wrote or warned", written, history, warnings, err)
			}
		})
	}
}
