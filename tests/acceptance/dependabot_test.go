//go:build acceptance

package acceptance

import (
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
)

func TestDependabotExportPlanApplyAndAcceptedDisable(t *testing.T) {
	binary := acceptanceBinary(t)
	var mu sync.Mutex
	a, u, paused := false, false, false
	mode := ""
	var writes []string
	dependabotReads := 0
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
				t.Error("unrelated repository write")
				w.WriteHeader(400)
				return
			}
			fmt.Fprint(w, planResponse)
			return
		case "/repos/owner/current/topics":
			fmt.Fprint(w, `{"names":[]}`)
			return
		case "/repos/owner/current/labels":
			fmt.Fprint(w, `[]`)
			return
		case "/repos/owner/current/vulnerability-alerts", "/repos/owner/current/automated-security-fixes":
			isAlert := strings.HasSuffix(r.URL.Path, "/vulnerability-alerts")
			if r.Method == "GET" {
				dependabotReads++
				if !isAlert && mode == "denied" {
					w.WriteHeader(403)
					fmt.Fprint(w, `{"message":"denied"}`)
					return
				}
				if !isAlert && mode == "malformed" {
					fmt.Fprint(w, `{"enabled":true}`)
					return
				}
				if isAlert {
					if a {
						w.WriteHeader(204)
					} else {
						w.WriteHeader(404)
						fmt.Fprint(w, `{"message":"Not Found"}`)
					}
					return
				}
				if !u {
					w.WriteHeader(404)
					fmt.Fprint(w, `{"message":"Not Found"}`)
					return
				}
				json.NewEncoder(w).Encode(map[string]bool{"enabled": u, "paused": paused})
				return
			}
			if r.Method != "PUT" && r.Method != "DELETE" {
				t.Error("unexpected method", r.Method)
				w.WriteHeader(400)
				return
			}
			body, _ := io.ReadAll(r.Body)
			if len(body) > 0 {
				t.Error("nonempty toggle payload", string(body))
			}
			name := "updates"
			if isAlert {
				name = "alerts"
			}
			writes = append(writes, r.Method+" "+name)
			if mode == "fail-parent" && isAlert {
				w.WriteHeader(403)
				fmt.Fprint(w, `{"message":"parent denied"}`)
				return
			}
			if isAlert {
				a = r.Method == "PUT"
			} else {
				u = r.Method == "PUT"
			}
			w.WriteHeader(204)
			return
		}
		t.Error("unexpected endpoint", r.Method, r.URL)
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
	run := func(args ...string) (string, string) {
		t.Helper()
		r, err := executeCLI(binary, fixture.work, fixture.root, env, args, false)
		if err != nil || r.runErr != nil {
			t.Fatal(args, err, r.stderr.String())
		}
		return r.stdout.String(), r.stderr.String()
	}
	write := func(body string) {
		t.Helper()
		if err := writeSettingsFile(fixture.work, repositoryInput("settings: {advanced_security: {dependabot: "+body+"}}\n")); err != nil {
			t.Fatal(err)
		}
	}
	path := filepath.Join(fixture.work, ".github/settings.yml")
	run("export")
	mu.Lock()
	if dependabotReads != 0 {
		t.Fatal("ordinary export acquired Dependabot")
	}
	mu.Unlock()
	_, warning := run("export", "--security", "dependabot")
	data, err := os.ReadFile(path)
	if err != nil || !strings.Contains(string(data), "alerts: false") || !strings.Contains(string(data), "security_updates: false") || !strings.Contains(warning, "insufficient permission may be misclassified") {
		t.Fatal(string(data), warning, err)
	}
	run("validate")
	if out, _ := run("plan"); out != "No differences for managed repository settings.\n" {
		t.Fatal(out)
	}
	write("{alerts: true, security_updates: true}")
	plan, _ := run("plan")
	if !strings.Contains(plan, "dependabot.alerts: false → true") || !strings.Contains(plan, "dependabot.security_updates: false → true") {
		t.Fatal(plan)
	}
	report := filepath.Join(fixture.root, "dependabot-report.json")
	run("action", "plan", report)
	data, _ = os.ReadFile(report)
	var typed map[string]any
	if err := json.Unmarshal(data, &typed); err != nil || typed["managed"] != float64(2) || len(typed["security"].([]any)) != 2 {
		t.Fatal(string(data), err)
	}
	summary := filepath.Join(fixture.root, "dependabot-summary.md")
	summaryEnv := append(fixtureEnv(fixture.cfg), "SETTINGS_ACTOR=user", "SETTINGS_TRIGGERING_ACTOR=user", "SETTINGS_EVENT=push", "SETTINGS_RUN_ID=123", "SETTINGS_RUN_ATTEMPT=1", "SETTINGS_WORKFLOW_REF=owner/current/.github/workflows/settings.yml@refs/heads/main", "SETTINGS_WORKFLOW_SHA="+strings.Repeat("a", 40), "SETTINGS_CONFIG_SHA="+strings.Repeat("b", 40), "SETTINGS_TARGET=owner/current", "SETTINGS_CLI_VERSION=dev", "SETTINGS_CLI_SHA256="+strings.Repeat("c", 64), "GH_TOKEN=", "GITHUB_TOKEN=")
	reads := fixture.reads.Load()
	r, err := executeCLI(binary, fixture.root, fixture.root, summaryEnv, []string{"action", "summary", report, summary}, false)
	if err != nil || r.runErr != nil || fixture.reads.Load() != reads {
		t.Fatal(err, r.stderr.String())
	}
	data, _ = os.ReadFile(summary)
	for _, line := range strings.Split(strings.TrimSpace(plan), "\n") {
		if !strings.Contains(html.UnescapeString(string(data)), line) {
			t.Fatal(string(data), line)
		}
	}
	run("apply")
	run("apply")
	mu.Lock()
	if !a || !u || !reflect.DeepEqual(writes, []string{"PUT alerts", "PUT updates"}) {
		t.Fatal(a, u, writes)
	}
	paused = true
	mu.Unlock()
	if out, diag := run("plan"); out != "No differences for managed repository settings.\n" || !strings.Contains(diag, "are paused") {
		t.Fatal(out, diag)
	}
	run("apply")
	mu.Lock()
	if len(writes) != 2 {
		t.Fatal("paused equality resumed updates", writes)
	}
	mu.Unlock()
	write("{security_updates: false}")
	out, diag := run("apply")
	if !strings.Contains(out, "not independently verified") || !strings.Contains(diag, "disable accepted after DELETE 204 and GET 404") {
		t.Fatal(out, diag)
	}
	mu.Lock()
	if !a || u || len(writes) != 3 || writes[2] != "DELETE updates" {
		t.Fatal("omitted alerts changed", a, u, writes)
	}
	a, u = true, true
	mode = "fail-parent"
	mu.Unlock()
	write("{alerts: false, security_updates: false}")
	r, err = executeCLI(binary, fixture.work, fixture.root, env, []string{"apply"}, false)
	if err != nil || r.runErr == nil || !strings.Contains(r.stderr.String(), "not independently verified") || !strings.Contains(r.stderr.String(), "parent denied") {
		t.Fatal(err, r.stdout.String(), r.stderr.String())
	}
	mu.Lock()
	if !a || u || !reflect.DeepEqual(writes[len(writes)-2:], []string{"DELETE updates", "DELETE alerts"}) {
		t.Fatal("ordered partial failure", a, u, writes)
	}
	mode = ""
	mu.Unlock()
	run("apply")
	run("apply")
	for _, failure := range []string{"denied", "malformed"} {
		mu.Lock()
		mode = failure
		beforeWrites := len(writes)
		mu.Unlock()
		before, _ := os.ReadFile(path)
		r, err = executeCLI(binary, fixture.work, fixture.root, env, []string{"export", "--security", "dependabot"}, false)
		after, _ := os.ReadFile(path)
		if err != nil || r.runErr == nil || string(before) != string(after) {
			t.Fatal(failure, err, r.stderr.String(), string(after))
		}
		mu.Lock()
		if len(writes) != beforeWrites {
			t.Fatal("export mutated")
		}
		mu.Unlock()
	}
	write("{alerts: false, security_updates: true}")
	reads = fixture.reads.Load()
	r, err = executeCLI(binary, fixture.work, fixture.root, env, []string{"apply"}, false)
	if err != nil || r.runErr == nil || fixture.reads.Load() != reads {
		t.Fatal("contradiction did remote work", err, r.stderr.String())
	}
}
