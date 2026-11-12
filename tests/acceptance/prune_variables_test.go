//go:build acceptance

package acceptance

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"
)

func TestVariablePruneExecutableScopesSummaryAndPartialFailure(t *testing.T) {
	const actionBase = "/repos/owner/current/actions/variables"
	const agentBase = "/repos/owner/current/agents/variables"
	const envBase = "/repos/owner/current/environments/prod%2Fblue/variables"
	const parent = "/repos/owner/current/environments/prod%2Fblue"
	var mu sync.Mutex
	states := map[string]map[string]string{actionBase: {"OLD": "actions-old", "KEEP": "actions-before"}, agentBase: {"OLD": "agents-old"}, envBase: {"OLD": "environment-old", "KEEP": "environment-before"}}
	var writes []string
	requests := 0
	environmentReads := 0
	denyAgent := true
	fixture, err := startHTTPFixture(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		requests++
		w.Header().Set("Content-Type", "application/json")
		path := r.URL.EscapedPath()
		if path == parent {
			environmentReads++
			if r.Method != "GET" {
				t.Error("Environment policy mutated")
				w.WriteHeader(400)
				return
			}
			fmt.Fprint(w, `{"id":42,"name":"prod/blue"}`)
			return
		}
		base := path
		name := ""
		for candidate := range states {
			if strings.HasPrefix(path, candidate+"/") {
				base = candidate
				name = strings.TrimPrefix(path, candidate+"/")
				break
			}
		}
		values, ok := states[base]
		if !ok {
			t.Error("unselected or secret API", r.Method, path)
			w.WriteHeader(400)
			return
		}
		switch r.Method {
		case "GET":
			if name != "" {
				value, exists := values[name]
				if !exists {
					w.WriteHeader(404)
					fmt.Fprint(w, `{"message":"not found"}`)
					return
				}
				json.NewEncoder(w).Encode(map[string]string{"name": name, "value": value})
				return
			}
			if r.URL.Query().Get("per_page") != "30" {
				t.Error("wrong page limit", r.URL)
			}
			names := []string{}
			for name := range values {
				names = append(names, name)
			}
			sort.Strings(names)
			items := []map[string]string{}
			for _, name := range names {
				items = append(items, map[string]string{"name": name, "value": values[name]})
			}
			json.NewEncoder(w).Encode(map[string]any{"total_count": len(items), "variables": items})
		case "POST", "PATCH":
			var payload map[string]string
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Error(err)
			}
			if r.Method == "POST" {
				name = payload["name"]
				if _, exists := values[name]; exists {
					t.Error("created variable twice", name)
				}
				w.WriteHeader(201)
			} else {
				if _, present := payload["name"]; present {
					t.Error("PATCH renamed variable")
				}
				w.WriteHeader(204)
			}
			values[name] = payload["value"]
			writes = append(writes, r.Method+" "+base+"/"+name)
		case "DELETE":
			writes = append(writes, "DELETE "+base+"/"+name)
			if base == agentBase && denyAgent {
				w.WriteHeader(403)
				fmt.Fprint(w, `{"message":"fixture denied"}`)
				return
			}
			if _, present := values[name]; !present {
				t.Error("deleted absent variable", path)
			}
			delete(values, name)
			w.WriteHeader(204)
		default:
			t.Error("unexpected method", r.Method)
			w.WriteHeader(400)
		}
	}, false)
	if err != nil {
		t.Fatal(err)
	}
	defer fixture.close()
	scopes := "actions-variables,agents-variables,environment-variables"
	env := append(exportFixtureEnv(fixture.cfg), "SETTINGS_PRUNE="+scopes, "SETTINGS_TARGET=owner/current", "SETTINGS_ACTOR=user", "SETTINGS_TRIGGERING_ACTOR=user", "SETTINGS_EVENT=push", "SETTINGS_RUN_ID=123", "SETTINGS_RUN_ATTEMPT=1", "SETTINGS_WORKFLOW_REF=owner/current/.github/workflows/settings.yml@refs/heads/main", "SETTINGS_WORKFLOW_SHA="+strings.Repeat("a", 40), "SETTINGS_CONFIG_SHA="+strings.Repeat("b", 40), "SETTINGS_CLI_VERSION=dev", "SETTINGS_CLI_SHA256="+strings.Repeat("c", 64))
	if err := setupOrdinaryOrigin(fixture.work, env); err != nil {
		t.Fatal(err)
	}
	config := repositoryInput(`settings:
  actions:
    variables: [{name: KEEP, value: actions-after}, {name: NEW, value: created}]
  agents:
    variables: []
  environments:
    - name: prod/blue
      variables: [{name: KEEP, value: environment-after}]
`)
	if err := writeSettingsFile(fixture.work, config); err != nil {
		t.Fatal(err)
	}
	binary := acceptanceBinary(t)
	run := func(success bool, args ...string) string {
		t.Helper()
		r, err := executeCLI(binary, fixture.work, fixture.root, env, args, false)
		if err != nil || (r.runErr == nil) != success {
			t.Fatalf("args=%v err=%v run=%v out=%s stderr=%s", args, err, r.runErr, r.stdout.String(), r.stderr.String())
		}
		return r.stdout.String() + r.stderr.String()
	}
	for _, scope := range []string{"all", "environments", "actions-policies", "actions-variables,,agents-variables"} {
		run(false, "apply", "--prune", scope)
	}
	mu.Lock()
	if requests != 0 {
		t.Fatal("invalid option reached remote")
	}
	mu.Unlock()
	plan := run(true, "plan", "--prune", scopes)
	if strings.Count(plan, `delete "OLD"`) != 3 || strings.Count(plan, "absent") != 3 {
		t.Fatal(plan)
	}
	for _, mode := range []string{"detailed", "status-only"} {
		report := filepath.Join(fixture.root, "variable-"+mode+".json")
		summary := filepath.Join(fixture.root, "variable-"+mode+".md")
		args := []string{"action", "plan", report}
		summaryArgs := []string{"action", "summary", report, summary}
		if mode == "status-only" {
			args = append(args, mode)
			summaryArgs = append(summaryArgs, mode)
		}
		run(true, args...)
		run(true, summaryArgs...)
		data, err := os.ReadFile(summary)
		if err != nil {
			t.Fatal(err)
		}
		if mode == "status-only" {
			if strings.Contains(string(data), "OLD") || strings.Contains(string(data), "environment-old") {
				t.Fatal(string(data))
			}
		} else if strings.Count(string(data), "absent") != 3 {
			t.Fatal(string(data))
		}
	}
	mu.Lock()
	if len(writes) != 0 {
		t.Fatal("plan/report wrote")
	}
	mu.Unlock()
	out := run(false, "apply", "--prune", scopes)
	if !strings.Contains(out, "agents.variables") || !strings.Contains(out, "earlier successful writes remain applied") {
		t.Fatal(out)
	}
	mu.Lock()
	if states[envBase]["KEEP"] != "environment-after" || len(states[envBase]) != 1 || states[actionBase]["NEW"] != "created" || len(states[actionBase]) != 2 || states[agentBase]["OLD"] != "agents-old" {
		t.Fatal(states, writes)
	}
	denyAgent = false
	mu.Unlock()
	run(true, "apply", "--prune", scopes)
	if out := run(true, "plan", "--prune", scopes); !strings.Contains(out, "No differences") {
		t.Fatal(out)
	}
	mu.Lock()
	if len(states[agentBase]) != 0 {
		t.Fatal(states)
	}
	before := environmentReads
	mu.Unlock()
	if err := writeSettingsFile(fixture.work, repositoryInput("settings: {actions: {variables: []}}\n")); err != nil {
		t.Fatal(err)
	}
	run(true, "apply", "--prune", scopes)
	mu.Lock()
	defer mu.Unlock()
	if len(states[actionBase]) != 0 || len(states[envBase]) != 1 || environmentReads != before {
		t.Fatal("omitted Environment variables were managed", states, environmentReads, before)
	}
}
