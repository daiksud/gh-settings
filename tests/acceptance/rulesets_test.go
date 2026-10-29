//go:build acceptance

package acceptance

import (
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"

	"go.yaml.in/yaml/v4"
)

const acceptanceRuleset = `{"id":7,"name":"basic","source_type":"Repository","source":"owner/current","current_user_can_bypass":"never","target":"branch","enforcement":"active","bypass_actors":[],"conditions":{"ref_name":{"include":["~DEFAULT_BRANCH"],"exclude":[]}},"rules":[{"type":"deletion"},{"type":"non_fast_forward"}]}`

func TestRulesetsIDOwnedLifecycleAndWriteback(t *testing.T) {
	binary := acceptanceBinary(t)
	var mu sync.Mutex
	states := map[string]map[string]any{}
	for _, pair := range []struct{ id, name string }{{"7", "basic"}, {"8", "old"}} {
		var state map[string]any
		json.Unmarshal([]byte(acceptanceRuleset), &state)
		state["name"] = pair.name
		state["id"] = float64(7)
		if pair.id == "8" {
			state["id"] = float64(8)
		}
		states[pair.id] = state
	}
	var writes []string
	fixture, err := startHTTPFixture(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		if servePRAccessFixtureRead(w, r) {
			return
		}
		if serveIssueCreationPolicyFixtureRead(w, r) || serveSponsorshipsFixtureRead(w, r) {
			return
		}
		if isMergeMethodsFixtureRead(r) {
			fmt.Fprint(w, mergeMethodsFixtureResponse)
			return
		}
		switch r.URL.Path {
		case "/repos/owner/current":
			if r.Method == "GET" {
				fmt.Fprint(w, planResponse)
			} else {
				t.Error("unexpected repository mutation")
				w.WriteHeader(400)
			}
			return
		case "/repos/owner/current/topics":
			fmt.Fprint(w, `{"names":[]}`)
			return
		case "/repos/owner/current/labels":
			fmt.Fprint(w, `[]`)
			return
		}
		if !strings.HasPrefix(r.URL.Path, "/repos/owner/current/rulesets") {
			t.Error("unexpected path", r.URL)
			w.WriteHeader(400)
			return
		}
		if r.Header.Get("X-GitHub-Api-Version") != "2026-03-10" {
			t.Error("Rulesets API version", r.Header)
		}
		if r.Method == "GET" {
			if r.URL.Query().Get("includes_parents") != "false" {
				t.Error("includes_parents", r.URL)
			}
			if r.URL.Path == "/repos/owner/current/rulesets" {
				items := []any{}
				for _, id := range []string{"7", "8", "10"} {
					if s, ok := states[id]; ok {
						items = append(items, map[string]any{"id": s["id"], "name": s["name"], "source_type": "Repository", "source": "OWNER/CURRENT"})
					}
				}
				items = append(items, map[string]any{"id": 9, "name": "push", "source_type": "Repository", "source": "owner/current"}, map[string]any{"id": 99, "name": "basic", "source_type": "Organization", "source": "owner"})
				json.NewEncoder(w).Encode(items)
				return
			}
			id := strings.TrimPrefix(r.URL.Path, "/repos/owner/current/rulesets/")
			if id == "9" {
				fmt.Fprint(w, `{"id":9,"target":"push","source_type":"Repository","source":"owner/current"}`)
				return
			}
			if state, ok := states[id]; ok {
				json.NewEncoder(w).Encode(state)
				return
			}
			t.Error("unexpected Ruleset read", id)
			w.WriteHeader(404)
			return
		}
		writes = append(writes, r.Method+" "+r.URL.Path)
		id := strings.TrimPrefix(r.URL.Path, "/repos/owner/current/rulesets/")
		if r.Method == "DELETE" {
			t.Error("unlisted Ruleset must never be deleted", id)
			delete(states, id)
			w.WriteHeader(204)
			return
		}
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Error(err)
		}
		if len(payload) != 6 || payload["source"] != nil || payload["id"] != nil {
			t.Error("unexpected payload", payload)
		}
		if r.Method == "POST" {
			id = "10"
		} else if r.Method != "PUT" || id != "7" {
			t.Error("unexpected mutation", r.Method, id)
		}
		if id == "7" {
			if payload["enforcement"] != "disabled" || len(payload["rules"].([]any)) != 0 || len(payload["bypass_actors"].([]any)) != 0 {
				t.Error("lost explicit removal", payload)
			}
			ref := payload["conditions"].(map[string]any)["ref_name"].(map[string]any)
			if !reflect.DeepEqual(ref["include"], []any{"~DEFAULT_BRANCH"}) {
				t.Error("lost omitted sibling", payload)
			}
		}
		payload["id"] = float64(10)
		if id == "7" {
			payload["id"] = float64(7)
		}
		payload["source_type"] = "Repository"
		payload["source"] = "owner/current"
		states[id] = payload
		json.NewEncoder(w).Encode(payload)
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
		result, err := executeCLI(binary, fixture.work, fixture.root, env, args, false)
		if err != nil || result.runErr != nil {
			t.Fatalf("%v: %v %s", args, err, result.stderr.String())
		}
		return result.stdout.String()
	}
	run("export", "--rulesets")
	data, err := os.ReadFile(filepath.Join(fixture.work, ".github/settings.yml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "rulesets:") || strings.Contains(string(data), "ruleset_name: push") {
		t.Fatal(string(data))
	}
	run("validate")
	if out := run("plan"); out != "No differences for managed repository settings.\n" {
		t.Fatal("export did not roundtrip", out)
	}
	// One listed ID owns its complete supported policy; omitted siblings remain untouched.
	desired := `# retained consumer comment
version: daiksud/gh-settings@v0
repository:
  settings:
    rulesets:
      - id: 7
        ruleset_name: renamed
        enforcement_status: disabled
        target_branches: {include_default_branch: true}
      - ruleset_name: new
        enforcement_status: active
        target_tags: {include_by_pattern: ['v*']}
        rules: {restrict_deletions: true}
`
	if err := writeSettingsFile(fixture.work, desired); err != nil {
		t.Fatal(err)
	}
	plan := run("plan")
	for _, shown := range []string{`update "basic"`, `create "new"`, `"id":7`, `"name":"renamed"`, `"rules":[]`} {
		if !strings.Contains(plan, shown) {
			t.Fatal("missing selected/default-derived change", shown, plan)
		}
	}
	if strings.Contains(plan, `"old"`) || strings.Contains(plan, "delete") && strings.Contains(plan, "removes repository protection") {
		t.Fatal("unlisted deletion planned", plan)
	}
	unchanged, _ := os.ReadFile(filepath.Join(fixture.work, ".github/settings.yml"))
	if string(unchanged) != desired {
		t.Fatal("plan wrote IDs")
	}
	report := filepath.Join(fixture.root, "rulesets-plan.json")
	run("action", "plan", report)
	reportData, err := os.ReadFile(report)
	if err != nil {
		t.Fatal(err)
	}
	var value map[string]any
	if err := json.Unmarshal(reportData, &value); err != nil {
		t.Fatal(err)
	}
	if value["outcome"] != "changed" || len(value["rulesets"].([]any)) != 2 || strings.Contains(string(reportData), `"old"`) {
		t.Fatal(string(reportData))
	}
	summaryPath := filepath.Join(fixture.root, "rulesets-summary.md")
	summaryEnv := append(fixtureEnv(fixture.cfg), "SETTINGS_ACTOR=user", "SETTINGS_TRIGGERING_ACTOR=user", "SETTINGS_EVENT=push", "SETTINGS_RUN_ID=123", "SETTINGS_RUN_ATTEMPT=1", "SETTINGS_WORKFLOW_REF=owner/current/.github/workflows/settings.yml@refs/heads/main", "SETTINGS_WORKFLOW_SHA="+strings.Repeat("a", 40), "SETTINGS_CONFIG_SHA="+strings.Repeat("b", 40), "SETTINGS_TARGET=owner/current", "SETTINGS_CLI_VERSION=dev", "SETTINGS_CLI_SHA256="+strings.Repeat("c", 64))
	result, err := executeCLI(binary, fixture.root, fixture.root, summaryEnv, []string{"action", "summary", report, summaryPath}, false)
	if err != nil || result.runErr != nil {
		t.Fatal(err, result.stderr.String())
	}
	summaryData, err := os.ReadFile(summaryPath)
	if err != nil || !strings.Contains(html.UnescapeString(string(summaryData)), `"id":7`) {
		t.Fatal(string(summaryData), err)
	}
	// Force local-mode writeback even when the acceptance harness itself runs in Actions.
	env = labelIDFixtureEnv(fixture.cfg, false)
	out := run("apply")
	if !strings.Contains(out, `Created rulesets "new": id 10`) {
		t.Fatal(out)
	}
	recorded, err := os.ReadFile(filepath.Join(fixture.work, ".github/settings.yml"))
	if err != nil || !strings.Contains(string(recorded), "id: 10") || !strings.Contains(string(recorded), "# retained consumer comment") {
		t.Fatal(string(recorded), err)
	}
	mu.Lock()
	got := append([]string{}, writes...)
	old := states["8"]
	mu.Unlock()
	if !reflect.DeepEqual(got, []string{"PUT /repos/owner/current/rulesets/7", "POST /repos/owner/current/rulesets"}) || old == nil || old["name"] != "old" || old["enforcement"] != "active" || len(old["rules"].([]any)) != 2 {
		t.Fatal("unlisted state changed", got, old)
	}
	if out := run("plan"); out != "No differences for managed repository settings.\n" {
		t.Fatal(out)
	}
	run("apply")
	mu.Lock()
	count := len(writes)
	mu.Unlock()
	if count != 2 {
		t.Fatal("repeat apply wrote", count)
	}
	if err := writeSettingsFile(fixture.work, repositoryInput("settings: {rulesets: []}\n")); err != nil {
		t.Fatal(err)
	}
	if out := run("plan"); out != "No managed repository settings.\n" {
		t.Fatal(out)
	}
	run("apply")
	mu.Lock()
	if len(writes) != 2 || len(states) != 3 {
		t.Fatal("empty selection changed remote membership", writes, states)
	}
	mu.Unlock()
	var saved map[string]any
	if err := yaml.Unmarshal(data, &saved); err != nil {
		t.Fatal(err)
	}
	exported := saved["repository"].(map[string]any)["settings"].(map[string]any)["rulesets"].([]any)
	wantExport := []any{
		map[string]any{"id": 7, "ruleset_name": "basic", "enforcement_status": "active", "target_branches": map[string]any{"include_default_branch": true, "include_all_branches": false, "include_by_pattern": []any{}, "exclude_by_pattern": []any{}}, "bypass_list": []any{}, "rules": map[string]any{"restrict_deletions": true, "block_force_pushes": true, "require_linear_history": false, "require_signed_commits": false}},
		map[string]any{"id": 8, "ruleset_name": "old", "enforcement_status": "active", "target_branches": map[string]any{"include_default_branch": true, "include_all_branches": false, "include_by_pattern": []any{}, "exclude_by_pattern": []any{}}, "bypass_list": []any{}, "rules": map[string]any{"restrict_deletions": true, "block_force_pushes": true, "require_linear_history": false, "require_signed_commits": false}},
	}
	if !reflect.DeepEqual(exported, wantExport) {
		t.Fatalf("exported=%v want=%v", exported, wantExport)
	}
}

func TestRulesetsFailureBoundaries(t *testing.T) {
	binary := acceptanceBinary(t)
	for _, tt := range []struct {
		name, command, detail, config string
		status                        int
	}{{"hidden bypass", "apply", strings.Replace(acceptanceRuleset, `"bypass_actors":[],`, "", 1), "[{id: 7, ruleset_name: basic, enforcement_status: disabled, target_branches: {include_default_branch: true}}]", 200}, {"unsupported rule", "apply", strings.Replace(acceptanceRuleset, `"type":"deletion"`, `"type":"merge_queue"`, 1), "[{id: 7, ruleset_name: basic, enforcement_status: disabled, target_branches: {include_default_branch: true}}]", 200}, {"unknown condition", "plan", strings.Replace(acceptanceRuleset, `"exclude":[]`, `"exclude":[],"future":true`, 1), "[{id: 7, ruleset_name: basic, enforcement_status: disabled, target_branches: {include_default_branch: true}}]", 200}, {"get redirect", "apply", acceptanceRuleset, "[{id: 7, ruleset_name: basic, enforcement_status: disabled, target_branches: {include_default_branch: true}}]", 302}, {"put failure stops later writes", "apply", acceptanceRuleset, `[{id: 7, ruleset_name: basic, enforcement_status: disabled, target_branches: {include_default_branch: true}}]`, 403}, {"invalid before reads", "apply", "", "null", 200}} {
		t.Run(tt.name, func(t *testing.T) {
			var mu sync.Mutex
			var requests []string
			fixture, err := startHTTPFixture(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				requests = append(requests, r.Method+" "+r.URL.Path)
				w.Header().Set("Content-Type", "application/json")
				if tt.status == 302 || r.Method == "PUT" {
					w.Header().Set("Location", "https://api.github.com/repos/wrong/target/rulesets")
					w.WriteHeader(tt.status)
					fmt.Fprint(w, `{"message":"denied"}`)
					return
				}
				if r.Method != "GET" {
					t.Error("unexpected write", r.Method)
					w.WriteHeader(400)
					return
				}
				if strings.HasSuffix(r.URL.Path, "/rulesets") {
					fmt.Fprint(w, `[{"id":7,"source_type":"Repository","source":"owner/current"},{"id":8,"source_type":"Repository","source":"owner/current"}]`)
				} else if strings.HasSuffix(r.URL.Path, "/8") {
					fmt.Fprint(w, strings.Replace(strings.Replace(acceptanceRuleset, `"id":7`, `"id":8`, 1), `"name":"basic"`, `"name":"old"`, 1))
				} else {
					fmt.Fprint(w, tt.detail)
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
			if err := writeSettingsFile(fixture.work, repositoryInput("settings: {rulesets: "+tt.config+"}\n")); err != nil {
				t.Fatal(err)
			}
			result, err := executeCLI(binary, fixture.work, fixture.root, env, []string{tt.command}, false)
			if err != nil || result.runErr == nil || result.stdout.Len() != 0 {
				t.Fatalf("%v %+v", err, result)
			}
			mu.Lock()
			defer mu.Unlock()
			if tt.config == "null" && len(requests) != 0 {
				t.Fatal(requests)
			}
			for _, r := range requests {
				if strings.HasPrefix(r, "DELETE") || strings.HasPrefix(r, "POST") {
					t.Fatal("later write", requests)
				}
			}
			if tt.status == 302 && len(requests) != 1 {
				t.Fatal("followed redirect", requests)
			}
		})
	}
}
func TestRulesetsOptedExportFailurePreservesFile(t *testing.T) {
	binary := acceptanceBinary(t)
	fixture, err := startHTTPFixture(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if servePRAccessFixtureRead(w, r) {
			return
		}
		if serveIssueCreationPolicyFixtureRead(w, r) || serveSponsorshipsFixtureRead(w, r) {
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
			fmt.Fprint(w, `[{"id":7,"source_type":"Repository","source":"owner/current"}]`)
		default:
			fmt.Fprint(w, strings.Replace(acceptanceRuleset, `"bypass_actors":[],`, "", 1))
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
	before := "version: daiksud/gh-settings@v0\n"
	writeSettingsFile(fixture.work, before)
	result, err := executeCLI(binary, fixture.work, fixture.root, env, []string{"export", "--rulesets"}, false)
	if err != nil || result.runErr == nil || result.stdout.Len() != 0 {
		t.Fatal(err, result)
	}
	after, _ := os.ReadFile(filepath.Join(fixture.work, ".github/settings.yml"))
	if string(after) != before {
		t.Fatal("failed export replaced file")
	}
}
