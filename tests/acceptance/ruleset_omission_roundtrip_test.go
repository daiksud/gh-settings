//go:build acceptance

package acceptance

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"

	"go.yaml.in/yaml/v4"
)

func TestWriteOnlyDismissalContract(t *testing.T) {
	binary := acceptanceBinary(t)
	for _, mode := range []string{"omitted stable", "observed explicit", "false omitted response", "true omitted response", "true omitted readback", "explicit mismatch", "other readback drift", "newly observable false", "readback denied", "write denied"} {
		t.Run(mode, func(t *testing.T) {
			var state map[string]any
			if err := json.Unmarshal([]byte(acceptanceReviewRuleset), &state); err != nil {
				t.Fatal(err)
			}
			if mode != "observed explicit" {
				delete(acceptanceRuleParams(t, state, "pull_request"), "dismissal_restriction")
			}
			var mu sync.Mutex
			writes, readbacks := 0, 0
			fixture, err := startHTTPFixture(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				w.Header().Set("Content-Type", "application/json")
				if servePRAccessFixtureRead(w, r) || serveIssueCreationPolicyFixtureRead(w, r) || serveSponsorshipsFixtureRead(w, r) {
					return
				}
				if isMergeMethodsFixtureRead(r) {
					fmt.Fprint(w, mergeMethodsFixtureResponse)
					return
				}
				if r.Method == "PUT" && r.URL.Path == "/repos/owner/current/rulesets/7" {
					writes++
					if mode == "write denied" {
						w.WriteHeader(http.StatusForbidden)
						fmt.Fprint(w, `{"message":"fixture denied write"}`)
						return
					}
					var payload map[string]any
					if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
						t.Error(err)
					}
					if len(payload) != 6 || payload["id"] != nil || payload["enforcement"] != "disabled" {
						t.Error("unexpected whole-object payload", payload)
					}
					parameters := acceptanceRuleParams(t, payload, "pull_request")
					v, present := parameters["dismissal_restriction"]
					explicit := strings.HasPrefix(mode, "false") || strings.HasPrefix(mode, "true") || mode == "explicit mismatch"
					if explicit {
						if !present || v.(map[string]any)["enabled"] != !strings.HasPrefix(mode, "false") {
							t.Error("explicit value was omitted or changed", parameters)
						}
					} else if present {
						t.Error("absent wire value was invented", parameters)
					}
					payload["id"], payload["source"], payload["source_type"] = float64(7), "owner/current", "Repository"
					state = payload
					if strings.HasSuffix(mode, "omitted response") {
						delete(parameters, "dismissal_restriction")
					} else if mode == "explicit mismatch" {
						parameters["dismissal_restriction"].(map[string]any)["enabled"] = false
					}
					json.NewEncoder(w).Encode(state)
					// Echoed PUT success cannot replace an independent post-write GET.
					switch mode {
					case "true omitted readback":
						delete(parameters, "dismissal_restriction")
					case "other readback drift":
						parameters["require_code_owner_review"] = false
					case "newly observable false":
						parameters["dismissal_restriction"] = map[string]any{"enabled": false, "allowed_actors": []any{}}
					}
					return
				}
				if r.Method != "GET" {
					t.Error("unexpected mutation", r.Method, r.URL)
					w.WriteHeader(400)
					return
				}
				switch r.URL.Path {
				case "/repos/owner/current":
					fmt.Fprint(w, planResponse)
				case "/repos/owner/current/topics":
					fmt.Fprint(w, `{"names":[]}`)
				case "/repos/owner/current/labels":
					fmt.Fprint(w, `[]`)
				case "/repos/owner/current/rulesets":
					fmt.Fprint(w, `[{"id":7,"source_type":"Repository","source":"owner/current"}]`)
				case "/repos/owner/current/rulesets/7":
					if writes != 0 {
						readbacks++
						if mode == "readback denied" {
							w.WriteHeader(403)
							fmt.Fprint(w, `{"message":"denied"}`)
							return
						}
					}
					json.NewEncoder(w).Encode(state)
				default:
					t.Error("unexpected path", r.URL)
					w.WriteHeader(404)
				}
			}, false)
			if err != nil {
				t.Fatal(err)
			}
			defer fixture.close()
			env := exportFixtureEnv(fixture.cfg)
			if err := setupOrdinaryOrigin(fixture.work, env); err != nil {
				t.Fatal(err)
			}
			run := func(args ...string) string {
				t.Helper()
				r, err := executeCLI(binary, fixture.work, fixture.root, env, args, false)
				if err != nil || r.runErr != nil {
					t.Fatalf("%v: %v %v %s", args, err, r.runErr, r.stderr.String())
				}
				return r.stdout.String()
			}
			run("export", "--rulesets")
			data, err := os.ReadFile(filepath.Join(fixture.work, ".github/settings.yml"))
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(data), "dismissal_restriction") || strings.Contains(string(data), "restrict_who_can_dismiss_pull_request_reviews") || strings.Contains(string(data), "people_teams_or_apps_that_can_dismiss_reviews") {
				t.Fatal("export included unsupported write-only field", string(data))
			}
			var document map[string]any
			if err := yaml.Unmarshal(data, &document); err != nil {
				t.Fatal(err)
			}
			members := document["repository"].(map[string]any)["settings"].(map[string]any)["rulesets"].([]any)
			desired := map[string]any{
				"id": 7, "ruleset_name": "default", "enforcement_status": "active",
				"target_branches": map[string]any{"include_default_branch": true, "include_all_branches": false, "include_by_pattern": []any{}, "exclude_by_pattern": []any{}},
				"bypass_list":     []any{},
				"rules": map[string]any{
					"restrict_deletions": true, "block_force_pushes": true, "require_linear_history": true, "require_signed_commits": false,
					"require_a_pull_request_before_merging": map[string]any{
						"required_approvals": 1, "allowed_merge_methods": []any{"squash"},
						"dismiss_stale_pull_request_approvals_when_new_commits_are_pushed":      true,
						"require_review_from_code_owners":                                       true,
						"require_an_additional_approval_for_unattributed_copilot_pull_requests": false,
						"require_approval_of_the_most_recent_reviewable_push":                   false,
						"require_conversation_resolution_before_merging":                        true,
						"require_review_from_specific_teams":                                    []any{map[string]any{"reviewer": 22, "approvals": 1, "file_patterns": []any{"**", "!docs/**", "docs/README.md", "**"}}},
					},
					"require_status_checks_to_pass": map[string]any{
						"required_status_checks":                           []any{map[string]any{"check": "gh-settings", "source": map[string]any{"app": 15368}}, map[string]any{"check": "pass", "source": map[string]any{"app": 15368}}},
						"require_branches_to_be_up_to_date_before_merging": true,
						"do_not_require_status_checks_on_creation":         false,
					},
				},
			}
			if !reflect.DeepEqual(members, []any{desired}) {
				t.Fatalf("export=%v want=%v", members, desired)
			}
			parameters := desired["rules"].(map[string]any)["require_a_pull_request_before_merging"].(map[string]any)
			document = map[string]any{"version": "daiksud/gh-settings@v0", "repository": map[string]any{"settings": map[string]any{"rulesets": []any{desired}}}}

			save := func() string {
				t.Helper()
				data, err := yaml.Marshal(document)
				if err != nil {
					t.Fatal(err)
				}
				if err := writeSettingsFile(fixture.work, string(data)); err != nil {
					t.Fatal(err)
				}
				return string(data)
			}
			save()
			run("validate")
			if out := run("plan"); out != "No differences for managed repository settings.\n" {
				t.Fatal("observable round trip produced spurious drift", out)
			}
			reportPath := filepath.Join(fixture.root, "observable-plan.json")
			run("action", "plan", reportPath)
			reportData, err := os.ReadFile(reportPath)
			if err != nil {
				t.Fatal(err)
			}
			var report map[string]any
			if err := json.Unmarshal(reportData, &report); err != nil || report["outcome"] != "no-difference" {
				t.Fatalf("observable plan report: %s %v", reportData, err)
			}
			desired["enforcement_status"] = "disabled"
			if strings.HasPrefix(mode, "false") || strings.HasPrefix(mode, "true") || mode == "explicit mismatch" {
				parameters["restrict_who_can_dismiss_pull_request_reviews"] = !strings.HasPrefix(mode, "false")
				parameters["people_teams_or_apps_that_can_dismiss_reviews"] = []any{}
			}
			input := save()
			explicit := parameters["restrict_who_can_dismiss_pull_request_reviews"] != nil
			if explicit {
				for _, args := range [][]string{{"plan"}, {"action", "plan", reportPath}} {
					r, err := executeCLI(binary, fixture.work, fixture.root, env, args, false)
					if err != nil || r.runErr == nil || !strings.Contains(r.stderr.String(), "plan does not support write-only repository.settings.rulesets.rules.require_a_pull_request_before_merging.restrict_who_can_dismiss_pull_request_reviews") || r.stdout.Len() != 0 {
						t.Fatalf("write-only plan did not fail honestly: %v %v %s %s", err, r.runErr, r.stdout.String(), r.stderr.String())
					}
				}
			}
			run("validate")
			r, err := executeCLI(binary, fixture.work, fixture.root, env, []string{"apply"}, false)
			success := mode != "other readback drift" && mode != "readback denied" && mode != "write denied"
			if err != nil || (r.runErr == nil) != success || (!success && (r.stdout.Len() != 0 || (mode != "write denied" && !strings.Contains(r.stderr.String(), "writes may already have applied")))) {
				t.Fatalf("wrong verification outcome: %v %v %s %s", err, r.runErr, r.stdout.String(), r.stderr.String())
			}
			mu.Lock()
			if writes != 1 || (success && readbacks != 1) {
				t.Errorf("retry or missing independent readback: writes=%d readbacks=%d", writes, readbacks)
			}
			mu.Unlock()
			if success && explicit && !strings.Contains(r.stdout.String(), "accepted by GitHub; value not verified (write-only)") {
				t.Fatal("write-only success was described as verified", r.stdout.String())
			}
			if success && !explicit && run("plan") != "No differences for managed repository settings.\n" {
				t.Fatal("observable update was not idempotent")
			}
			if success && explicit {
				run("apply")
				mu.Lock()
				if writes != 2 {
					t.Error("repeat write-only apply incorrectly inferred a verified no-op", writes)
				}
				mu.Unlock()
			}
			if err := checkSettingsUnchanged(fixture.work, input); err != nil {
				t.Fatal(err)
			}
		})
	}
}
