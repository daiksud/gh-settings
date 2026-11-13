//go:build acceptance

package acceptance

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
)

func TestCodeQLThreatModelProcess(t *testing.T) {
	binary := acceptanceBinary(t)
	var mu sync.Mutex
	raw := map[string]any{"state": "configured", "languages": []string{"csharp"}, "query_suite": "default", "runner_type": "standard", "runner_label": nil, "threat_model": "remote"}
	wantRaw, mode := "remote_and_local", "ok"
	var methods []string
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
				t.Error("repository mutation")
				w.WriteHeader(400)
				return
			}
			var base map[string]any
			json.Unmarshal([]byte(planResponse), &base)
			base["visibility"], base["owner"] = "public", map[string]string{"type": "User"}
			json.NewEncoder(w).Encode(base)
		case "/repos/owner/current/topics":
			fmt.Fprint(w, `{"names":[]}`)
		case "/repos/owner/current/labels":
			fmt.Fprint(w, `[]`)
		case "/repos/owner/current/code-scanning/default-setup":
			methods = append(methods, r.Method)
			if r.Method == "GET" {
				json.NewEncoder(w).Encode(raw)
				return
			}
			var payload map[string]any
			if r.Method != "PATCH" || json.NewDecoder(r.Body).Decode(&payload) != nil || !reflect.DeepEqual(payload, map[string]any{"threat_model": wantRaw}) {
				t.Error("unexpected payload", payload)
				w.WriteHeader(400)
				return
			}
			if mode != "mismatch" {
				raw["threat_model"] = wantRaw
			}
			w.WriteHeader(202)
			fmt.Fprint(w, `{"run_id":42}`)
		default:
			t.Error("unexpected endpoint", r.Method, r.URL)
			w.WriteHeader(400)
		}
	}, false, "/code-scanning/default-setup", "/labels", "/topics")
	if err != nil {
		t.Fatal(err)
	}
	defer fixture.close()
	env := append(exportFixtureEnv(fixture.cfg), "SETTINGS_TARGET=owner/current")
	if err := setupOrdinaryOrigin(fixture.work, env); err != nil {
		t.Fatal(err)
	}
	write := func(value, extra string) {
		t.Helper()
		if err := writeSettingsFile(fixture.work, repositoryInput("settings: {advanced_security: {code_security: {codeql_analysis: {threat_model: "+value+extra+"}}}}\n")); err != nil {
			t.Fatal(err)
		}
	}
	run := func(fail bool, args ...string) (string, string) {
		t.Helper()
		r, err := executeCLI(binary, fixture.work, fixture.root, env, args, false)
		if err != nil || (r.runErr != nil) != fail || (!fail && hasNonInfoDiagnostic(r.stderr.String())) {
			t.Fatal(args, err, r.runErr, r.stdout.String(), r.stderr.String())
		}
		return r.stdout.String(), r.stderr.String()
	}
	expect := func(want ...string) {
		t.Helper()
		mu.Lock()
		defer mu.Unlock()
		if !reflect.DeepEqual(methods, want) {
			t.Fatal(methods, want)
		}
		methods = nil
	}
	readFile := func() string {
		t.Helper()
		b, err := os.ReadFile(filepath.Join(fixture.work, ".github/settings.yml"))
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	write("remote_and_local_sources", "")
	out, _ := run(false, "plan")
	if !strings.Contains(out, `codeql_analysis.threat_model: "remote_sources" → "remote_and_local_sources"`) {
		t.Fatal(out)
	}
	expect("GET")
	report := filepath.Join(fixture.root, "threat.json")
	run(false, "action", "plan", report, "status-only")
	expect("GET")
	b, err := os.ReadFile(report)
	var wire struct {
		Managed int `json:"managed"`
		Changes []struct {
			Field  string `json:"field"`
			Before string `json:"before"`
			After  string `json:"after"`
		} `json:"codeql_configuration"`
	}
	if err != nil || json.Unmarshal(b, &wire) != nil || wire.Managed != 1 || len(wire.Changes) != 1 || wire.Changes[0].Field != "threat_model" || wire.Changes[0].Before != "remote_sources" || wire.Changes[0].After != "remote_and_local_sources" {
		t.Fatal(string(b), err)
	}
	summary := filepath.Join(fixture.root, "summary.md")
	run(false, "action", "summary", report, summary, "status-only")
	expect()
	b, err = os.ReadFile(summary)
	if err != nil || strings.Contains(string(b), "threat_model") || strings.Contains(string(b), "remote_sources") || strings.Contains(string(b), "remote_and_local_sources") {
		t.Fatal(string(b), err)
	}
	run(false, "apply")
	expect("GET", "GET", "PATCH", "GET")
	run(false, "apply")
	expect("GET")
	mu.Lock()
	if raw["query_suite"] != "default" || raw["runner_type"] != "standard" || raw["runner_label"] != nil || !reflect.DeepEqual(raw["languages"], []string{"csharp"}) {
		t.Fatal(raw)
	}
	mu.Unlock()
	run(false, "export")
	expect("GET")
	if !strings.Contains(readFile(), "threat_model: remote_and_local_sources") {
		t.Fatal(readFile())
	}
	// Unknown details retain acquired prior intent; they never become remote-only.
	mu.Lock()
	raw["threat_model"] = "future"
	mu.Unlock()
	_, info := run(false, "export")
	expect("GET")
	if !strings.Contains(info, "codeql_analysis") || !strings.Contains(readFile(), "threat_model: remote_and_local_sources") {
		t.Fatal(info, readFile())
	}
	// Threat-only intent must not proceed to the separately declared setup read
	// after a mismatching independent readback.
	write("remote_sources", ", default_setup: true")
	mu.Lock()
	raw["threat_model"], wantRaw, mode = "remote_and_local", "remote", "mismatch"
	mu.Unlock()
	_, diagnostic := run(true, "apply")
	expect("GET", "GET", "PATCH", "GET")
	if !strings.Contains(diagnostic, "may already have started") {
		t.Fatal(diagnostic)
	}
	write("remote_sources", "")
	mu.Lock()
	mode = "ok"
	mu.Unlock()
	run(false, "apply")
	expect("GET", "GET", "PATCH", "GET")
	run(false, "apply")
	expect("GET")
	run(false, "export")
	expect("GET")
	if !strings.Contains(readFile(), "threat_model: remote_sources") {
		t.Fatal(readFile())
	}
	mu.Lock()
	raw["state"] = "not-configured"
	mu.Unlock()
	_, info = run(false, "export")
	expect("GET")
	output := readFile()
	if !strings.Contains(output, "default_setup: false") || strings.Contains(output, "threat_model:") || strings.Contains(output, "query_suite:") || strings.Contains(output, "runner:") || strings.Contains(output, "languages:") || !strings.Contains(info, "codeql_analysis") {
		t.Fatal(output, info)
	}
	write("remote_sources", ", default_setup: true")
	run(true, "apply")
	expect("GET")
}
