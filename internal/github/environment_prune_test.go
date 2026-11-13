package github

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

// The fixture keeps child evidence distinct from ordinary Environment detail.
// Every prewrite failure must leave DELETE uncalled; only acknowledged absence
// can complete a deletion. It never reads a secret value.
func TestEnvironmentPruneFreshSnapshotAndAbsence(t *testing.T) {
	for _, mode := range []string{"success", "custom-added", "custom-denied", "custom-null", "parent-replaced", "policy-drift", "pattern-drift", "variable-drift", "secret-drift", "secret-partial", "secret-denied", "archive-drift", "late-archive-drift", "target-drift", "delete-rejected", "delete-redirect", "retained", "recreated", "readback-denied", "readback-incomplete"} {
		t.Run(mode, func(t *testing.T) {
			repo := repository.Repository{Owner: "owner", Name: "current"}
			phase, writes := "acquire", 0
			archiveReads := 0
			name := "preview/path name"
			base := "/repos/owner/current/environments/" + name
			client := &EnvironmentsClient{rest: actionsClient(t, func(r *http.Request) (*http.Response, error) {
				response := func(status int, body any) (*http.Response, error) {
					data, _ := json.Marshal(body)
					return actionsResponse(r, status, string(data)), nil
				}
				changed := phase == "delete"
				if r.Method == "DELETE" {
					if r.URL.EscapedPath() != "/repos/owner/current/environments/preview%2Fpath%20name" {
						t.Error("unescaped target", r.URL)
					}
					writes++
					switch mode {
					case "delete-rejected":
						return actionsResponse(r, 403, `{}`), nil
					case "delete-redirect":
						return actionsResponse(r, 302, `{}`), nil
					}
					return actionsResponse(r, 204, ""), nil
				}
				if r.Method != "GET" {
					t.Fatal(r.Method, r.URL)
				}
				switch r.URL.Path {
				case "/repos/owner/current":
					archiveReads++
					id := 71
					if changed && mode == "target-drift" {
						id++
					}
					return response(200, map[string]any{"id": id, "full_name": "owner/current", "archived": changed && (mode == "archive-drift" || mode == "late-archive-drift" && archiveReads >= 4)})
				case "/repos/owner/current/environments":
					if writes > 0 {
						switch mode {
						case "readback-denied":
							return actionsResponse(r, 403, `{}`), nil
						case "readback-incomplete":
							return actionsResponse(r, 200, `{"total_count":2,"environments":[]}`), nil
						case "retained":
						case "recreated":
							return response(200, map[string]any{"total_count": 1, "environments": []any{map[string]any{"id": 2, "name": name}}})
						default:
							return actionsResponse(r, 200, `{"total_count":0,"environments":[]}`), nil
						}
					}
					id := 1
					if changed && mode == "parent-replaced" {
						id++
					}
					return response(200, map[string]any{"total_count": 2, "environments": []any{map[string]any{"id": id, "name": name}, map[string]any{"id": 8, "name": "KEEP"}}})
				case base:
					bypass := changed && mode == "policy-drift"
					return response(200, map[string]any{"id": 1, "name": name, "protection_rules": []any{}, "can_admins_bypass": bypass, "deployment_branch_policy": map[string]any{"protected_branches": false, "custom_branch_policies": true}})
				case base + "/deployment_protection_rules":
					if changed {
						switch mode {
						case "custom-added":
							return actionsResponse(r, 200, `{"total_count":1,"custom_deployment_protection_rules":[{"id":9}]}`), nil
						case "custom-denied":
							return actionsResponse(r, 403, `{}`), nil
						case "custom-null":
							return actionsResponse(r, 200, `{"total_count":0,"custom_deployment_protection_rules":null}`), nil
						}
					}
					return actionsResponse(r, 200, customEnvironmentEmpty), nil
				case base + "/deployment-branch-policies":
					id := 7
					if changed && mode == "pattern-drift" {
						id++
					}
					return response(200, map[string]any{"total_count": 1, "branch_policies": []any{map[string]any{"id": id, "name": "main", "type": "branch"}}})
				case base + "/variables":
					value := "private-variable-value"
					if changed && mode == "variable-drift" {
						value = "changed"
					}
					return response(200, map[string]any{"total_count": 1, "variables": []any{map[string]any{"name": "KEY", "value": value}}})
				case base + "/secrets":
					if changed && mode == "secret-denied" {
						return actionsResponse(r, 403, `{"message":"SECRET-CANARY"}`), nil
					}
					if changed && mode == "secret-partial" {
						return actionsResponse(r, 200, `{"total_count":1,"secrets":[]}`), nil
					}
					updated := "2026-01-01T00:00:00Z"
					if changed && mode == "secret-drift" {
						updated = "2026-01-02T00:00:00Z"
					}
					return response(200, map[string]any{"total_count": 1, "secrets": []any{map[string]any{"name": "SECRET_CANARY", "created_at": "2026-01-01T00:00:00Z", "updated_at": updated}}})
				default:
					t.Fatalf("read retained/unknown resource: %s", r.URL)
					return nil, nil
				}
			}).rest}
			observation, err := client.ReadEnvironmentPrune(repo, []string{"keep"})
			if err != nil {
				t.Fatal(err)
			}
			candidates, err := observation.Candidates()
			if err != nil || len(candidates) != 1 {
				t.Fatal(candidates, err)
			}
			phase = "delete"
			err = client.DeleteEnvironment(repo, candidates[0])
			if mode == "success" {
				if err != nil || writes != 1 {
					t.Fatal(writes, err)
				}
				return
			}
			if err == nil {
				t.Fatal("unsafe deletion succeeded", writes)
			}
			prewrite := !strings.HasPrefix(mode, "delete-") && mode != "retained" && mode != "recreated" && !strings.HasPrefix(mode, "readback-")
			var failure *environmentPreWriteFailure
			if errors.As(err, &failure) != prewrite || (writes == 0) != prewrite {
				t.Fatal("incorrect effect boundary", writes, err)
			}
			if strings.Contains(err.Error(), "SECRET-CANARY") || strings.Contains(err.Error(), "private-variable-value") {
				t.Fatal("private evidence leaked", err)
			}
		})
	}
}

func TestEnvironmentPruneSecretMetadataPagination(t *testing.T) {
	repo := repository.Repository{Owner: "owner", Name: "current"}
	for _, mode := range []string{"complete", "short", "duplicate", "changed-count", "missing-member", "null", "unexpected-value"} {
		t.Run(mode, func(t *testing.T) {
			requests := 0
			c := &EnvironmentsClient{rest: actionsClient(t, func(r *http.Request) (*http.Response, error) {
				requests++
				if r.URL.Query().Get("per_page") != "100" || r.URL.Query().Get("page") != fmt.Sprint(requests) {
					t.Fatal(r.URL)
				}
				entries := []map[string]any{}
				count := 100
				if requests == 2 {
					count = 1
				}
				if mode == "short" {
					count = 1
				}
				for i := 0; i < count; i++ {
					name := fmt.Sprintf("KEY_%d", (requests-1)*100+i)
					if mode == "duplicate" {
						name = "SAME"
					}
					entry := map[string]any{"name": name, "created_at": "2026-01-01T00:00:00Z", "updated_at": "2026-01-01T00:00:00Z"}
					if mode == "missing-member" {
						delete(entry, "updated_at")
					}
					if mode == "unexpected-value" {
						entry["value"] = "MUST_NOT_READ"
					}
					entries = append(entries, entry)
				}
				total := 101
				if mode == "changed-count" && requests == 2 {
					total = 102
				}
				var values any = entries
				if mode == "null" {
					values = nil
				}
				body, _ := json.Marshal(map[string]any{"total_count": total, "secrets": values})
				return actionsResponse(r, 200, string(body)), nil
			}).rest}
			values, err := c.readEnvironmentSecretMetadata(repo, "production")
			if err == nil {
				timer, flag, reviewers := 0, false, []settings.EnvironmentReviewer{}
				inventory, _ := settings.NewVariableInventory(nil)
				_, err = settings.NewEnvironmentPruneCandidate(repo, settings.ArchiveState{ID: 1, FullName: "owner/current"}, settings.EnvironmentState{ID: 2, Policy: settings.EnvironmentDeclaration{Name: "production", WaitTimer: &timer, PreventSelfReview: &flag, CanAdminsBypass: &flag, Reviewers: &reviewers, DeploymentBranchPolicyDeclared: true}}, inventory, values, true)
			}
			if (err == nil) != (mode == "complete") {
				t.Fatal(requests, len(values), err)
			}
		})
	}
}
