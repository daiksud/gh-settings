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

func TestPagesExistingSiteExportPlanApplyAndPrivateSummary(t *testing.T) {
	binary := acceptanceBinary(t)
	var mu sync.Mutex
	state := map[string]any{"build_type": "legacy", "source": map[string]any{"branch": "main", "path": "/docs"}, "cname": "example.com", "https_enforced": true, "public": true, "status": "building"}
	mode := ""
	pagesReads, redirected := 0, 0
	writes := []map[string]any{}
	events := []string{}
	fixture, err := startHTTPFixture(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/repos/other/redirected" {
			redirected++
			w.WriteHeader(400)
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
			if r.Method == "PATCH" {
				events = append(events, "repository")
				fmt.Fprint(w, `{}`)
				return
			}
			if r.Method != "GET" {
				t.Error("unexpected repository method", r.Method)
				w.WriteHeader(400)
				return
			}
			var base map[string]any
			json.Unmarshal([]byte(planResponse), &base)
			base["id"] = 123
			base["full_name"] = "owner/current"
			base["archived"] = false
			json.NewEncoder(w).Encode(base)
		case "/repos/owner/current/topics":
			fmt.Fprint(w, `{"names":[]}`)
		case "/repos/owner/current/labels":
			fmt.Fprint(w, `[]`)
		case "/repos/owner/current/pages":
			if r.Method == "GET" {
				pagesReads++
				switch mode {
				case "404":
					w.WriteHeader(404)
					fmt.Fprint(w, `{"message":"CANARY-hidden"}`)
					return
				case "incomplete":
					fmt.Fprint(w, `{"cname":null}`)
					return
				case "get-redirect":
					w.Header().Set("Location", "/repos/other/redirected")
					w.WriteHeader(301)
					return
				}
				json.NewEncoder(w).Encode(state)
				return
			}
			if r.Method != "PUT" {
				t.Error("lifecycle/build write", r.Method)
				w.WriteHeader(400)
				return
			}
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Error(err)
			}
			for key := range body {
				if key != "cname" && key != "https_enforced" && key != "build_type" && key != "source" {
					t.Error("unowned Pages write", body)
				}
			}
			writes = append(writes, body)
			events = append(events, "pages")
			if mode == "write422" {
				w.WriteHeader(422)
				fmt.Fprint(w, `{"message":"certificate pending"}`)
				return
			}
			if mode == "put-redirect" {
				w.Header().Set("Location", "/repos/other/redirected")
				w.WriteHeader(307)
				return
			}
			if mode != "ignored" {
				for key, v := range body {
					state[key] = v
				}
			}
			if state["build_type"] == "workflow" {
				delete(state, "source")
			}
			if mode == "visibility" {
				state["public"] = false
			}
			w.WriteHeader(204)
		default:
			t.Error("unrelated request", r.Method, r.URL)
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
	write := func(pages string) {
		t.Helper()
		if err := writeSettingsFile(fixture.work, "schemaVersion: 1\nsettings:\n  pages: "+pages+"\n"); err != nil {
			t.Fatal(err)
		}
	}
	path := filepath.Join(fixture.work, ".github/settings.yml")
	run("export")
	mu.Lock()
	if pagesReads != 0 {
		t.Fatal("ordinary export requested Pages")
	}
	mu.Unlock()
	run("export", "--pages")
	run("validate")
	if got := run("plan"); !strings.Contains(got, "No differences") {
		t.Fatal(got)
	}
	exported, _ := os.ReadFile(path)
	if !strings.Contains(string(exported), "cname: example.com") || strings.Contains(string(exported), "public:") || strings.Contains(string(exported), "status:") {
		t.Fatal(string(exported))
	}
	write("{cname: null, https_enforced: false}")
	plan := run("plan")
	if !strings.Contains(plan, `"example.com" → null`) || !strings.Contains(plan, "true → false") {
		t.Fatal(plan)
	}
	report := filepath.Join(fixture.root, "pages-report.json")
	run("action", "plan", report, "status-only")
	data, _ := os.ReadFile(report)
	var typed map[string]any
	json.Unmarshal(data, &typed)
	changes := typed["pages"].([]any)
	if typed["managed"] != float64(2) || changes[0].(map[string]any)["after"] != nil {
		t.Fatal(string(data))
	}
	summary := filepath.Join(fixture.root, "pages-summary.md")
	summaryEnv := append(fixtureEnv(fixture.cfg), "SETTINGS_TARGET=owner/current", "GH_TOKEN=", "GITHUB_TOKEN=")
	reads := fixture.reads.Load()
	r, err := executeCLI(binary, fixture.root, fixture.root, summaryEnv, []string{"action", "summary", report, summary, "status-only"}, false)
	data, _ = os.ReadFile(summary)
	if err != nil || r.runErr != nil || fixture.reads.Load() != reads || strings.Contains(string(data), "example.com") || r.stdout.String() != "plan-outcome=changed\n" {
		t.Fatal(err, r.stderr.String(), string(data))
	}
	run("apply")
	run("apply")
	mu.Lock()
	if len(writes) != 1 || !reflect.DeepEqual(writes[0], map[string]any{"cname": nil, "https_enforced": false}) {
		t.Fatal(writes)
	}
	mu.Unlock()
	run("export", "--pages")
	data, _ = os.ReadFile(path)
	if !strings.Contains(string(data), "cname: null") || !strings.Contains(run("plan"), "No differences") {
		t.Fatal(string(data))
	}
	write("{build_type: workflow}")
	run("apply")
	write("{build_type: legacy, source: {branch: docs, path: /}}")
	if got := run("plan"); !strings.Contains(got, "inactive →") {
		t.Fatal(got)
	}
	run("apply")
	for _, failure := range []string{"404", "incomplete", "get-redirect"} {
		mu.Lock()
		mode = failure
		before := len(writes)
		mu.Unlock()
		original, _ := os.ReadFile(path)
		r, err = executeCLI(binary, fixture.work, fixture.root, env, []string{"export", "--pages"}, false)
		after, _ := os.ReadFile(path)
		if err != nil || r.runErr == nil || r.stdout.Len() != 0 || string(after) != string(original) {
			t.Fatal(failure, err, r.stderr.String())
		}
		r, err = executeCLI(binary, fixture.work, fixture.root, env, []string{"action", "plan", report, "status-only"}, false)
		if err != nil || r.runErr == nil || strings.Contains(r.stderr.String(), "CANARY") {
			t.Fatal("private error disclosure", err, r.stderr.String())
		}
		mu.Lock()
		if len(writes) != before || redirected != 0 {
			t.Fatal("read failure mutated/redirected")
		}
		mu.Unlock()
	}
	for _, failure := range []string{"write422", "put-redirect", "ignored", "visibility"} {
		mu.Lock()
		mode = failure
		state["public"] = true
		state["cname"] = nil
		before := len(writes)
		mu.Unlock()
		write("{cname: changed.example}")
		r, err = executeCLI(binary, fixture.work, fixture.root, env, []string{"apply"}, false)
		if err != nil || r.runErr == nil {
			t.Fatal(failure, err, r.stderr.String())
		}
		mu.Lock()
		if len(writes) != before+1 || redirected != 0 {
			t.Fatal("retry/fallback/redirect", writes)
		}
		mu.Unlock()
	}
	mu.Lock()
	mode = "write422"
	state["public"] = true
	state["cname"] = nil
	events = nil
	mu.Unlock()
	if err := writeSettingsFile(fixture.work, "schemaVersion: 1\nsettings:\n  repository: {description: changed}\n  pages: {cname: changed.example}\n"); err != nil {
		t.Fatal(err)
	}
	r, err = executeCLI(binary, fixture.work, fixture.root, env, []string{"apply"}, false)
	if err != nil || r.runErr == nil || !strings.Contains(r.stderr.String(), "earlier successful writes remain applied") {
		t.Fatal(err, r.stderr.String())
	}
	mu.Lock()
	defer mu.Unlock()
	if !reflect.DeepEqual(events, []string{"repository", "pages"}) {
		t.Fatal(events)
	}
}
