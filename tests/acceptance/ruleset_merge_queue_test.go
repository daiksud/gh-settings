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

func TestRulesetMergeQueueExportValidatePlanApplyConverges(t *testing.T) {
	binary := acceptanceBinary(t)
	data, err := os.ReadFile("../fixtures/ruleset-merge-queue.json")
	if err != nil {
		t.Fatal(err)
	}
	var state map[string]any
	if err := json.Unmarshal(data, &state); err != nil {
		t.Fatal(err)
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
		if r.Method == "GET" {
			switch r.URL.Path {
			case "/repos/owner/current":
				fmt.Fprint(w, planResponse)
			case "/repos/owner/current/topics":
				fmt.Fprint(w, `{"names":[]}`)
			case "/repos/owner/current/labels":
				fmt.Fprint(w, `[]`)
			case "/repos/owner/current/rulesets":
				fmt.Fprint(w, `[{"id":26024001,"source_type":"Repository","source":"owner/current"}]`)
			case "/repos/owner/current/rulesets/26024001":
				if writes > 0 {
					readbacks++
				}
				json.NewEncoder(w).Encode(state)
			default:
				t.Error("unexpected GET", r.URL)
				w.WriteHeader(404)
			}
			return
		}
		if r.Method != "PUT" || r.URL.Path != "/repos/owner/current/rulesets/26024001" {
			t.Error("unexpected mutation", r.Method, r.URL)
			w.WriteHeader(400)
			return
		}
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Error(err)
		}
		rules := payload["rules"].([]any)
		if len(rules) != 3 {
			t.Error("lost companion protections", payload)
		}
		queue := acceptanceRuleParams(t, payload, "merge_queue")
		if queue["max_entries_to_build"] != float64(0) || queue["merge_method"] != "SQUASH" || queue["grouping_strategy"] != "ALLGREEN" {
			t.Error("incorrect queue payload", queue)
		}
		writes++
		payload["id"], payload["source"], payload["source_type"] = float64(26024001), "owner/current", "Repository"
		state = payload
		json.NewEncoder(w).Encode(state)
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
		out, err := executeCLI(binary, fixture.work, fixture.root, env, args, false)
		if err != nil || out.runErr != nil {
			t.Fatalf("%v: %v %s", args, err, out.stderr.String())
		}
		return out.stdout.String()
	}
	run("export", "--rulesets")
	run("validate")
	if out := run("plan"); out != "No differences for managed repository settings.\n" {
		t.Fatal(out)
	}
	path := filepath.Join(fixture.work, ".github/settings.yml")
	exported, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	// Focus apply on the selected Ruleset, preserving the literal exported policy.
	start := strings.Index(string(exported), "    rulesets:")
	if start < 0 {
		t.Fatal(string(exported))
	}
	focused := "version: daiksud/gh-settings@v0\nrepository:\n  settings:\n" + string(exported)[start:]
	focused = strings.Replace(focused, "build_concurrency: 5", "build_concurrency: 0", 1)
	if err := writeSettingsFile(fixture.work, focused); err != nil {
		t.Fatal(err)
	}
	run("validate")
	if out := run("plan"); !strings.Contains(out, `"build_concurrency":0`) {
		t.Fatal(out)
	}
	run("apply")
	mu.Lock()
	firstWrites, firstReads := writes, readbacks
	mu.Unlock()
	if firstWrites != 1 || firstReads != 1 {
		t.Fatal("missing independent readback", firstWrites, firstReads)
	}
	if out := run("plan"); out != "No differences for managed repository settings.\n" {
		t.Fatal(out)
	}
	run("apply")
	mu.Lock()
	defer mu.Unlock()
	if writes != 1 {
		t.Fatal("repeat mutation", writes)
	}
}
