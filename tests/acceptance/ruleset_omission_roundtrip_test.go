//go:build acceptance

package acceptance

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
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
			if strings.Contains(string(data), "dismissal_restriction") {
				t.Fatal("export included unsupported write-only field", string(data))
			}
			var document map[string]any
			if err := yaml.Unmarshal(data, &document); err != nil {
				t.Fatal(err)
			}
			members := document["settings"].(map[string]any)["rulesets"].([]any)
			desired := members[0].(map[string]any)
			parameters := acceptanceRuleParams(t, desired, "pull_request")
			document = map[string]any{"schemaVersion": 1, "settings": map[string]any{"rulesets": members}}
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
			if mode == "observed explicit" {
				delete(parameters, "dismissal_restriction")
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
			desired["enforcement"] = "disabled"
			if strings.HasPrefix(mode, "false") || strings.HasPrefix(mode, "true") || mode == "explicit mismatch" {
				parameters["dismissal_restriction"] = map[string]any{"enabled": !strings.HasPrefix(mode, "false"), "allowed_actors": []any{}}
			}
			input := save()
			explicit := parameters["dismissal_restriction"] != nil
			if explicit {
				for _, args := range [][]string{{"plan"}, {"action", "plan", reportPath}} {
					r, err := executeCLI(binary, fixture.work, fixture.root, env, args, false)
					if err != nil || r.runErr == nil || !strings.Contains(r.stderr.String(), "plan does not support write-only dismissal_restriction") || r.stdout.Len() != 0 {
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
