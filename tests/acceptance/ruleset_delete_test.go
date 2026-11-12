//go:build acceptance

package acceptance

import (
	"encoding/json"
	"fmt"
	"go.yaml.in/yaml/v4"
	"html"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestRulesetPruneExecutableLifecycle(t *testing.T) {
	for _, target := range []string{"branch", "tag"} {
		t.Run(target, func(t *testing.T) {
			var mu sync.Mutex
			present, ignore := true, false
			writes, reads := 0, 0
			var state map[string]any
			json.Unmarshal([]byte(acceptanceRuleset), &state)
			state["target"] = target
			state["conditions"] = map[string]any{"ref_name": map[string]any{"include": []any{"~ALL"}, "exclude": []any{}}}
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
				switch r.URL.Path {
				case "/repos/owner/current":
					fmt.Fprint(w, planResponse)
				case "/repos/owner/current/topics":
					fmt.Fprint(w, `{"names":[]}`)
				case "/repos/owner/current/labels":
					fmt.Fprint(w, `[]`)
				case "/repos/owner/current/rulesets":
					if r.Method != "GET" {
						t.Error("unexpected method", r.Method)
						w.WriteHeader(400)
						return
					}
					reads++
					if r.URL.Query().Get("includes_parents") != "false" {
						t.Error("parents not excluded", r.URL)
					}
					items := []any{map[string]any{"id": 8, "source_type": "Repository", "source": "owner/current"}}
					if present {
						items = append(items, map[string]any{"id": 7, "source_type": "Repository", "source": "owner/current"})
					}
					json.NewEncoder(w).Encode(items)
				case "/repos/owner/current/rulesets/8":
					if r.Method != "GET" {
						t.Error("unselected object mutated", r.Method)
						w.WriteHeader(400)
						return
					}
					other := map[string]any{}
					for k, v := range state {
						other[k] = v
					}
					other["id"], other["name"] = 8, "unmanaged"
					json.NewEncoder(w).Encode(other)
				case "/repos/owner/current/rulesets/7":
					if r.Method == "GET" {
						if !present {
							w.WriteHeader(404)
							return
						}
						json.NewEncoder(w).Encode(state)
						return
					}
					if r.Method != "DELETE" || !present {
						t.Error("unexpected mutation", r.Method, present)
						w.WriteHeader(400)
						return
					}
					writes++
					if !ignore {
						present = false
					}
					w.WriteHeader(204)
				default:
					t.Error("unexpected path", r.Method, r.URL)
					w.WriteHeader(400)
				}
			}, false)
			if err != nil {
				t.Fatal(err)
			}
			defer fixture.close()
			env := append(exportFixtureEnv(fixture.cfg), "SETTINGS_PRUNE=rulesets", "SETTINGS_TARGET=owner/current", "SETTINGS_ACTOR=user", "SETTINGS_TRIGGERING_ACTOR=user", "SETTINGS_EVENT=push", "SETTINGS_RUN_ID=123", "SETTINGS_RUN_ATTEMPT=1", "SETTINGS_WORKFLOW_REF=owner/current/.github/workflows/settings.yml@refs/heads/main", "SETTINGS_WORKFLOW_SHA="+strings.Repeat("a", 40), "SETTINGS_CONFIG_SHA="+strings.Repeat("b", 40), "SETTINGS_CLI_VERSION=dev", "SETTINGS_CLI_SHA256="+strings.Repeat("c", 64))
			if err := setupOrdinaryOrigin(fixture.work, env); err != nil {
				t.Fatal(err)
			}
			run := func(success bool, args ...string) string {
				t.Helper()
				result, err := executeCLI(acceptanceBinary(t), fixture.work, fixture.root, env, args, false)
				if err != nil || (result.runErr == nil) != success {
					t.Fatalf("args=%v err=%v run=%v out=%s stderr=%s", args, err, result.runErr, result.stdout.String(), result.stderr.String())
				}
				return result.stdout.String() + withoutExportInfo(result.stderr.String())
			}
			run(true, "export", "--rulesets")
			data, err := os.ReadFile(filepath.Join(fixture.work, ".github", "settings.yml"))
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(data), "delete:") {
				t.Fatal("export manufactured deletion", string(data))
			}
			for _, config := range []string{"[{ruleset_name: basic, delete: true}]", "[{id: 0, ruleset_name: basic, delete: true}]", "[{id: 7, delete: true}]", "[{id: 7, ruleset_name: basic, delete: false}]", "[{id: 7, ruleset_name: basic, delete: true, enforcement_status: active}]", "[{id: 7, ruleset_name: basic, delete: true, rules: {restrict_deletions: false}}]"} {
				if err := writeSettingsFile(fixture.work, repositoryInput("settings: {rulesets: "+config+"}\n")); err != nil {
					t.Fatal(err)
				}
				mu.Lock()
				before := reads
				mu.Unlock()
				run(false, "validate")
				mu.Lock()
				if reads != before || writes != 0 {
					t.Error("invalid tombstone reached remote")
				}
				mu.Unlock()
			}
			var exported map[string]any
			if err := yaml.Unmarshal(data, &exported); err != nil {
				t.Fatal(err)
			}
			settingsMap := exported["repository"].(map[string]any)["settings"].(map[string]any)
			keep := []any{}
			for _, entry := range settingsMap["rulesets"].([]any) {
				object := entry.(map[string]any)
				if object["ruleset_name"] == "unmanaged" {
					keep = append(keep, object)
				}
			}
			settingsMap["rulesets"] = keep
			exported["repository"] = map[string]any{"settings": map[string]any{"rulesets": keep}}
			retained, err := yaml.Marshal(exported)
			if err != nil {
				t.Fatal(err)
			}
			intent := string(retained)
			if err := writeSettingsFile(fixture.work, intent); err != nil {
				t.Fatal(err)
			}
			run(true, "validate")
			if out := run(true, "plan", "--prune", "rulesets"); !strings.Contains(out, `delete "basic"`) || !strings.Contains(out, "→ absent") {
				t.Fatal(out)
			}
			report, summary := filepath.Join(fixture.root, "plan.json"), filepath.Join(fixture.root, "summary.md")
			run(true, "action", "plan", report)
			data, err = os.ReadFile(report)
			if err != nil {
				t.Fatal(err)
			}
			var saved struct {
				Outcome  string
				Rulesets []struct {
					Operation string
					Before    map[string]any
					After     map[string]any
				}
			}
			if json.Unmarshal(data, &saved) != nil || saved.Outcome != "changed" || len(saved.Rulesets) != 1 || saved.Rulesets[0].Operation != "delete" || saved.Rulesets[0].Before["id"] != float64(7) || saved.Rulesets[0].After != nil {
				t.Fatal(string(data))
			}
			run(true, "action", "summary", report, summary)
			data, err = os.ReadFile(summary)
			if err != nil || !strings.Contains(html.UnescapeString(string(data)), `delete "basic"`) || !strings.Contains(string(data), "absent") {
				t.Fatal(string(data), err)
			}
			mu.Lock()
			if writes != 0 {
				t.Fatal("plan/summary mutated")
			}
			ignore = true
			mu.Unlock()
			if out := run(false, "apply", "--prune", "rulesets"); !strings.Contains(out, "write may have applied") {
				t.Fatal("uncertain deletion not reported", out)
			}
			mu.Lock()
			ignore = false
			before := reads
			mu.Unlock()
			run(true, "apply", "--prune", "rulesets")
			mu.Lock()
			if reads-before != 3 || writes != 2 || present {
				t.Error("preflight and fresh inventory required", reads-before, writes, present)
			}
			mu.Unlock()
			if err := checkSettingsUnchanged(fixture.work, intent); err != nil {
				t.Fatal(err)
			}
			if out := run(true, "plan", "--prune", "rulesets"); !strings.Contains(out, "No differences") {
				t.Fatal(out)
			}
			run(true, "apply", "--prune", "rulesets")
			mu.Lock()
			if writes != 2 {
				t.Error("repeat deleted", writes)
			}
			mu.Unlock()
			run(true, "export", "--rulesets")
			data, err = os.ReadFile(filepath.Join(fixture.work, ".github", "settings.yml"))
			if err != nil || strings.Contains(string(data), "ruleset_name: basic") || !strings.Contains(string(data), "ruleset_name: unmanaged") || strings.Contains(string(data), "delete:") {
				t.Fatal(string(data), err)
			}
		})
	}
}
