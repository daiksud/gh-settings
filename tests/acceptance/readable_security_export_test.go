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

func TestReadableSecurityExportFirstFileAndRetainedIntent(t *testing.T) {
	const sensitive = "credential-and-secret-value-must-not-be-logged"
	var mu sync.Mutex
	mode := "first"
	lastReads := 0
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
		if r.Method != "GET" {
			t.Error("export mutated remote state", r.Method, r.URL)
			w.WriteHeader(500)
			return
		}
		switch r.URL.Path {
		case "/repos/owner/current":
			var body map[string]any
			json.Unmarshal([]byte(planResponse), &body)
			body["visibility"] = "public"
			body["owner"] = map[string]any{"type": "User"}
			if mode == "existing" {
				body["security_and_analysis"] = map[string]any{"secret_scanning": map[string]string{"status": "disabled"}, "secret_scanning_push_protection": map[string]string{"status": "disabled"}}
			}
			json.NewEncoder(w).Encode(body)
		case "/repos/owner/current/topics":
			fmt.Fprint(w, `{"names":[]}`)
		case "/repos/owner/current/labels":
			fmt.Fprint(w, `[]`)
		case "/repos/owner/current/private-vulnerability-reporting":
			fmt.Fprint(w, `{"enabled":false}`)
		case "/repos/owner/current/immutable-releases":
			lastReads++
			fmt.Fprint(w, `{"enabled":true,"enforced_by_owner":false}`)
		case "/repos/owner/current/vulnerability-alerts":
			if mode == "existing" {
				w.WriteHeader(204)
			} else {
				w.WriteHeader(403)
				fmt.Fprintf(w, `{"message":%q}`, sensitive)
			}
		case "/repos/owner/current/automated-security-fixes":
			fmt.Fprint(w, `{"enabled":true}`) // incomplete family, not false
		case "/repos/owner/current/code-scanning/default-setup", "/repos/owner/current/code-scanning/ai-scan":
			w.WriteHeader(403)
			fmt.Fprintf(w, `{"message":%q}`, sensitive)
		default:
			t.Error("unexpected endpoint", r.URL)
			w.WriteHeader(500)
		}
	}, false, "/private-vulnerability-reporting", "/immutable-releases", "/vulnerability-alerts", "/automated-security-fixes", "/code-scanning/default-setup", "/code-scanning/ai-scan")
	if err != nil {
		t.Fatal(err)
	}
	defer fixture.close()
	env := exportFixtureEnv(fixture.cfg)
	if err := setupOrdinaryOrigin(fixture.work, env); err != nil {
		t.Fatal(err)
	}
	binary := acceptanceBinary(t)
	path := filepath.Join(fixture.work, ".github/settings.yml")
	run := func() string {
		t.Helper()
		r, err := executeCLI(binary, fixture.work, fixture.root, env, []string{"export"}, false)
		if err != nil || r.runErr != nil || r.stdout.Len() != 0 || !strings.Contains(r.stderr.String(), "info: export ") || strings.Contains(r.stderr.String(), sensitive) {
			t.Fatal(err, r.runErr, r.stdout.String(), r.stderr.String())
		}
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		return string(data)
	}
	first := run()
	if strings.Contains(first, "secret_protection:") || strings.Contains(first, "code_security:") || strings.Contains(first, "dependabot:") || !strings.Contains(first, "private_vulnerability_reporting: false") || !strings.Contains(first, "immutable_release: true") {
		t.Fatal(first)
	}
	prior := `version: daiksud/gh-settings@v0
repository:
  settings:
    general:
      releases: {immutable_release: false}
    advanced_security:
      secret_protection: {enabled: true, push_protection: true}
      dependabot: {alerts: true, security_updates: false}
      private_vulnerability_reporting: true
      code_security:
        ai_scan: {ai_scan_for_pull_requests: true}
        codeql_analysis:
          default_setup: true
          languages: [go]
          query_suite: default
          runner: {type: standard_github_runner}
`
	if err := writeSettingsFile(fixture.work, prior); err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	mode = "existing"
	mu.Unlock()
	second := run()
	for _, want := range []string{"enabled: false", "push_protection: false", "alerts: true", "security_updates: false", "ai_scan_for_pull_requests: true", "default_setup: true", "query_suite: default", "private_vulnerability_reporting: false", "immutable_release: true"} {
		if !strings.Contains(second, want) {
			t.Fatal(want, second)
		}
	}
	mu.Lock()
	if lastReads != 2 {
		t.Fatal("later independent read skipped", lastReads)
	}
	mu.Unlock()
	if err := writeSettingsFile(fixture.work, "invalid: "+sensitive+"\n"); err != nil {
		t.Fatal(err)
	}
	before := fixture.reads.Load()
	r, err := executeCLI(binary, fixture.work, fixture.root, env, []string{"export"}, false)
	if err != nil || r.runErr == nil || fixture.reads.Load() != before || strings.Contains(r.stderr.String(), sensitive) {
		t.Fatal(err, r.runErr, r.stderr.String())
	}
	after, _ := os.ReadFile(path)
	if string(after) != "invalid: "+sensitive+"\n" {
		t.Fatal("invalid existing file overwritten")
	}
}
