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

func TestCodeQLConfigurationExportPlanApplyAndDisabledGuard(t *testing.T) {
	binary := acceptanceBinary(t)
	var mu sync.Mutex
	raw := map[string]any{"state": "configured", "languages": []string{"go"}, "query_suite": "default", "runner_type": "standard", "runner_label": nil, "threat_model": "remote_and_local"}
	reads, writes := 0, 0
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
			if r.Method != "GET" {
				t.Error("repository write")
				w.WriteHeader(400)
				return
			}
			var base map[string]any
			json.Unmarshal([]byte(planResponse), &base)
			base["visibility"] = "public"
			base["owner"] = map[string]string{"type": "User"}
			json.NewEncoder(w).Encode(base)
			return
		case "/repos/owner/current/topics":
			fmt.Fprint(w, `{"names":[]}`)
			return
		case "/repos/owner/current/labels":
			fmt.Fprint(w, `[]`)
			return
		case "/repos/owner/current/code-scanning/default-setup":
			if r.Method == "GET" {
				reads++
				json.NewEncoder(w).Encode(raw)
				return
			}
			var payload map[string]any
			json.NewDecoder(r.Body).Decode(&payload)
			if r.Method != "PATCH" || len(payload) != 4 || payload["query_suite"] != "extended" || payload["runner_type"] != "labeled" || payload["runner_label"] != "scan-pool" || payload["state"] != nil || payload["threat_model"] != nil {
				t.Error("unexpected configuration payload", payload)
				w.WriteHeader(400)
				return
			}
			langs, ok := payload["languages"].([]any)
			if !ok || len(langs) != 2 || langs[0] != "actions" || langs[1] != "go" {
				t.Error(langs)
			}
			writes++
			for k, v := range payload {
				raw[k] = v
			}
			w.WriteHeader(202)
			fmt.Fprint(w, `{"run_id":42}`)
			return
		}
		t.Error("unrelated endpoint", r.Method, r.URL)
		w.WriteHeader(400)
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
		if err != nil || r.runErr != nil || r.stderr.Len() != 0 {
			t.Fatal(args, err, r.stderr.String())
		}
		return r.stdout.String()
	}
	run("export")
	mu.Lock()
	n := reads
	mu.Unlock()
	if n != 0 {
		t.Fatal("ordinary export acquired config")
	}
	run("export", "--security", "codeql-configuration")
	run("validate")
	data, _ := os.ReadFile(filepath.Join(fixture.work, ".github/settings.yml"))
	if !strings.Contains(string(data), "query_suite: default") || strings.Contains(string(data), "default_setup:") || strings.Contains(string(data), "threat_model:") {
		t.Fatal(string(data))
	}
	// Excluded Preview declarations fail before any remote acquisition or write.
	for _, value := range []string{"remote_and_local_sources", "remote_sources", "remote_and_local", "null"} {
		if err := writeSettingsFile(fixture.work, repositoryInput("settings: {advanced_security: {code_security: {codeql_analysis: {threat_model: "+value+"}}}}\n")); err != nil {
			t.Fatal(err)
		}
		mu.Lock()
		beforeReads, beforeWrites := reads, writes
		mu.Unlock()
		r, err := executeCLI(binary, fixture.work, fixture.root, env, []string{"apply"}, false)
		if err != nil || r.runErr == nil || !strings.Contains(r.stderr.String(), "unknown") {
			t.Fatal(value, err, r.stderr.String())
		}
		mu.Lock()
		unchanged := reads == beforeReads && writes == beforeWrites
		mu.Unlock()
		if !unchanged {
			t.Fatal("excluded declaration contacted API")
		}
	}
	input := repositoryInput("settings: {advanced_security: {code_security: {codeql_analysis: {languages: [github_actions, go], query_suite: extended, runner: {type: labeled_runner, label: scan-pool}}}}}\n")
	if err := writeSettingsFile(fixture.work, input); err != nil {
		t.Fatal(err)
	}
	plan := run("plan")
	if !strings.Contains(plan, "codeql_analysis.languages:") || !strings.Contains(plan, "codeql_analysis.runner:") {
		t.Fatal(plan)
	}
	report := filepath.Join(fixture.root, "codeql-configuration.json")
	run("action", "plan", report)
	data, _ = os.ReadFile(report)
	var typed map[string]any
	json.Unmarshal(data, &typed)
	if typed["managed"] != float64(3) || len(typed["codeql_configuration"].([]any)) != 3 {
		t.Fatal(string(data))
	}
	run("apply")
	run("apply")
	if got := run("plan"); got != "No differences for managed repository settings.\n" {
		t.Fatal(got)
	}
	mu.Lock()
	if writes != 1 || raw["state"] != "configured" || raw["threat_model"] != "remote_and_local" {
		t.Fatal(raw, writes)
	}
	raw["state"] = "not-configured"
	mu.Unlock()
	r, err := executeCLI(binary, fixture.work, fixture.root, env, []string{"apply"}, false)
	if err != nil || r.runErr == nil || !strings.Contains(r.stderr.String(), "already configured") {
		t.Fatal(err, r.stderr.String())
	}
	mu.Lock()
	defer mu.Unlock()
	if writes != 1 {
		t.Fatal("enabled prerequisite", writes)
	}
}
