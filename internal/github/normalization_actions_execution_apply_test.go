package github

import (
	"encoding/json"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func TestNormalizedActionsExecutionHTTPWritesAndRawReadback(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		for _, raw := range []string{"FUTURE_POLICY", "", " \t"} {
			for _, choice := range []string{"all", "local_only", "selected", "disable", "sha"} {
				for _, drift := range []string{"", "ignored", "policy", "sha", "enabled"} {
					t.Run(choice+drift, func(t *testing.T) {
						base := map[string]any{"enabled": enabled, "allowed_actions": raw, "sha_pinning_required": true}
						selected := map[string]any{"github_owned_allowed": false, "verified_allowed": false, "patterns_allowed": []any{}}
						var writes []map[string]any
						var operations []string
						client := actionsClient(t, func(req *http.Request) (*http.Response, error) {
							isSelected := strings.HasSuffix(req.URL.Path, "/selected-actions")
							operations = append(operations, req.Method+" "+req.URL.Path)
							target := base
							if isSelected {
								if base["allowed_actions"] != "selected" {
									if choice == "selected" && drift == "ignored" && req.Method == "PUT" {
										return actionsResponse(req, 409, `{"message":"policy installation not retained"}`), nil
									}
									t.Fatal("inactive selected read/write")
								}
								target = selected
							}
							if req.Method == "PUT" {
								var payload map[string]any
								if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
									t.Fatal(err)
								}
								if isSelected && !enabled && base["enabled"] != false {
									t.Fatal("selection installed after enabling")
								}
								writes = append(writes, payload)
								if drift != "ignored" {
									for k, v := range payload {
										target[k] = v
									}
								}
								return actionsResponse(req, 204, ""), nil
							}
							if req.Method != "GET" {
								t.Fatal(req.Method)
							}
							if len(writes) > 0 && !isSelected {
								switch drift {
								case "policy":
									base["allowed_actions"] = "OTHER_POLICY"
								case "sha":
									base["sha_pinning_required"] = false
								case "enabled":
									base["enabled"] = !base["enabled"].(bool)
								}
							}
							body, _ := json.Marshal(target)
							return actionsResponse(req, 200, string(body)), nil
						})
						d := settings.ActionsPolicyDeclaration{}
						var wantWrites []map[string]any
						switch choice {
						case "disable":
							d.Enabled = actionsPointer(false)
							wantWrites = []map[string]any{{"enabled": false}}
						case "sha":
							d.SHAPinningRequired = actionsPointer(false)
							wantWrites = []map[string]any{{"enabled": enabled, "sha_pinning_required": false}}
						default:
							d.Enabled = actionsPointer(true)
							d.AllowedActions = actionsPointer(choice)
							wantWrites = []map[string]any{{"enabled": true, "allowed_actions": choice}}
							if choice == "selected" {
								d.SelectedActions = settings.SelectedActionsDeclaration{GitHubOwnedAllowed: actionsPointer(true), VerifiedAllowed: actionsPointer(false), PatternsAllowed: actionsPointer([]string{"org/*", "!org/bad@*", "org/*"})}
								wantWrites[0]["enabled"] = enabled
								wantWrites = append(wantWrites, map[string]any{"github_owned_allowed": true, "verified_allowed": false, "patterns_allowed": []any{"org/*", "!org/bad@*", "org/*"}})
								if !enabled {
									wantWrites = append(wantWrites, map[string]any{"enabled": true})
								}
							}
						}
						var warnings []settings.ObservationNormalization
						client.NormalizationWarning = func(n settings.ObservationNormalization) { warnings = append(warnings, n) }
						err := client.ApplyActionsPolicy(repository.Repository{Owner: "owner", Name: "current"}, d)
						wantErr := drift != ""
						if choice == "disable" && !enabled && drift == "ignored" || choice == "sha" && drift == "sha" {
							wantErr = false
						}
						// An ignored selected-policy installation cannot proceed to the selected endpoint.
						if choice == "selected" && drift == "ignored" {
							if err == nil || !strings.Contains(err.Error(), "may already have applied") || !reflect.DeepEqual(writes, wantWrites[:1]) {
								t.Fatal("failed selection installation", writes, err)
							}
							return
						}
						if (err != nil) != wantErr || !reflect.DeepEqual(writes, wantWrites) {
							t.Fatal(enabled, raw, choice, drift, writes, wantWrites, err)
						}
						if wantErr && !strings.Contains(err.Error(), "writes may already have applied") {
							t.Fatal("certainty lost", err)
						}
						var wantWarnings []settings.ObservationNormalization
						if choice != "sha" {
							if enabled {
								wantWarnings = append(wantWarnings, settings.ObservationNormalization{Path: settings.ActionsPermissionsPath, RawValue: raw, Fallback: string(settings.DefaultActionsPermissionsChoice)})
							}
							if base["enabled"] == true && !settings.ValidAllowedActions(base["allowed_actions"].(string)) {
								wantWarnings = append(wantWarnings, settings.ObservationNormalization{Path: settings.ActionsPermissionsPath, RawValue: base["allowed_actions"].(string), Fallback: string(settings.DefaultActionsPermissionsChoice)})
							}
						}
						if !reflect.DeepEqual(warnings, wantWarnings) {
							t.Fatal("warning visibility", warnings, wantWarnings, operations)
						}
					})
				}
			}
		}
	}
}

func TestNormalizedActionsExecutionIncompletePreflightNeverWrites(t *testing.T) {
	for _, key := range []string{"enabled", "allowed_actions", "sha_pinning_required"} {
		for _, invalid := range []string{"missing", "null", "wrongtype"} {
			t.Run(key+invalid, func(t *testing.T) {
				fields := map[string]any{"enabled": true, "allowed_actions": "FUTURE_POLICY", "sha_pinning_required": true}
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
					if req.Method != "GET" {
						t.Fatal("incomplete observation wrote")
					}
					return actionsResponse(req, 200, string(body)), nil
				})
				client.NormalizationWarning = func(settings.ObservationNormalization) { t.Fatal("incomplete observation warned") }
				err := client.ApplyActionsPolicy(repository.Repository{}, settings.ActionsPolicyDeclaration{Enabled: actionsPointer(true), AllowedActions: actionsPointer("all")})
				if err == nil || calls != 1 {
					t.Fatal(err, calls)
				}
			})
		}
	}
}
