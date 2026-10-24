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

func TestForkingExportPlanApplyFailureAndPrivateReport(t *testing.T) {
	binary := acceptanceBinary(t)
	var mu sync.Mutex
	allowed := true
	archived := false
	mode := ""
	orgReads, writes, redirects := 0, 0, 0
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
		case "/orgs/owner":
			orgReads++
			if r.Method != "GET" {
				t.Error("organization mutation")
				w.WriteHeader(400)
				return
			}
			if mode == "denied" {
				w.WriteHeader(404)
				fmt.Fprint(w, `{"message":"CANARY-hidden-policy"}`)
				return
			}
			if mode == "hidden" {
				fmt.Fprint(w, `{"id":7,"login":"owner"}`)
				return
			}
			fmt.Fprint(w, `{"id":7,"login":"owner","members_can_fork_private_repositories":true}`)
		case "/repos/owner/current":
			if mode == "redirect" && r.Header.Get("X-GitHub-Api-Version") == "2026-03-10" {
				w.Header().Set("Location", "/repos/other/redirected")
				w.WriteHeader(301)
				return
			}
			var base map[string]any
			json.Unmarshal([]byte(planResponse), &base)
			if r.Method == "PATCH" {
				var payload map[string]any
				json.NewDecoder(r.Body).Decode(&payload)
				if len(payload) != 1 {
					t.Error("overbroad", payload)
					w.WriteHeader(400)
					return
				}
				if v, ok := payload["allow_forking"].(bool); ok {
					allowed = v
				} else if v, ok := payload["archived"].(bool); ok {
					archived = v
				} else if _, ok := payload["description"]; !ok {
					t.Error(payload)
					w.WriteHeader(400)
					return
				}
				writes++
			} else if r.Method != "GET" {
				t.Error(r.Method)
				w.WriteHeader(400)
				return
			}
			base["id"] = 42
			base["full_name"] = "owner/current"
			base["private"] = true
			base["visibility"] = "private"
			base["fork"] = false
			base["archived"] = archived
			base["allow_forking"] = allowed
			base["owner"] = map[string]any{"id": 7, "login": "owner", "type": "Organization"}
			if mode == "public" {
				base["private"] = false
				base["visibility"] = "public"
			}
			json.NewEncoder(w).Encode(base)
		case "/repos/owner/current/topics":
			fmt.Fprint(w, `{"names":[]}`)
		case "/repos/owner/current/labels":
			fmt.Fprint(w, `[]`)
		case "/repos/other/redirected":
			redirects++
			w.WriteHeader(400)
		default:
			t.Error("unexpected request", r.Method, r.URL)
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
	run := func(ok bool, args ...string) string {
		t.Helper()
		r, err := executeCLI(binary, fixture.work, fixture.root, env, args, false)
		if err != nil || (r.runErr == nil) != ok {
			t.Fatalf("%v: %v %v %s", args, err, r.runErr, r.stderr.String())
		}
		return r.stdout.String() + r.stderr.String()
	}
	run(true, "export")
	path := filepath.Join(fixture.work, ".github/settings.yml")
	raw, _ := os.ReadFile(path)
	mu.Lock()
	reads := orgReads
	mu.Unlock()
	if reads != 0 || strings.Contains(string(raw), "allow_forking") {
		t.Fatal("ordinary export gained policy access", reads, string(raw))
	}
	run(true, "export", "--forking")
	raw, _ = os.ReadFile(path)
	if !strings.Contains(string(raw), "allow_forking: true") {
		t.Fatal(string(raw))
	}
	run(true, "validate")
	if err := writeSettingsFile(fixture.work, repositoryInput("allow_forking: false\n")); err != nil {
		t.Fatal(err)
	}
	plan := run(true, "plan")
	if !strings.Contains(plan, "settings.repository.allow_forking: true → false") {
		t.Fatal(plan)
	}
	report := filepath.Join(fixture.root, "forking-plan.json")
	run(true, "action", "plan", report)
	data, _ := os.ReadFile(report)
	var native struct {
		Changes []struct {
			Field         string
			Before, After any
		}
	}
	if err := json.Unmarshal(data, &native); err != nil || len(native.Changes) != 1 || native.Changes[0].Field != "repository.allow_forking" || native.Changes[0].Before != true || native.Changes[0].After != false {
		t.Fatal(string(data), err)
	}
	mu.Lock()
	count := writes
	mu.Unlock()
	if count != 0 {
		t.Fatal("plan wrote", count)
	}
	run(true, "apply")
	run(true, "apply")
	mu.Lock()
	count = writes
	value := allowed
	mu.Unlock()
	if count != 1 || value {
		t.Fatal("narrow/idempotent false", count, value)
	}
	if err := writeSettingsFile(fixture.work, repositoryInput("allow_forking: true\n")); err != nil {
		t.Fatal(err)
	}
	run(true, "apply")
	for _, failure := range []string{"hidden", "denied", "public", "redirect"} {
		mu.Lock()
		mode = failure
		count = writes
		mu.Unlock()
		run(false, "plan")
		run(false, "apply")
		before, _ := os.ReadFile(path)
		run(false, "export", "--forking")
		after, _ := os.ReadFile(path)
		if string(before) != string(after) {
			t.Fatal("failed export replaced file")
		}
		mu.Lock()
		afterWrites := writes
		mu.Unlock()
		if afterWrites != count {
			t.Fatal("unknown observation wrote")
		}
	}
	mu.Lock()
	mode = "denied"
	mu.Unlock()
	privateReport := filepath.Join(fixture.root, "forking-private.json")
	result, err := executeCLI(binary, fixture.work, fixture.root, env, []string{"action", "plan", privateReport, "status-only"}, false)
	if err != nil || result.runErr == nil || strings.Contains(result.stderr.String(), "CANARY") {
		t.Fatal(err, result.runErr)
	}
	data, _ = os.ReadFile(privateReport)
	if strings.Contains(string(data), "CANARY") || strings.Contains(string(data), "repository.allow_forking") {
		t.Fatal(string(data))
	}
	if err := writeSettingsFile(fixture.work, repositoryInput("description: kept\nallow_forking: false\n")); err != nil {
		t.Fatal(err)
	}
	output := run(false, "apply")
	if !strings.Contains(output, "earlier successful writes remain applied") {
		t.Fatal(output)
	}
	mu.Lock()
	mode = ""
	n := writes
	redirected := redirects
	mu.Unlock()
	if n != count+1 || redirected != 0 {
		t.Fatal("partial writes or redirect", n, count, redirected)
	}
	if err := writeSettingsFile(fixture.work, repositoryInput("allow_forking: false\narchived: false\n")); err != nil {
		t.Fatal(err)
	}
	run(true, "validate")
	mu.Lock()
	archived = true
	allowed = true
	count = writes
	mu.Unlock()
	plan = run(true, "plan")
	archiveChange := "settings.repository.archived: true → false"
	forkingChange := "settings.repository.allow_forking: true → false"
	if !strings.Contains(plan, archiveChange) || !strings.Contains(plan, forkingChange) || strings.Index(plan, archiveChange) > strings.Index(plan, forkingChange) {
		t.Fatal(plan)
	}
	run(true, "apply")
	run(true, "apply")
	mu.Lock()
	afterWrites := writes
	afterArchive := archived
	afterForking := allowed
	mu.Unlock()
	if afterWrites != count+2 || afterArchive || afterForking {
		t.Fatal("explicit unarchive-first composition", count, afterWrites, afterArchive, afterForking)
	}
}
