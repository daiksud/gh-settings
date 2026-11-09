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

func TestRepositoryVariablesProcess(t *testing.T) {
	binary := acceptanceBinary(t)
	var mu sync.Mutex
	variables := map[string]string{"UNMANAGED": "keep-external"}
	var requests []string
	redirect := false
	fixture, err := startHTTPFixture(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "/actions/variables") || r.URL.Path == "/unexpected" {
			requests = append(requests, r.Method+" "+r.URL.Path)
			if r.Header.Get("X-GitHub-Api-Version") != "2026-03-10" {
				t.Errorf("wrong Variables version: %q", r.Header.Get("X-GitHub-Api-Version"))
			}
			name := strings.TrimPrefix(r.URL.Path, "/repos/owner/current/actions/variables/")
			if redirect {
				w.Header().Set("Location", "/unexpected")
				w.WriteHeader(307)
				return
			}
			if r.Method == "GET" {
				if value, ok := variables[name]; ok {
					json.NewEncoder(w).Encode(map[string]string{"name": name, "value": value})
				} else {
					w.WriteHeader(404)
					fmt.Fprint(w, `{"message":"not found"}`)
				}
				return
			}
			var body map[string]string
			if json.NewDecoder(r.Body).Decode(&body) != nil {
				t.Error("invalid write body")
				w.WriteHeader(400)
				return
			}
			switch r.Method {
			case "POST":
				if r.URL.Path != "/repos/owner/current/actions/variables" || !reflect.DeepEqual(body, map[string]string{"name": "DISPLAY_NAME", "value": "日本語-v2"}) {
					t.Errorf("unexpected POST %s %v", r.URL.Path, body)
				}
				variables[body["name"]] = body["value"]
				w.WriteHeader(201)
			case "PATCH":
				if name != "DISPLAY_NAME" || !reflect.DeepEqual(body, map[string]string{"value": "updated-v3"}) {
					t.Errorf("unexpected PATCH %s %v", name, body)
				}
				variables[name] = body["value"]
				w.WriteHeader(204)
			default:
				t.Errorf("unexpected request %s %s", r.Method, r.URL)
				w.WriteHeader(400)
			}
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
			t.Errorf("unexpected request %s %s", r.Method, r.URL)
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
	declaration := func(value string) string {
		return "version: daiksud/gh-settings@v0\nrepository:\n  settings:\n    actions:\n      variables:\n        - name: display_name\n          value: " + value + "\n"
	}
	input := declaration("日本語-v2")
	if err := writeSettingsFile(fixture.work, input); err != nil {
		t.Fatal(err)
	}
	run := func(args []string, want string, fail bool) {
		t.Helper()
		result, err := executeCLI(binary, fixture.work, fixture.root, env, args, false)
		if err != nil || (result.runErr != nil) != fail || !fail && (result.stdout.String() != want || result.stderr.Len() != 0) {
			t.Fatalf("args=%v result=%+v err=%v", args, result, err)
		}
	}
	expect := func(want ...string) {
		t.Helper()
		mu.Lock()
		defer mu.Unlock()
		if !reflect.DeepEqual(requests, want) {
			t.Fatalf("requests=%v want=%v", requests, want)
		}
		requests = nil
	}
	path := "/repos/owner/current/actions/variables/DISPLAY_NAME"
	if err := writeSettingsFile(fixture.work, declaration("''")); err != nil {
		t.Fatal(err)
	}
	run([]string{"apply"}, "", true)
	expect()
	if err := writeSettingsFile(fixture.work, input); err != nil {
		t.Fatal(err)
	}
	run([]string{"plan"}, "repository.settings.actions.variables: create \"DISPLAY_NAME\" value=\"日本語-v2\"\n", false)
	expect("GET " + path)
	run([]string{"apply"}, "Applied repository settings.\n", false)
	expect("GET "+path, "POST /repos/owner/current/actions/variables", "GET "+path)
	if err := checkSettingsUnchanged(fixture.work, input); err != nil {
		t.Fatal(err)
	}
	run([]string{"apply"}, "Applied repository settings.\n", false)
	expect("GET " + path)
	input = declaration("updated-v3")
	if err := writeSettingsFile(fixture.work, input); err != nil {
		t.Fatal(err)
	}
	run([]string{"apply"}, "Applied repository settings.\n", false)
	expect("GET "+path, "PATCH "+path, "GET "+path)
	run([]string{"export", "--variable", "display_name"}, "", false)
	expect("GET " + path)
	output, err := os.ReadFile(filepath.Join(fixture.work, ".github/settings.yml"))
	if err != nil || !strings.Contains(string(output), "DISPLAY_NAME") || !strings.Contains(string(output), "updated-v3") || strings.Contains(string(output), "UNMANAGED") {
		t.Fatal(string(output), err)
	}
	run([]string{"export", "--variable", "MISSING"}, "", true)
	expect("GET /repos/owner/current/actions/variables/MISSING")
	if err := checkSettingsUnchanged(fixture.work, string(output)); err != nil {
		t.Fatal(err)
	}
	run([]string{"export"}, "", false)
	expect()
	mu.Lock()
	if variables["UNMANAGED"] != "keep-external" || len(variables) != 2 {
		t.Fatal(variables)
	}
	redirect = true
	mu.Unlock()
	if err := writeSettingsFile(fixture.work, input); err != nil {
		t.Fatal(err)
	}
	run([]string{"plan"}, "", true)
	expect("GET " + path)
}
