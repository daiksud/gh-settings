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
)

func TestLabelsPruneExecutableLifecycle(t *testing.T) {
	var mu sync.Mutex
	states := map[int]map[string]any{7: {"id": 7, "name": "old", "color": "abcdef", "description": "old description"}, 8: {"id": 8, "name": "dependencies", "color": "abcdef", "description": ""}}
	writes := []string{}
	reads := 0
	denyNew := false
	fixture, err := startHTTPFixture(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		if r.Method == "GET" {
			reads++
		}
		if r.URL.Path == "/repos/owner/current/labels" {
			switch r.Method {
			case "GET":
				items := []any{}
				for _, id := range []int{7, 8, 9} {
					if state, ok := states[id]; ok {
						items = append(items, state)
					}
				}
				json.NewEncoder(w).Encode(items)
			case "POST":
				var payload map[string]any
				json.NewDecoder(r.Body).Decode(&payload)
				payload["id"] = 9
				states[9] = payload
				writes = append(writes, "create:new")
				w.WriteHeader(201)
				json.NewEncoder(w).Encode(payload)
			default:
				t.Error("unexpected collection mutation", r.Method)
				w.WriteHeader(400)
			}
			return
		}
		name := strings.TrimPrefix(r.URL.Path, "/repos/owner/current/labels/")
		var id int
		for candidate, state := range states {
			if state["name"] == name {
				id = candidate
			}
		}
		if id == 0 {
			t.Error("wrong label identity", r.Method, r.URL)
			w.WriteHeader(404)
			return
		}
		switch r.Method {
		case "GET":
			json.NewEncoder(w).Encode(states[id])
		case "PATCH":
			var payload map[string]any
			json.NewDecoder(r.Body).Decode(&payload)
			states[id]["name"] = payload["new_name"]
			states[id]["color"] = payload["color"]
			states[id]["description"] = payload["description"]
			writes = append(writes, "update:old")
			json.NewEncoder(w).Encode(states[id])
		case "DELETE":
			writes = append(writes, "delete:"+name)
			if denyNew && id == 9 {
				w.WriteHeader(403)
				fmt.Fprint(w, `{"message":"denied"}`)
				return
			}
			delete(states, id)
			w.WriteHeader(204)
		default:
			t.Error("unexpected request", r.Method, r.URL)
			w.WriteHeader(400)
		}
	}, false, "/labels", "/labels/")
	if err != nil {
		t.Fatal(err)
	}
	defer fixture.close()
	// This lifecycle exercises local created-ID writeback, including when run by CI.
	env := append(labelIDFixtureEnv(fixture.cfg, false), "SETTINGS_PRUNE=labels", "SETTINGS_TARGET=owner/current", "SETTINGS_ACTOR=user", "SETTINGS_TRIGGERING_ACTOR=user", "SETTINGS_EVENT=push", "SETTINGS_RUN_ID=123", "SETTINGS_RUN_ATTEMPT=1", "SETTINGS_WORKFLOW_REF=owner/current/.github/workflows/settings.yml@refs/heads/main", "SETTINGS_WORKFLOW_SHA="+strings.Repeat("a", 40), "SETTINGS_CONFIG_SHA="+strings.Repeat("b", 40), "SETTINGS_CLI_VERSION=dev", "SETTINGS_CLI_SHA256="+strings.Repeat("c", 64))
	if err := setupOrdinaryOrigin(fixture.work, env); err != nil {
		t.Fatal(err)
	}
	input := repositoryInput(`labels: [{id: 7, name: kept, color: '#abcdef'}, {name: new, color: '#abcdef'}]` + "\n")
	if err := writeSettingsFile(fixture.work, input); err != nil {
		t.Fatal(err)
	}
	binary := acceptanceBinary(t)
	run := func(success bool, args ...string) string {
		t.Helper()
		r, err := executeCLI(binary, fixture.work, fixture.root, env, args, false)
		if err != nil || (r.runErr == nil) != success {
			t.Fatalf("args=%v err=%v run=%v out=%s stderr=%s", args, err, r.runErr, r.stdout.String(), r.stderr.String())
		}
		return r.stdout.String() + withoutExportInfo(r.stderr.String())
	}
	for _, value := range []string{"all", "labels,all", "labels,", "environments"} {
		run(false, "apply", "--prune", value)
	}
	mu.Lock()
	if reads != 0 || len(writes) != 0 {
		t.Fatal("invalid prune touched remote")
	}
	mu.Unlock()
	plan := run(true, "plan", "--prune", "labels,rulesets")
	if !strings.Contains(plan, `delete "dependencies"`) || !strings.Contains(plan, "issue/PR associations") || !strings.Contains(plan, "Dependabot") {
		t.Fatal(plan)
	}
	mu.Lock()
	if len(writes) != 0 {
		t.Fatal("plan wrote")
	}
	mu.Unlock()
	for _, mode := range []string{"detailed", "status-only"} {
		report := filepath.Join(fixture.root, "label-"+mode+".json")
		summary := filepath.Join(fixture.root, "label-"+mode+".md")
		planArgs := []string{"action", "plan", report}
		summaryArgs := []string{"action", "summary", report, summary}
		if mode == "status-only" {
			planArgs = append(planArgs, mode)
			summaryArgs = append(summaryArgs, mode)
		}
		run(true, planArgs...)
		run(true, summaryArgs...)
		data, err := os.ReadFile(summary)
		if err != nil {
			t.Fatal(err)
		}
		if mode == "status-only" {
			if strings.Contains(string(data), "dependencies") || !strings.Contains(string(data), "changed") {
				t.Fatal(string(data))
			}
		} else if !strings.Contains(string(data), "delete") || !strings.Contains(string(data), "issue/PR associations") {
			t.Fatal(string(data))
		}
	}

	run(true, "apply", "--prune", "labels,rulesets")
	mu.Lock()
	if strings.Join(writes, ",") != "update:old,create:new,delete:dependencies" || len(states) != 2 || states[9] == nil || states[7]["name"] != "kept" {
		t.Fatal(writes, states)
	}
	mu.Unlock()
	if out := run(true, "plan", "--prune", "labels"); !strings.Contains(out, "No differences") {
		t.Fatal(out)
	}
	run(true, "apply", "--prune", "labels")
	if err := writeSettingsFile(fixture.work, repositoryInput("labels: []\n")); err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	denyNew = true
	mu.Unlock()
	out := run(false, "apply", "--prune", "labels")
	if !strings.Contains(out, "earlier successful writes remain applied") {
		t.Fatal(out)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(states) != 1 || states[9] == nil {
		t.Fatal("partial deletion not retained", states)
	}
}
