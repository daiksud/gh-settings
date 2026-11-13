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

func TestPullRequestExemptionsProcess(t *testing.T) {
	binary := acceptanceBinary(t)
	const endpoint = "/repos/owner/current/interaction-limits/pulls/bypass-list"
	var mu sync.Mutex
	members := map[string]int64{"bob": 2}
	requests := []string{}
	denied := false
	failAdd := false
	fixture, err := startHTTPFixture(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == endpoint {
			requests = append(requests, r.Method)
			if r.Header.Get("X-GitHub-Api-Version") != "2026-03-10" {
				t.Error("wrong API version")
			}
			if denied {
				w.WriteHeader(403)
				fmt.Fprint(w, `{"message":"denied"}`)
				return
			}
			if r.Method == "GET" {
				var users []map[string]any = []map[string]any{}
				for login, id := range members {
					users = append(users, map[string]any{"login": login, "id": id, "type": "User"})
				}
				json.NewEncoder(w).Encode(users)
				return
			}
			var body map[string][]string
			if json.NewDecoder(r.Body).Decode(&body) != nil || len(body) != 1 || len(body["users"]) == 0 {
				t.Error("invalid body", body)
				w.WriteHeader(400)
				return
			}
			if r.Method == "PUT" {
				if failAdd {
					w.WriteHeader(422)
					fmt.Fprint(w, `{"message":"invalid"}`)
					return
				}
				for _, login := range body["users"] {
					members[login] = int64(len(login) + 10)
				}
			} else if r.Method == "DELETE" {
				for _, login := range body["users"] {
					delete(members, login)
				}
			} else {
				t.Error("unexpected method", r.Method)
				w.WriteHeader(400)
				return
			}
			w.WriteHeader(204)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/users/") {
			login := strings.TrimPrefix(r.URL.Path, "/users/")
			requests = append(requests, "USER "+login)
			json.NewEncoder(w).Encode(map[string]any{"login": login, "id": len(login) + 10, "type": "User"})
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
			fmt.Fprint(w, planResponse)
		case "/repos/owner/current/topics":
			fmt.Fprint(w, `{"names":[]}`)
		case "/repos/owner/current/labels":
			fmt.Fprint(w, `[]`)
		default:
			t.Errorf("unselected endpoint %s %s", r.Method, r.URL)
			w.WriteHeader(400)
		}
	}, false, "/interaction-limits/pulls/bypass-list", "/labels", "/topics")
	if err != nil {
		t.Fatal(err)
	}
	defer fixture.close()
	env := exportFixtureEnv(fixture.cfg)
	if err := setupOrdinaryOrigin(fixture.work, env); err != nil {
		t.Fatal(err)
	}
	config := func(value string) string {
		return "version: daiksud/gh-settings@v0\nrepository:\n  settings:\n    interaction_limits:\n      pull_request_limits:\n        exemption_list: " + value + "\n"
	}
	write := func(value string) {
		t.Helper()
		if err := writeSettingsFile(fixture.work, config(value)); err != nil {
			t.Fatal(err)
		}
	}
	run := func(args []string, fail bool) *executionResult {
		t.Helper()
		result, err := executeCLI(binary, fixture.work, fixture.root, env, args, false)
		if err != nil || (result.runErr != nil) != fail {
			t.Fatalf("args=%v err=%v result=%+v", args, err, result)
		}
		return result
	}
	expect := func(want ...string) {
		t.Helper()
		mu.Lock()
		defer mu.Unlock()
		if !reflect.DeepEqual(requests, want) && !(len(requests) == 0 && len(want) == 0) {
			t.Fatalf("requests=%v want=%v", requests, want)
		}
		requests = nil
	}
	write("null")
	run([]string{"apply"}, true)
	expect()
	write("[Alice]")
	run([]string{"validate"}, false)
	expect()
	plan := run([]string{"plan"}, false)
	if !strings.Contains(plan.stdout.String(), `exemption_list: ["bob"] → ["alice"]`) {
		t.Fatal(plan.stdout.String())
	}
	expect("GET")
	reportPath := filepath.Join(fixture.root, "exemptions.json")
	run([]string{"action", "plan", reportPath}, false)
	expect("GET")
	data, err := os.ReadFile(reportPath)
	if err != nil || !strings.Contains(string(data), `"pull_request_exemptions"`) || !strings.Contains(string(data), `"managed":1`) {
		t.Fatal(string(data), err)
	}
	run([]string{"apply"}, false)
	expect("GET", "USER alice", "DELETE", "GET", "PUT", "GET")
	run([]string{"apply"}, false)
	expect("GET")
	run([]string{"export"}, false)
	expect("GET")
	exported, err := os.ReadFile(filepath.Join(fixture.work, ".github/settings.yml"))
	if err != nil || !strings.Contains(string(exported), "exemption_list:") || !strings.Contains(string(exported), "alice") || strings.Contains(string(exported), "maximum_open_pull_requests_per_user") {
		t.Fatal(string(exported), err)
	}
	mu.Lock()
	denied = true
	mu.Unlock()
	run([]string{"export"}, false)
	expect("GET")
	if err := checkSettingsUnchanged(fixture.work, string(exported)); err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	denied = false
	mu.Unlock()
	write("[]")
	run([]string{"apply"}, false)
	expect("GET", "DELETE", "GET")
	run([]string{"export"}, false)
	expect("GET")
	mu.Lock()
	members = map[string]int64{"bob": 2}
	failAdd = true
	mu.Unlock()
	write("[alice]")
	failed := run([]string{"apply"}, true)
	expect("GET", "USER alice", "DELETE", "GET", "PUT")
	if !strings.Contains(withoutExportInfo(failed.stderr.String()), "earlier successful writes remain applied") {
		t.Fatal(failed.stderr.String())
	}
	mu.Lock()
	defer mu.Unlock()
	if len(members) != 0 {
		t.Fatal("confirmed removal was not retained", members)
	}
}
