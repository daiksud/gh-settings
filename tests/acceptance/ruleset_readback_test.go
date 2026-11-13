//go:build acceptance

package acceptance

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"testing"
)

func TestBasicRulesetReadbackBeforePrune(t *testing.T) {
	binary := acceptanceBinary(t)
	for _, target := range []string{"branch", "tag"} {
		for _, mode := range []string{"verified and repeat", "mismatch", "denied", "fallback equality"} {
			t.Run(target+"/"+mode, func(t *testing.T) {
				var mu sync.Mutex
				var current map[string]any
				if err := json.Unmarshal([]byte(acceptanceRuleset), &current); err != nil {
					t.Fatal(err)
				}
				current["target"] = target
				current["conditions"] = map[string]any{"ref_name": map[string]any{"include": []string{"~ALL"}, "exclude": []string{}}}
				unlisted := map[string]any{}
				for key, value := range current {
					unlisted[key] = value
				}
				unlisted["id"], unlisted["name"] = 8, "unlisted"
				present, updated := true, false
				var mutations []string
				var phases []string
				fixture, err := startHTTPFixture(func(w http.ResponseWriter, r *http.Request) {
					mu.Lock()
					defer mu.Unlock()
					w.Header().Set("Content-Type", "application/json")
					if r.Method == "GET" && r.URL.Query().Get("includes_parents") != "false" {
						t.Error("missing repository-only read scope", r.URL)
						w.WriteHeader(400)
						return
					}
					switch r.URL.Path {
					case "/repos/owner/current/rulesets":
						if r.Method != "GET" {
							t.Error("unexpected collection write", r.Method)
							w.WriteHeader(400)
							return
						}
						items := []any{map[string]any{"id": 7, "source_type": "Repository", "source": "owner/current"}}
						if present {
							items = append(items, map[string]any{"id": 8, "source_type": "Repository", "source": "owner/current"})
						}
						json.NewEncoder(w).Encode(items)
					case "/repos/owner/current/rulesets/7":
						if r.Method == "GET" {
							if updated {
								phases = append(phases, "readback")
								if mode == "denied" {
									w.WriteHeader(403)
									fmt.Fprint(w, `{"message":"fixture denied"}`)
									return
								}
								observed := map[string]any{}
								for key, value := range current {
									observed[key] = value
								}
								if mode == "mismatch" {
									observed["enforcement"] = "active"
								}
								if mode == "fallback equality" {
									observed["enforcement"] = "FUTURE"
								}
								json.NewEncoder(w).Encode(observed)
								return
							}
							json.NewEncoder(w).Encode(current)
							return
						}
						if r.Method != "PUT" {
							t.Error("unexpected selected mutation", r.Method)
							w.WriteHeader(400)
							return
						}
						var payload map[string]any
						if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
							t.Error(err)
						}
						if len(payload) != 6 || payload["id"] != nil || payload["enforcement"] != "disabled" || len(payload["rules"].([]any)) != 0 {
							t.Error("unexpected whole-object policy", payload)
						}
						payload["id"], payload["source_type"], payload["source"] = 7, "Repository", "owner/current"
						current, updated = payload, true
						mutations = append(mutations, "PUT 7")
						phases = append(phases, "PUT 7")
						json.NewEncoder(w).Encode(payload)
					case "/repos/owner/current/rulesets/8":
						if r.Method == "GET" && present {
							json.NewEncoder(w).Encode(unlisted)
							return
						}
						if r.Method != "DELETE" || !present {
							t.Error("unexpected unlisted request", r.Method)
							w.WriteHeader(400)
							return
						}
						mutations = append(mutations, "DELETE 8")
						phases = append(phases, "DELETE 8")
						present = false
						w.WriteHeader(204)
					default:
						t.Error("unexpected later resource", r.Method, r.URL)
						w.WriteHeader(400)
					}
				}, false, "/rulesets", "/rulesets/")
				if err != nil {
					t.Fatal(err)
				}
				defer fixture.close()
				env := labelIDFixtureEnv(fixture.cfg, false)
				if err := setupOrdinaryOrigin(fixture.work, env); err != nil {
					t.Fatal(err)
				}
				selector := "target_branches: {include_all_branches: true}"
				if target == "tag" {
					selector = "target_tags: {include_all_tags: true}"
				}
				input := repositoryInput("settings:\n  rulesets:\n    - {id: 7, ruleset_name: basic, enforcement_status: disabled, " + selector + "}\n")
				if err := writeSettingsFile(fixture.work, input); err != nil {
					t.Fatal(err)
				}
				result, err := executeCLI(binary, fixture.work, fixture.root, env, []string{"apply", "--prune", "rulesets"}, false)
				if err != nil {
					t.Fatal(err)
				}
				if mode == "verified and repeat" {
					if result.runErr != nil || result.stdout.String() != "Applied repository settings.\n" || hasNonInfoDiagnostic(result.stderr.String()) {
						t.Fatal(result)
					}
					repeat, err := executeCLI(binary, fixture.work, fixture.root, env, []string{"apply", "--prune", "rulesets"}, false)
					if err != nil || repeat.runErr != nil || hasNonInfoDiagnostic(repeat.stderr.String()) {
						t.Fatal(repeat, err)
					}
				} else if result.runErr == nil || result.stdout.Len() != 0 || !strings.Contains(result.stderr.String(), "writes may already have applied") {
					t.Fatalf("unverified basic update reported success: %+v", result)
				}
				mu.Lock()
				defer mu.Unlock()
				want := []string{"PUT 7"}
				if mode == "verified and repeat" {
					want = append(want, "DELETE 8")
				}
				if !reflect.DeepEqual(mutations, want) || len(phases) < 2 || phases[0] != "PUT 7" || phases[1] != "readback" {
					t.Fatal("readback ordering or stop lost", mutations, phases)
				}
				if mode != "verified and repeat" && (!present || len(phases) != 2) {
					t.Fatal("continued after failed readback", phases)
				}
				if err := checkSettingsUnchanged(fixture.work, input); err != nil {
					t.Fatal(err)
				}
			})
		}
	}
}
