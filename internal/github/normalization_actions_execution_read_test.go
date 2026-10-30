package github

import (
	"encoding/json"
	"net/http"
	"reflect"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func TestActionsExecutionReadNormalizesOnlyAcquiredPolicy(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		for _, sha := range []bool{false, true} {
			for _, raw := range []string{"FUTURE_POLICY", "", " \t", "all", "local_only", "selected"} {
				for _, acquireSelected := range []bool{false, true} {
					t.Run(raw, func(t *testing.T) {
						calls := 0
						client := actionsClient(t, func(req *http.Request) (*http.Response, error) {
							calls++
							if req.Method != "GET" {
								t.Fatal("read wrote", req.Method)
							}
							if calls == 2 {
								if raw != "selected" || !acquireSelected || req.URL.Path != "/repos/owner/current/actions/permissions/selected-actions" {
									t.Fatal("unexpected selected acquisition", req.URL)
								}
								return actionsResponse(req, 200, `{"github_owned_allowed":false,"verified_allowed":true,"patterns_allowed":[]}`), nil
							}
							body, _ := json.Marshal(map[string]any{"enabled": enabled, "sha_pinning_required": sha, "allowed_actions": raw, "selected_actions_url": "https://untrusted.invalid/never"})
							return actionsResponse(req, 200, string(body)), nil
						})
						client.NormalizationWarning = func(settings.ObservationNormalization) { t.Fatal("pure read warned") }
						state, err := client.ReadActionsPolicy(repository.Repository{Owner: "owner", Name: "current"}, acquireSelected)
						policy, record := raw, settings.ObservationNormalization{}
						if !settings.ValidAllowedActions(raw) {
							policy = "all"
							record = settings.ObservationNormalization{Path: settings.ActionsPermissionsPath, RawValue: raw, Fallback: string(settings.DefaultActionsPermissionsChoice)}
						}
						if err != nil || state.Enabled != enabled || state.SHAPinningRequired != sha || state.AllowedActions != policy || state.AllowedActionsNormalization != record {
							t.Fatal(state, err)
						}
						original, err := state.RawAllowedActions()
						if err != nil || original != raw {
							t.Fatal(original, err)
						}
						wantCalls := 1
						if raw == "selected" && acquireSelected {
							wantCalls = 2
						}
						if calls != wantCalls || (state.Selected != nil) != (wantCalls == 2) {
							t.Fatal(calls, state)
						}
					})
				}
			}
		}
	}
}

func TestActionsExecutionReadIncompleteDoesNotEscapeAsDefault(t *testing.T) {
	for _, key := range []string{"enabled", "allowed_actions", "sha_pinning_required"} {
		for _, invalid := range []string{"missing", "null", "wrongtype"} {
			t.Run(key+invalid, func(t *testing.T) {
				fields := map[string]any{"enabled": false, "allowed_actions": "FUTURE_POLICY", "sha_pinning_required": true}
				switch invalid {
				case "missing":
					delete(fields, key)
				case "null":
					fields[key] = nil
				case "wrongtype":
					fields[key] = 42
				}
				body, _ := json.Marshal(fields)
				calls := 0
				client := actionsClient(t, func(req *http.Request) (*http.Response, error) {
					calls++
					return actionsResponse(req, 200, string(body)), nil
				})
				state, err := client.ReadActionsPolicy(repository.Repository{}, true)
				if err == nil || !reflect.DeepEqual(state, settings.ActionsPolicyState{}) || calls != 1 {
					t.Fatal(state, err, calls)
				}
			})
		}
	}
}
