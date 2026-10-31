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
)

func TestSecretScanningExportPlanApplyAndSummary(t *testing.T) {
	binary := acceptanceBinary(t)
	var mu sync.Mutex
	s, p := false, false
	mode := ""
	writes := []string{}
	reads, redirected := 0, 0
	fixture, err := startHTTPFixture(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/repos/other/redirected" {
			redirected++
			fmt.Fprint(w, `{}`)
			return
		}
		if servePRAccessFixtureRead(w, r) || serveIssueCreationPolicyFixtureRead(w, r) || serveSponsorshipsFixtureRead(w, r) {
			return
		}
		if isMergeMethodsFixtureRead(r) {
			fmt.Fprint(w, mergeMethodsFixtureResponse)
			return
		}
		if r.URL.Path == "/repos/owner/current" {
			if r.Method == "GET" {
				reads++
				if mode == "redirect" {
					w.Header().Set("Location", "/repos/other/redirected")
					w.WriteHeader(301)
					return
				}
				if mode == "denied" {
					w.WriteHeader(403)
					fmt.Fprint(w, `{"message":"feature unavailable"}`)
					return
				}
				var base map[string]any
				json.Unmarshal([]byte(planResponse), &base)
				if mode != "hidden" {
					a, b := "disabled", "disabled"
					if s {
						a = "enabled"
					}
					if p {
						b = "enabled"
					}
					base["security_and_analysis"] = map[string]any{"secret_scanning": map[string]string{"status": a}, "secret_scanning_push_protection": map[string]string{"status": b}, "secret_scanning_non_provider_patterns": map[string]string{"status": "disabled"}, "future_security": map[string]string{"status": "enabled"}}
				}
				json.NewEncoder(w).Encode(base)
				return
			}
			if r.Method != "PATCH" {
				t.Error("unexpected method", r.Method)
				w.WriteHeader(400)
				return
			}
			var body map[string]map[string]map[string]string
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Error(err)
				w.WriteHeader(400)
				return
			}
			block := body["security_and_analysis"]
			if len(body) != 1 || len(block) != 1 {
				t.Error("not a declared-only status update", body)
				w.WriteHeader(400)
				return
			}
			for key, value := range block {
				if len(value) != 1 || value["status"] != "enabled" && value["status"] != "disabled" {
					t.Error("invalid status payload", value)
				}
				writes = append(writes, key)
				if mode == "write-redirect" {
					w.Header().Set("Location", "/repos/other/redirected")
					w.WriteHeader(307)
					return
				}
				if mode == "deny-parent" && key == "secret_scanning" {
					w.WriteHeader(422)
					fmt.Fprint(w, `{"message":"organization refused"}`)
					return
				}
				if mode != "enforced" {
					switch key {
					case "secret_scanning":
						s = value["status"] == "enabled"
					case "secret_scanning_push_protection":
						p = value["status"] == "enabled"
					default:
						t.Error("unmanaged key sent", key)
					}
				}
			}
			fmt.Fprint(w, `{}`)
			return
		}
		if r.Method != "GET" {
			t.Error("unrelated mutation", r.Method, r.URL)
			w.WriteHeader(400)
			return
		}
		switch r.URL.Path {
		case "/repos/owner/current/topics":
			fmt.Fprint(w, `{"names":[]}`)
		case "/repos/owner/current/labels":
			fmt.Fprint(w, `[]`)
		default:
			t.Error("unexpected endpoint", r.URL)
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
		r, err := executeCLI(binary, fixture.work, fixture.root, env, args, false)
		if err != nil || r.runErr != nil || r.stderr.Len() != 0 {
			t.Fatal(args, err, r.stderr.String())
		}
		return r.stdout.String()
	}
	write := func(value string) {
		t.Helper()
		if err := writeSettingsFile(fixture.work, repositoryInput("settings: {advanced_security: {secret_protection: "+value+"}}\n")); err != nil {
			t.Fatal(err)
		}
	}
	path := filepath.Join(fixture.work, ".github/settings.yml")
	run("export")
	data, err := os.ReadFile(path)
	if err != nil || strings.Contains(string(data), "advanced_security:") {
		t.Fatal("ordinary export acquired security", string(data), err)
	}
	mu.Lock()
	ordinaryReads := reads
	mu.Unlock()
	if ordinaryReads != 1 {
		t.Fatal("ordinary export added repository reads", ordinaryReads)
	}
	run("export", "--security", "secret-scanning")
	run("validate")
	data, err = os.ReadFile(path)
	if err != nil || !strings.Contains(string(data), "secret_protection:\n") || !strings.Contains(string(data), "enabled: false") || !strings.Contains(string(data), "push_protection: false") || strings.Contains(string(data), "future_security") || strings.Contains(string(data), "security_and_quality:") {
		t.Fatal(string(data), err)
	}
	if got := run("plan"); got != "No differences for managed repository settings.\n" {
		t.Fatal(got)
	}
	write("{enabled: true, push_protection: true}")
	plan := run("plan")
	if !strings.Contains(plan, "repository.settings.advanced_security.secret_protection.enabled: false → true") || !strings.Contains(plan, "repository.settings.advanced_security.secret_protection.push_protection: false → true") {
		t.Fatal(plan)
	}
	report := filepath.Join(fixture.root, "security-plan.json")
	run("action", "plan", report)
	data, _ = os.ReadFile(report)
	var typed map[string]any
	if err := json.Unmarshal(data, &typed); err != nil {
		t.Fatal(err)
	}
	if typed["managed"] != float64(2) || len(typed["security"].([]any)) != 2 || len(typed["changes"].([]any)) != 0 {
		t.Fatal(string(data))
	}
	summary := filepath.Join(fixture.root, "security-summary.md")
	summaryEnv := append(fixtureEnv(fixture.cfg), "SETTINGS_ACTOR=user", "SETTINGS_TRIGGERING_ACTOR=user", "SETTINGS_EVENT=push", "SETTINGS_RUN_ID=123", "SETTINGS_RUN_ATTEMPT=1", "SETTINGS_WORKFLOW_REF=owner/current/.github/workflows/settings.yml@refs/heads/main", "SETTINGS_WORKFLOW_SHA="+strings.Repeat("a", 40), "SETTINGS_CONFIG_SHA="+strings.Repeat("b", 40), "SETTINGS_TARGET=owner/current", "SETTINGS_CLI_VERSION=dev", "SETTINGS_CLI_SHA256="+strings.Repeat("c", 64), "GH_TOKEN=", "GITHUB_TOKEN=")
	beforeReads := fixture.reads.Load()
	r, err := executeCLI(binary, fixture.root, fixture.root, summaryEnv, []string{"action", "summary", report, summary}, false)
	if err != nil || r.runErr != nil || fixture.reads.Load() != beforeReads {
		t.Fatal(err, r.stderr.String())
	}
	data, _ = os.ReadFile(summary)
	if !strings.Contains(html.UnescapeString(string(data)), strings.TrimSpace(plan)) {
		t.Fatal(string(data))
	}
	run("apply")
	run("apply")
	if got := run("plan"); got != "No differences for managed repository settings.\n" {
		t.Fatal(got)
	}
	mu.Lock()
	if !s || !p || !reflect.DeepEqual(writes, []string{"secret_scanning", "secret_scanning_push_protection"}) {
		t.Fatal(s, p, writes)
	}
	mu.Unlock()
	write("{push_protection: false}")
	run("apply")
	write("{enabled: false}")
	run("apply")
	mu.Lock()
	if s || p || !reflect.DeepEqual(writes[2:], []string{"secret_scanning_push_protection", "secret_scanning"}) {
		t.Fatal(s, p, writes)
	}
	mu.Unlock()
	// Unknown requested state and invalid selections preserve the existing file.
	for _, failure := range []string{"hidden", "denied", "redirect", "invalid-selection"} {
		mu.Lock()
		mode = failure
		before := len(writes)
		mu.Unlock()
		original, _ := os.ReadFile(path)
		args := []string{"export", "--security", "secret-scanning"}
		if failure == "redirect" {
			args = []string{"plan"}
		}
		if failure == "invalid-selection" {
			args = []string{"export", "--security", "all"}
		}
		r, err := executeCLI(binary, fixture.work, fixture.root, env, args, false)
		after, _ := os.ReadFile(path)
		if err != nil || r.runErr == nil || r.stdout.Len() != 0 || string(after) != string(original) {
			t.Fatal(failure, err, r.stderr.String(), string(after))
		}
		mu.Lock()
		if len(writes) != before || redirected != 0 {
			t.Fatal("failure mutated/followed redirect", failure, writes, redirected)
		}
		mu.Unlock()
	}
	mu.Lock()
	mode = "enforced"
	before := len(writes)
	mu.Unlock()
	write("{enabled: true}")
	r, err = executeCLI(binary, fixture.work, fixture.root, env, []string{"apply"}, false)
	if err != nil || r.runErr == nil || !strings.Contains(r.stderr.String(), "did not retain requested") {
		t.Fatal("enforced no-change accepted", err, r.stderr.String())
	}
	mu.Lock()
	if s || len(writes) != before+1 {
		t.Fatal(s, writes)
	}
	mode = ""
	mu.Unlock()
	// The production PATCH client rejects redirects without forwarding the write.
	mu.Lock()
	mode = "write-redirect"
	before = len(writes)
	mu.Unlock()
	r, err = executeCLI(binary, fixture.work, fixture.root, env, []string{"apply"}, false)
	if err != nil || r.runErr == nil {
		t.Fatal("PATCH redirect accepted", err, r.stderr.String())
	}
	mu.Lock()
	if s || p || redirected != 0 || len(writes) != before+1 {
		t.Fatal("PATCH redirect followed or retried", s, p, redirected, writes)
	}
	mode = ""
	mu.Unlock()
	// Child success survives a later parent refusal; no rollback or retry.
	write("{enabled: true, push_protection: true}")
	run("apply")
	mu.Lock()
	mode = "deny-parent"
	before = len(writes)
	mu.Unlock()
	write("{enabled: false, push_protection: false}")
	r, err = executeCLI(binary, fixture.work, fixture.root, env, []string{"apply"}, false)
	if err != nil || r.runErr == nil || !strings.Contains(r.stderr.String(), "earlier successful writes remain applied") {
		t.Fatal(err, r.stderr.String())
	}
	mu.Lock()
	if !s || p || !reflect.DeepEqual(writes[before:], []string{"secret_scanning_push_protection", "secret_scanning"}) {
		t.Fatal("partial progress lost or retried", s, p, writes)
	}
	mu.Unlock()
}
