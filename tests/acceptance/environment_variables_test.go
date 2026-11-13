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

func TestEnvironmentVariablesProcess(t *testing.T) {
	environment := "preview/path name"
	environmentPath := "/repos/owner/current/environments/" + environment
	endpoint := environmentPath + "/variables"
	binary := acceptanceBinary(t)
	var mu sync.Mutex
	variables := map[string]string{"UNMANAGED": "keep-external"}
	var requests []string
	apiCalls := 0
	redirect := false
	missingEnvironment := false
	fixture, err := startHTTPFixture(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		apiCalls++
		w.Header().Set("Content-Type", "application/json")
		if r.Method == "GET" && r.URL.Path == "/repos/owner/current/environments" {
			json.NewEncoder(w).Encode(map[string]any{"total_count": 1, "environments": []map[string]any{{"id": 123, "name": environment}}})
			return
		}
		if r.URL.Path == environmentPath {
			if missingEnvironment {
				w.WriteHeader(404)
				fmt.Fprint(w, `{"message":"not found"}`)
				return
			}
			if r.Method != "GET" {
				t.Errorf("unexpected policy write %s", r.Method)
				w.WriteHeader(400)
				return
			}
			if !strings.Contains(r.URL.EscapedPath(), "preview%2Fpath%20name") {
				t.Errorf("environment not escaped as one segment: %s", r.URL.EscapedPath())
			}
			fmt.Fprint(w, `{"id":123,"name":"preview/path name"}`)
			return
		}
		if strings.HasPrefix(r.URL.Path, endpoint) || r.URL.Path == "/unexpected" {
			requests = append(requests, r.Method+" "+r.URL.Path)
			if r.Header.Get("X-GitHub-Api-Version") != "2026-03-10" {
				t.Errorf("wrong Variables version: %q", r.Header.Get("X-GitHub-Api-Version"))
			}
			name := strings.TrimPrefix(r.URL.Path, endpoint+"/")
			if redirect {
				w.Header().Set("Location", "/unexpected")
				w.WriteHeader(307)
				return
			}
			if r.Method == "GET" && r.URL.Path == endpoint {
				values := []map[string]string{}
				for name, value := range variables {
					values = append(values, map[string]string{"name": name, "value": value})
				}
				json.NewEncoder(w).Encode(map[string]any{"total_count": len(values), "variables": values})
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
				if r.URL.Path != endpoint || !reflect.DeepEqual(body, map[string]string{"name": "DISPLAY_NAME", "value": "日本語-v2"}) {
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
	}, false, "/environments", "/environments/*/variables", "/environments/", "/labels", "/topics")
	if err != nil {
		t.Fatal(err)
	}
	defer fixture.close()
	env := exportFixtureEnv(fixture.cfg)
	if err := setupOrdinaryOrigin(fixture.work, env); err != nil {
		t.Fatal(err)
	}
	declaration := func(value string) string {
		return "version: daiksud/gh-settings@v0\nrepository:\n  settings:\n    environments:\n      - name: preview/path name\n        variables:\n          - name: display_name\n            value: " + value + "\n"
	}
	input := declaration("日本語-v2")
	if err := writeSettingsFile(fixture.work, input); err != nil {
		t.Fatal(err)
	}
	lastExport := false
	run := func(args []string, want string, fail bool) {
		t.Helper()
		lastExport = len(args) > 0 && args[0] == "export"
		result, err := executeCLI(binary, fixture.work, fixture.root, env, args, false)
		if err != nil || (result.runErr != nil) != fail || !fail && (result.stdout.String() != want || hasNonInfoDiagnostic(result.stderr.String())) {
			t.Fatalf("args=%v result=%+v err=%v", args, result, err)
		}
	}
	expect := func(want ...string) {
		if lastExport {
			want = []string{"GET " + endpoint}
			if missingEnvironment {
				want = nil
			}
		}
		t.Helper()
		mu.Lock()
		defer mu.Unlock()
		if !reflect.DeepEqual(requests, want) {
			t.Fatalf("requests=%v want=%v", requests, want)
		}
		requests = nil
	}
	path := endpoint + "/DISPLAY_NAME"
	if err := writeSettingsFile(fixture.work, declaration("''")); err != nil {
		t.Fatal(err)
	}
	run([]string{"apply"}, "", true)
	expect()
	if err := writeSettingsFile(fixture.work, input); err != nil {
		t.Fatal(err)
	}
	run([]string{"validate"}, "", false)
	mu.Lock()
	callsBeforePlan := apiCalls
	mu.Unlock()
	if callsBeforePlan != 0 {
		t.Fatal("local validation made API requests", callsBeforePlan)
	}
	run([]string{"plan"}, "repository.settings.environments[\"preview/path name\"].variables: create \"DISPLAY_NAME\" value=\"日本語-v2\"\n", false)
	expect("GET " + path)
	reportPath := filepath.Join(fixture.root, "variables-plan.json")
	run([]string{"action", "plan", reportPath}, "", false)
	expect("GET " + path)
	reportData, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatal(err)
	}
	var report map[string]json.RawMessage
	if err := json.Unmarshal(reportData, &report); err != nil {
		t.Fatal(err)
	}
	var changes []struct {
		Environment string  `json:"environment"`
		Name        string  `json:"name"`
		After       *string `json:"after"`
	}
	if json.Unmarshal(report["environment_variables"], &changes) != nil || len(changes) != 1 || changes[0].Environment != environment || changes[0].Name != "DISPLAY_NAME" || changes[0].After == nil || *changes[0].After != "日本語-v2" || string(report["managed"]) != "1" {
		t.Fatalf("report=%s", reportData)
	}
	run([]string{"apply"}, "Applied repository settings.\n", false)
	expect("GET "+path, "POST "+endpoint, "GET "+path)
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
	run([]string{"export"}, "", false)
	expect("GET " + path)
	output, err := os.ReadFile(filepath.Join(fixture.work, ".github/settings.yml"))
	if err != nil || !strings.Contains(string(output), environment) || strings.Contains(string(output), "deployment_protection_rules") || !strings.Contains(string(output), "DISPLAY_NAME") || !strings.Contains(string(output), "updated-v3") || !strings.Contains(string(output), "UNMANAGED") {
		t.Fatal(string(output), err)
	}
	mu.Lock()
	missingEnvironment = true
	mu.Unlock()
	run([]string{"export"}, "", false)
	expect()
	if err := checkSettingsUnchanged(fixture.work, string(output)); err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	missingEnvironment = false
	mu.Unlock()
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
