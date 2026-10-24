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

func TestArchiveExportPlanApplyAndSummary(t *testing.T) {
	binary := acceptanceBinary(t)
	var mu sync.Mutex
	archived, description := false, "Initial"
	var writes []string
	var reads int
	missingArchive := false
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
		if r.URL.Path == "/repos/owner/current" {
			if r.Method == "GET" {
				reads++
				var base map[string]any
				_ = json.Unmarshal([]byte(planResponse), &base)
				base["id"] = float64(42)
				base["full_name"] = "owner/current"
				base["description"] = description
				if !missingArchive {
					base["archived"] = archived
				}
				_ = json.NewEncoder(w).Encode(base)
				return
			}
			if r.Method == "PATCH" {
				var payload map[string]any
				if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
					t.Error(err)
					w.WriteHeader(400)
					return
				}
				if len(payload) != 1 {
					t.Error("overbroad lifecycle PATCH", payload)
					w.WriteHeader(400)
					return
				}
				if value, ok := payload["archived"]; ok {
					state, isBool := value.(bool)
					if !isBool {
						t.Error(payload)
						w.WriteHeader(400)
						return
					}
					writes = append(writes, fmt.Sprintf("archive:%t", state))
					archived = state
				} else if value, ok := payload["description"]; ok {
					str, isString := value.(string)
					if !isString || archived {
						t.Error("ordinary write to archived repo", payload)
						w.WriteHeader(422)
						return
					}
					description = str
					writes = append(writes, "description")
				} else {
					t.Error("unexpected PATCH", payload)
					w.WriteHeader(400)
					return
				}
				_ = json.NewEncoder(w).Encode(map[string]any{"id": 42, "full_name": "owner/current", "archived": archived})
				return
			}
		}
		if r.Method == "GET" && r.URL.Path == "/repos/owner/current/topics" {
			fmt.Fprint(w, `{"names":[]}`)
			return
		}
		if r.Method == "GET" && r.URL.Path == "/repos/owner/current/labels" {
			fmt.Fprint(w, `[]`)
			return
		}
		t.Error("unexpected request", r.Method, r.URL)
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
			t.Fatalf("%v: %v %v %s", args, err, r.runErr, r.stderr.String())
		}
		return r.stdout.String()
	}
	path := filepath.Join(fixture.work, ".github/settings.yml")
	run("export")
	raw, err := os.ReadFile(path)
	if err != nil || strings.Contains(string(raw), "archived:") {
		t.Fatal("ordinary export added lifecycle", string(raw), err)
	}
	run("export", "--archive")
	raw, err = os.ReadFile(path)
	if err != nil || !strings.Contains(string(raw), "archived: false") {
		t.Fatal("explicit archive export missing", string(raw), err)
	}
	run("validate")
	if got := run("plan"); got != "No differences for managed repository settings.\n" {
		t.Fatal(got)
	}
	input := repositoryInput("archived: true\ndescription: Closed for maintenance\n")
	if err := writeSettingsFile(fixture.work, input); err != nil {
		t.Fatal(err)
	}
	got := run("plan")
	if !strings.Contains(got, "settings.repository.archived: false → true") || !strings.Contains(got, "settings.repository.description:") {
		t.Fatal(got)
	}
	report := filepath.Join(fixture.root, "archive-plan.json")
	run("action", "plan", report)
	data, err := os.ReadFile(report)
	if err != nil {
		t.Fatal(err)
	}
	var outcome map[string]any
	if err := json.Unmarshal(data, &outcome); err != nil || outcome["managed"] != float64(2) {
		t.Fatal(string(data), err)
	}
	summary := filepath.Join(fixture.root, "archive-summary.md")
	summaryEnv := append(fixtureEnv(fixture.cfg), "SETTINGS_ACTOR=owner", "SETTINGS_TRIGGERING_ACTOR=owner", "SETTINGS_EVENT=push", "SETTINGS_RUN_ID=123", "SETTINGS_RUN_ATTEMPT=1", "SETTINGS_WORKFLOW_REF=owner/current/.github/workflows/main.yml@refs/heads/main", "SETTINGS_WORKFLOW_SHA="+strings.Repeat("a", 40), "SETTINGS_CONFIG_SHA="+strings.Repeat("b", 40), "SETTINGS_TARGET=owner/current", "SETTINGS_CLI_VERSION=dev", "SETTINGS_CLI_SHA256="+strings.Repeat("c", 64))
	result, err := executeCLI(binary, fixture.root, fixture.root, summaryEnv, []string{"action", "summary", report, summary}, false)
	if err != nil || result.runErr != nil {
		t.Fatal(err, result.stderr.String())
	}
	data, err = os.ReadFile(summary)
	if err != nil || !strings.Contains(html.UnescapeString(string(data)), "settings.repository.archived: false → true") {
		t.Fatal(string(data), err)
	}
	run("apply")
	mu.Lock()
	gotWrites := append([]string(nil), writes...)
	mu.Unlock()
	if !reflect.DeepEqual(gotWrites, []string{"description", "archive:true"}) {
		t.Fatal(gotWrites)
	}
	input = repositoryInput("archived: true\ndescription: Closed for maintenance\n")
	if err := writeSettingsFile(fixture.work, input); err != nil {
		t.Fatal(err)
	}
	if got := run("plan"); got != "No differences for managed repository settings.\n" {
		t.Fatal(got)
	}
	input = repositoryInput("archived: true\ndescription: Different\n")
	if err := writeSettingsFile(fixture.work, input); err != nil {
		t.Fatal(err)
	}
	if got := run("plan"); !strings.Contains(got, "settings.repository.description: \"Closed for maintenance\" → \"Different\"") {
		t.Fatal(got)
	}
	failedApply, err := executeCLI(binary, fixture.work, fixture.root, env, []string{"apply"}, false)
	if err != nil || failedApply.runErr == nil || !strings.Contains(failedApply.stderr.String(), "without explicit archived: false") {
		t.Fatal("archived mixed apply did not fail closed", err, failedApply.runErr, failedApply.stderr.String())
	}
	mu.Lock()
	if len(writes) != 2 {
		t.Fatal("failed apply wrote settings", writes)
	}
	mu.Unlock()
	input = repositoryInput("archived: true\n")
	if err := writeSettingsFile(fixture.work, input); err != nil {
		t.Fatal(err)
	}
	run("apply") // Matching archive requires only a read.
	mu.Lock()
	count := len(writes)
	mu.Unlock()
	if count != 2 {
		t.Fatal(writes)
	}
	input = repositoryInput("archived: false\ndescription: Active again\n")
	if err := writeSettingsFile(fixture.work, input); err != nil {
		t.Fatal(err)
	}
	run("apply")
	mu.Lock()
	gotWrites = append([]string(nil), writes...)
	active := !archived
	mu.Unlock()
	if !active || !reflect.DeepEqual(gotWrites, []string{"description", "archive:true", "archive:false", "description"}) {
		t.Fatal(gotWrites, active)
	}
	// An unavailable archive observation cannot be silently represented as false on opt-in export.
	mu.Lock()
	missingArchive = true
	mu.Unlock()
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	failed, err := executeCLI(binary, fixture.work, fixture.root, env, []string{"export", "--archive"}, false)
	if err != nil || failed.runErr == nil {
		t.Fatal("missing archive read succeeded", err)
	}
	after, err := os.ReadFile(path)
	if err != nil || string(before) != string(after) {
		t.Fatal("failed export modified target", err)
	}
	mu.Lock()
	if reads < 5 {
		t.Fatal("archive reads were not performed", reads)
	}
	mu.Unlock()
}
