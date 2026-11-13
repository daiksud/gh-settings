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

func TestMatchingActionsExecutionDoesNotWrite(t *testing.T) {
	reads := 0
	client := actionsClient(t, func(req *http.Request) (*http.Response, error) {
		if req.Method != "GET" {
			t.Fatalf("matching execution policy sent %s", req.Method)
		}
		reads++
		return actionsResponse(req, 200, `{"enabled":true,"allowed_actions":"all","sha_pinning_required":false}`), nil
	})
	if wrote, err := client.ApplyActionsPolicy(repository.Repository{Owner: "owner", Name: "current"}, settings.ActionsPolicyDeclaration{Enabled: actionsPointer(true), AllowedActions: actionsPointer("all"), SHAPinningRequired: actionsPointer(false)}); err != nil || wrote {
		t.Fatal(wrote, err)
	}
	if reads != 1 {
		t.Fatal(reads)
	}
}

func TestActionsExecutionSkipsMatchingStages(t *testing.T) {
	for _, scenario := range []string{"matching selected", "selected only", "sha only", "enable matching selection", "enable changed selection", "declared companion drift", "omitted companion drift"} {
		t.Run(scenario, func(t *testing.T) {
			base := map[string]any{"enabled": !strings.HasPrefix(scenario, "enable"), "allowed_actions": "selected", "sha_pinning_required": true}
			selected := map[string]any{"github_owned_allowed": true, "verified_allowed": true, "patterns_allowed": []any{"keep/*"}}
			d := settings.ActionsPolicyDeclaration{Enabled: actionsPointer(true), AllowedActions: actionsPointer("selected"), SHAPinningRequired: actionsPointer(true), SelectedActions: settings.SelectedActionsDeclaration{VerifiedAllowed: actionsPointer(true)}}
			var want []map[string]any
			var wantPaths []string
			if scenario == "selected only" || scenario == "enable changed selection" || scenario == "omitted companion drift" {
				d.SelectedActions.VerifiedAllowed = actionsPointer(false)
				want = append(want, map[string]any{"github_owned_allowed": true, "verified_allowed": false, "patterns_allowed": []any{"keep/*"}})
				wantPaths = append(wantPaths, "/selected-actions")
			}
			if scenario == "sha only" || scenario == "declared companion drift" {
				d.SHAPinningRequired = actionsPointer(false)
				want = append(want, map[string]any{"enabled": true, "sha_pinning_required": false})
				wantPaths = append(wantPaths, "")
			}
			if strings.HasPrefix(scenario, "enable") {
				want = append(want, map[string]any{"enabled": true})
				wantPaths = append(wantPaths, "")
			}
			var writes []map[string]any
			var paths []string
			reads := 0
			client := actionsClient(t, func(req *http.Request) (*http.Response, error) {
				path := strings.TrimPrefix(req.URL.Path, "/repos/owner/current/actions/permissions")
				target := base
				if path == "/selected-actions" {
					target = selected
				} else if path != "" {
					t.Fatal(req.URL)
				}
				if req.Method == "PUT" {
					var body map[string]any
					if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
						t.Fatal(err)
					}
					writes = append(writes, body)
					paths = append(paths, path)
					if scenario == "enable changed selection" && path != "" && base["enabled"] != false {
						t.Fatal("enabled before selected write")
					}
					for k, v := range body {
						target[k] = v
					}
					return actionsResponse(req, 204, ""), nil
				}
				if req.Method != "GET" {
					t.Fatal(req.Method)
				}
				reads++
				if len(writes) > 0 {
					if scenario == "declared companion drift" {
						base["enabled"] = false
					}
					if scenario == "omitted companion drift" {
						selected["patterns_allowed"] = []any{}
					}
				}
				body, _ := json.Marshal(target)
				return actionsResponse(req, 200, string(body)), nil
			})
			wantFailure := strings.Contains(scenario, "drift")
			wrote, err := client.ApplyActionsPolicy(repository.Repository{Owner: "owner", Name: "current"}, d)
			if (err != nil) != wantFailure || wrote != (len(want) > 0 && !wantFailure) || !reflect.DeepEqual(writes, want) || !reflect.DeepEqual(paths, wantPaths) {
				t.Fatal(wrote, err, writes, paths, want, wantPaths)
			}
			if wantFailure && !strings.Contains(err.Error(), "may already have applied") {
				t.Fatal(err)
			}
			if !wantFailure {
				priorReads := reads
				if wrote, err := client.ApplyActionsPolicy(repository.Repository{Owner: "owner", Name: "current"}, d); err != nil || wrote || len(writes) != len(want) || reads != priorReads+2 {
					t.Fatal("repeat was not read-only", wrote, err, writes, reads)
				}
			}
		})
	}
}
