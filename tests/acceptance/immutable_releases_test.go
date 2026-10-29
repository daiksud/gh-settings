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

func TestImmutableReleasesExportPlanApplyAndSummary(t *testing.T) {
	binary := acceptanceBinary(t)
	var mu sync.Mutex
	enabled, enforced := false, false
	mode := ""
	policyReads, redirects := 0, 0
	writes := []string{}
	fixture, err := startHTTPFixture(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/redirected" {
			redirects++
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
		switch r.URL.Path {
		case "/repos/owner/current":
			if r.Method == "GET" {
				fmt.Fprint(w, planResponse)
				return
			}
			if r.Method == "PATCH" {
				var fields map[string]any
				if err := json.NewDecoder(r.Body).Decode(&fields); err != nil || len(fields) != 1 || fields["description"] != "earlier" {
					t.Error("unexpected earlier payload", fields, err)
				}
				writes = append(writes, "PATCH description")
				fmt.Fprint(w, `{}`)
				return
			}
		case "/repos/owner/current/topics":
			if r.Method == "GET" {
				fmt.Fprint(w, `{"names":[]}`)
				return
			}
		case "/repos/owner/current/labels":
			if r.Method == "GET" {
				fmt.Fprint(w, `[]`)
				return
			}
		case "/repos/owner/current/immutable-releases":
			if r.Method == "GET" {
				policyReads++
				switch mode {
				case "denied":
					w.WriteHeader(403)
					fmt.Fprint(w, `{"message":"unavailable"}`)
					return
				case "incomplete":
					fmt.Fprint(w, `{"enabled":true}`)
					return
				case "redirect":
					w.Header().Set("Location", "/redirected")
					w.WriteHeader(301)
					return
				}
				if !enabled {
					w.WriteHeader(404)
					fmt.Fprint(w, `{"message":"Not Found"}`)
					return
				}
				fmt.Fprintf(w, `{"enabled":true,"enforced_by_owner":%t}`, enforced)
				return
			}
			if r.Method != "PUT" && r.Method != "DELETE" {
				t.Error("unexpected policy method", r.Method)
				w.WriteHeader(400)
				return
			}
			if r.Body != nil {
				body, _ := io.ReadAll(r.Body)
				if len(body) > 0 {
					t.Error("toggle sent body", string(body))
				}
			}
			writes = append(writes, r.Method+" immutable-releases")
			if mode == "conflict" {
				w.WriteHeader(409)
				fmt.Fprint(w, `{"message":"policy conflict"}`)
				return
			}
			enabled = r.Method == "PUT"
			w.WriteHeader(204)
			return
		}
		t.Error("individual release/tag/asset or unrelated endpoint", r.Method, r.URL)
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
	write := func(value string) {
		t.Helper()
		if err := writeSettingsFile(fixture.work, generalInput("releases: {enable_release_immutability: "+value+"}\n")); err != nil {
			t.Fatal(err)
		}
	}
	path := filepath.Join(fixture.work, ".github/settings.yml")
	run("export")
	mu.Lock()
	ordinaryReads := policyReads
	mu.Unlock()
	if ordinaryReads != 0 {
		t.Fatal("ordinary export acquired immutable releases")
	}
	run("export", "--security", "immutable-releases")
	data, err := os.ReadFile(path)
	if err != nil || !strings.Contains(string(data), "enable_release_immutability: false") || strings.Contains(string(data), "enforced_by_owner") {
		t.Fatal(string(data), err)
	}
	if got := run("plan"); got != "No differences for managed repository settings.\n" {
		t.Fatal(got)
	}
	write("true")
	plan := run("plan")
	if !strings.Contains(plan, "settings.security.immutable_releases: false → true") {
		t.Fatal(plan)
	}
	report := filepath.Join(fixture.root, "immutable-releases-plan.json")
	run("action", "plan", report)
	data, _ = os.ReadFile(report)
	var typed map[string]any
	if err := json.Unmarshal(data, &typed); err != nil || typed["managed"] != float64(1) || len(typed["security"].([]any)) != 1 {
		t.Fatal(string(data), err)
	}
	summary := filepath.Join(fixture.root, "immutable-releases-summary.md")
	summaryEnv := append(fixtureEnv(fixture.cfg), "SETTINGS_ACTOR=user", "SETTINGS_TRIGGERING_ACTOR=user", "SETTINGS_EVENT=push", "SETTINGS_RUN_ID=123", "SETTINGS_RUN_ATTEMPT=1", "SETTINGS_WORKFLOW_REF=owner/current/.github/workflows/settings.yml@refs/heads/main", "SETTINGS_WORKFLOW_SHA="+strings.Repeat("a", 40), "SETTINGS_CONFIG_SHA="+strings.Repeat("b", 40), "SETTINGS_TARGET=owner/current", "SETTINGS_CLI_VERSION=dev", "SETTINGS_CLI_SHA256="+strings.Repeat("c", 64), "GH_TOKEN=", "GITHUB_TOKEN=")
	reads := fixture.reads.Load()
	r, err := executeCLI(binary, fixture.root, fixture.root, summaryEnv, []string{"action", "summary", report, summary}, false)
	if err != nil || r.runErr != nil || fixture.reads.Load() != reads {
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
	write("false")
	run("apply")
	run("apply")
	mu.Lock()
	gotWrites := append([]string{}, writes...)
	mu.Unlock()
	if !reflect.DeepEqual(gotWrites, []string{"PUT immutable-releases", "DELETE immutable-releases"}) {
		t.Fatal(gotWrites)
	}
	for _, failure := range []string{"denied", "incomplete", "redirect", "owner-enforced"} {
		mu.Lock()
		mode = failure
		enforced = failure == "owner-enforced"
		enabled = enforced
		before := len(writes)
		mu.Unlock()
		write("false")
		original, _ := os.ReadFile(path)
		for _, command := range []string{"plan", "apply"} {
			r, err := executeCLI(binary, fixture.work, fixture.root, env, []string{command}, false)
			if err != nil || r.runErr == nil || r.stdout.Len() != 0 {
				t.Fatal(failure, command, err, r.stderr.String())
			}
		}
		if failure != "owner-enforced" {
			r, err := executeCLI(binary, fixture.work, fixture.root, env, []string{"export", "--security", "immutable-releases"}, false)
			after, _ := os.ReadFile(path)
			if err != nil || r.runErr == nil || string(after) != string(original) {
				t.Fatal(failure, err, string(after), r.stderr.String())
			}
		}
		mu.Lock()
		unchanged := len(writes) == before && redirects == 0
		mu.Unlock()
		if !unchanged {
			t.Fatal("failure mutated or followed redirect", failure)
		}
	}
	mu.Lock()
	mode = "conflict"
	enforced = false
	enabled = false
	before := len(writes)
	mu.Unlock()
	if err := writeSettingsFile(fixture.work, repositoryInput("about: {description: earlier}\nsettings: {general: {releases: {enable_release_immutability: true}}}\n")); err != nil {
		t.Fatal(err)
	}
	r, err = executeCLI(binary, fixture.work, fixture.root, env, []string{"apply"}, false)
	if err != nil || r.runErr == nil || !strings.Contains(r.stderr.String(), "may already have applied") {
		t.Fatal(err, r.stderr.String())
	}
	mu.Lock()
	partial := append([]string{}, writes[before:]...)
	mu.Unlock()
	if !reflect.DeepEqual(partial, []string{"PATCH description", "PUT immutable-releases"}) {
		t.Fatal("partial writes/retry/rollback", partial)
	}
}
