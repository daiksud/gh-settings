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

func TestCodeQLDefaultSetupExportPlanApply(t *testing.T) {
	binary := acceptanceBinary(t)
	var mu sync.Mutex
	enabled := false
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
				t.Error("repository mutated")
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
				state := "not-configured"
				if enabled {
					state = "configured"
				}
				fmt.Fprintf(w, `{"state":%q,"query_suite":"extended","languages":["go"]}`, state)
				return
			}
			var payload map[string]any
			json.NewDecoder(r.Body).Decode(&payload)
			if r.Method != "PATCH" || len(payload) != 1 || (payload["state"] != "configured" && payload["state"] != "not-configured") {
				t.Error("unexpected payload", payload)
				w.WriteHeader(400)
				return
			}
			writes++
			enabled = payload["state"] == "configured"
			w.WriteHeader(202)
			fmt.Fprint(w, `{"run_id":42}`)
			return
		}
		t.Error("unrelated endpoint", r.Method, r.URL)
		w.WriteHeader(400)
	}, false, "/code-scanning/default-setup")
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
		if err != nil || r.runErr != nil || hasNonInfoDiagnostic(r.stderr.String()) {
			t.Fatal(args, err, r.stderr.String())
		}
		return r.stdout.String()
	}
	run("export")
	mu.Lock()
	n := reads
	mu.Unlock()
	if n == 0 {
		t.Fatal("ordinary export did not acquire CodeQL")
	}
	run("export")
	run("validate")
	data, _ := os.ReadFile(filepath.Join(fixture.work, ".github/settings.yml"))
	if !strings.Contains(string(data), "default_setup: false") {
		t.Fatal(string(data))
	}
	for _, desired := range []bool{true, false} {
		input := repositoryInput(fmt.Sprintf("settings: {advanced_security: {code_security: {codeql_analysis: {default_setup: %t}}}}\n", desired))
		if err := writeSettingsFile(fixture.work, input); err != nil {
			t.Fatal(err)
		}
		plan := run("plan")
		if !strings.Contains(plan, "repository.settings.advanced_security.code_security.codeql_analysis.default_setup:") {
			t.Fatal(plan)
		}
		report := filepath.Join(fixture.root, "codeql-plan.json")
		run("action", "plan", report)
		data, _ := os.ReadFile(report)
		var typed map[string]any
		json.Unmarshal(data, &typed)
		if typed["managed"] != float64(1) || len(typed["security"].([]any)) != 1 {
			t.Fatal(string(data))
		}
		run("apply")
		run("apply")
		if got := run("plan"); got != "No differences for managed repository settings.\n" {
			t.Fatal(got)
		}
	}
	mu.Lock()
	defer mu.Unlock()
	if writes != 2 || enabled {
		t.Fatal(writes, enabled)
	}
}
