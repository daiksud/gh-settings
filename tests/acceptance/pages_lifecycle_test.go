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

func TestPagesLifecycleExportPlanApplyAndPartialFailure(t *testing.T) {
	binary := acceptanceBinary(t)
	var mu sync.Mutex
	var state map[string]any
	var events []string
	denyOptional := false
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
			if r.Method == "PATCH" {
				events = append(events, "PATCH repository")
				fmt.Fprint(w, `{}`)
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
				if state == nil {
					w.WriteHeader(404)
					fmt.Fprint(w, `{"message":"Not Found"}`)
				} else {
					json.NewEncoder(w).Encode(state)
				}
				return
			}
			events = append(events, r.Method+" pages")
			switch r.Method {
			case "POST":
				var body map[string]any
				json.NewDecoder(r.Body).Decode(&body)
				expected := map[string]any{"build_type": body["build_type"]}
				if body["build_type"] == "legacy" {
					expected["source"] = map[string]any{"branch": "Main", "path": "/docs"}
				}
				if !reflect.DeepEqual(body, expected) || state != nil {
					t.Error("bad creation", body, state)
					w.WriteHeader(422)
					return
				}
				state = map[string]any{"build_type": body["build_type"], "cname": nil, "https_enforced": false, "public": true}
				if body["source"] != nil {
					state["source"] = body["source"]
				}
				w.WriteHeader(201)
			case "PUT":
				var body map[string]any
				json.NewDecoder(r.Body).Decode(&body)
				if denyOptional {
					w.WriteHeader(422)
					fmt.Fprint(w, `{"message":"certificate pending"}`)
					return
				}
				if state == nil {
					t.Error("updated absent site")
					w.WriteHeader(404)
					return
				}
				for k, v := range body {
					if k == "public" {
						t.Error("visibility mutation")
					}
					state[k] = v
				}
				w.WriteHeader(204)
			case "DELETE":
				state = nil
				w.WriteHeader(204)
			default:
				t.Error("unexpected method", r.Method)
				w.WriteHeader(400)
			}
		default:
			t.Error("unexpected endpoint", r.Method, r.URL)
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
		if err := writeSettingsFile(fixture.work, repositoryInput("settings: {pages: "+pages+"}\n")); err != nil {
			t.Fatal(err)
		}
	}
	config := filepath.Join(fixture.work, ".github/settings.yml")
	run("export", "--pages")
	data, _ := os.ReadFile(config)
	if !strings.Contains(string(data), "branch: none") || strings.Contains(string(data), "custom_domain:") || strings.Contains(string(data), "select_folder:") {
		t.Fatal(string(data))
	}
	if !strings.Contains(run("plan"), "No differences") {
		t.Fatal("absent export drift")
	}
	write("{build_and_deployment: {source: deploy_from_a_branch, branch: none}}")
	run("apply")
	mu.Lock()
	if len(events) != 0 {
		t.Fatal(events)
	}
	mu.Unlock()
	for _, bad := range []string{"{custom_domain: null}", "{build_and_deployment: {source: deploy_from_a_branch}}", "{build_and_deployment: {branch: Main, select_folder: docs}}", "{build_and_deployment: {source: deploy_from_a_branch, branch: none, enforce_https: false}}"} {
		write(bad)
		r, err := executeCLI(binary, fixture.work, fixture.root, env, []string{"apply"}, false)
		if err != nil || r.runErr == nil {
			t.Fatal(bad, err, r.stderr.String())
		}
		mu.Lock()
		if len(events) != 0 {
			t.Fatal("incomplete/conflicting intent wrote", events)
		}
		mu.Unlock()
	}
	report := filepath.Join(fixture.root, "pages-lifecycle.json")
	summary := filepath.Join(fixture.root, "pages-summary.md")
	for _, mode := range []string{"deploy_from_a_branch", "github_actions"} {
		controls := "{build_and_deployment: {source: " + mode
		if mode == "deploy_from_a_branch" {
			controls += ", branch: Main, select_folder: docs"
		}
		controls += "}, custom_domain: site.example, enforce_https: true}"
		write(controls)
		if !strings.Contains(run("plan"), "create site") {
			t.Fatal("creation not reported")
		}
		run("action", "plan", report)
		data, _ = os.ReadFile(report)
		if !strings.Contains(string(data), `"before_absent":true`) {
			t.Fatal(string(data))
		}
		summaryEnv := append(fixtureEnv(fixture.cfg), "SETTINGS_TARGET=owner/current", "GH_TOKEN=", "GITHUB_TOKEN=", "SETTINGS_ACTOR=user", "SETTINGS_TRIGGERING_ACTOR=user", "SETTINGS_EVENT=push", "SETTINGS_RUN_ID=123", "SETTINGS_RUN_ATTEMPT=1", "SETTINGS_WORKFLOW_REF=owner/current/.github/workflows/settings.yml@refs/heads/main", "SETTINGS_WORKFLOW_SHA="+strings.Repeat("a", 40), "SETTINGS_CONFIG_SHA="+strings.Repeat("b", 40), "SETTINGS_CLI_VERSION=dev", "SETTINGS_CLI_SHA256="+strings.Repeat("c", 64))
		for _, privacy := range []string{"", "status-only"} {
			args := []string{"action", "summary", report, summary}
			if privacy != "" {
				args = append(args, privacy)
			}
			r, err := executeCLI(binary, fixture.root, fixture.root, summaryEnv, args, false)
			if err != nil || r.runErr != nil {
				t.Fatal(err, r.stderr.String())
			}
			data, _ = os.ReadFile(summary)
			if privacy == "" && !strings.Contains(string(data), "create site") {
				t.Fatal(string(data))
			}
			if privacy == "status-only" && strings.Contains(string(data), "site.example") {
				t.Fatal(string(data))
			}
			os.Remove(summary)
		}
		run("apply")
		mu.Lock()
		before := len(events)
		mu.Unlock()
		run("apply")
		mu.Lock()
		if len(events) != before {
			t.Fatal("repeat mutated", events)
		}
		mu.Unlock()
		run("export", "--pages")
		if !strings.Contains(run("plan"), "No differences") {
			t.Fatal("present export drift")
		}
		write("{build_and_deployment: {source: deploy_from_a_branch, branch: none}}")
		if !strings.Contains(run("plan"), "delete site") {
			t.Fatal("deletion not reported")
		}
		run("apply")
		run("apply")
	}
	mu.Lock()
	events = nil
	denyOptional = true
	mu.Unlock()
	write("{build_and_deployment: {source: github_actions}, enforce_https: true}")
	r, err := executeCLI(binary, fixture.work, fixture.root, env, []string{"apply"}, false)
	if err != nil || r.runErr == nil || !strings.Contains(r.stderr.String(), "Pages site created") || !strings.Contains(r.stderr.String(), "creation remains applied") {
		t.Fatal(err, r.stderr.String())
	}
	mu.Lock()
	defer mu.Unlock()
	if state == nil || !reflect.DeepEqual(events, []string{"POST pages", "PUT pages"}) {
		t.Fatal("lost creation or retried", state, events)
	}
}
