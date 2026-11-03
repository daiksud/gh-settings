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

func TestRulesetEvaluateAcrossExecutableCommands(t *testing.T) {
	binary := acceptanceBinary(t)
	var mu sync.Mutex
	var state map[string]any
	json.Unmarshal([]byte(acceptanceRuleset), &state)
	state["enforcement"] = "evaluate"
	var calls []string
	stale := false
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
			fmt.Fprint(w, `[{"id":7,"source_type":"Repository","source":"owner/current"}]`)
		case "/repos/owner/current/rulesets/7":
			calls = append(calls, r.Method)
			if r.Method == "GET" {
				json.NewEncoder(w).Encode(state)
				return
			}
			if r.Method != "PUT" {
				t.Error(r.Method)
				w.WriteHeader(400)
				return
			}
			var payload map[string]any
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Error(err)
			}
			if len(payload) != 6 || payload["enforcement"] != "evaluate" {
				t.Error(payload)
			}
			payload["id"], payload["source_type"], payload["source"] = 7, "Repository", "owner/current"
			if !stale {
				state = payload
			}
			json.NewEncoder(w).Encode(payload)
		default:
			t.Error(r.Method, r.URL)
			w.WriteHeader(400)
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
		result, err := executeCLI(binary, fixture.work, fixture.root, env, args, false)
		if err != nil || result.runErr != nil || result.stderr.Len() != 0 {
			t.Fatalf("%v: %v %v %s", args, err, result.runErr, result.stderr.String())
		}
		return result.stdout.String()
	}
	run("export", "--rulesets")
	data, err := os.ReadFile(filepath.Join(fixture.work, ".github/settings.yml"))
	if err != nil || !strings.Contains(string(data), "enforcement_status: evaluate") {
		t.Fatal(string(data), err)
	}
	if err := writeSettingsFile(fixture.work, repositoryInput("settings: {rulesets: [{id: 7, ruleset_name: basic, enforcement_status: evaluate, target_branches: {include_default_branch: true}, rules: {restrict_deletions: true, block_force_pushes: true}}]}\n")); err != nil {
		t.Fatal(err)
	}
	run("validate")
	if out := run("plan"); out != "No differences for managed repository settings.\n" {
		t.Fatal(out)
	}
	for _, prior := range []string{"active", "disabled"} {
		mu.Lock()
		state["enforcement"] = prior
		calls = nil
		mu.Unlock()
		if out := run("plan"); !strings.Contains(out, "evaluate") || strings.Contains(out, "normalized") {
			t.Fatal(out)
		}
		mu.Lock()
		calls = nil
		mu.Unlock()
		run("apply")
		mu.Lock()
		sequence := strings.Join(calls, ",")
		mu.Unlock()
		if !strings.Contains(sequence, "PUT,GET") {
			t.Fatal("no independent GET", sequence)
		}
		if out := run("plan"); out != "No differences for managed repository settings.\n" {
			t.Fatal(out)
		}
		mu.Lock()
		calls = nil
		mu.Unlock()
		run("apply")
		mu.Lock()
		sequence = strings.Join(calls, ",")
		mu.Unlock()
		if strings.Contains(sequence, "PUT") {
			t.Fatal("repeat mutated", sequence)
		}
	}
	mu.Lock()
	state["enforcement"] = "active"
	stale = true
	calls = nil
	mu.Unlock()
	result, err := executeCLI(binary, fixture.work, fixture.root, env, []string{"apply"}, false)
	if err != nil || result.runErr == nil || !strings.Contains(result.stderr.String(), "writes may already have applied") {
		t.Fatal(err, result.runErr, result.stderr.String())
	}
}
