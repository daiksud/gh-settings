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
	for _, family := range []string{"actions", "agents"} {
		t.Run(family, func(t *testing.T) { testRepositoryVariablesProcess(t, family) })
	}
}

func testRepositoryVariablesProcess(t *testing.T, family string) {
	endpoint := "/repos/owner/current/" + family + "/variables"
	binary := acceptanceBinary(t)
	var mu sync.Mutex
	variables := map[string]string{"UNMANAGED": "keep-external"}
	var requests []string
	apiCalls := 0
	redirect := false
	fixture, err := startHTTPFixture(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		apiCalls++
		w.Header().Set("Content-Type", "application/json")
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
	}, false, "/"+family+"/variables", "/labels", "/topics")
	if err != nil {
		t.Fatal(err)
	}
	defer fixture.close()
	env := exportFixtureEnv(fixture.cfg)
	if err := setupOrdinaryOrigin(fixture.work, env); err != nil {
		t.Fatal(err)
	}
	declaration := func(value string) string {
		return "version: daiksud/gh-settings@v0\nrepository:\n  settings:\n    secrets_and_variables:\n      " + family + ":\n        variables:\n          - name: display_name\n            value: " + value + "\n"
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
	run([]string{"plan"}, "repository.settings.secrets_and_variables."+family+".variables: create \"DISPLAY_NAME\" value=\"日本語-v2\"\n", false)
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
	key, otherKey := "variables", "agents_variables"
	if family == "agents" {
		key, otherKey = otherKey, key
	}
	var changes []struct {
		Name   string  `json:"name"`
		Before *string `json:"before"`
		After  string  `json:"after"`
	}
	if json.Unmarshal(report[key], &changes) != nil || len(changes) != 1 || changes[0].Name != "DISPLAY_NAME" || changes[0].Before != nil || changes[0].After != "日本語-v2" || report[otherKey] != nil || string(report["outcome"]) != `"changed"` || string(report["managed"]) != "1" {
		t.Fatalf("wrong family report: %s", reportData)
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
	if err != nil || !strings.Contains(string(output), "DISPLAY_NAME") || !strings.Contains(string(output), "updated-v3") || !strings.Contains(string(output), "UNMANAGED") {
		t.Fatal(string(output), err)
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

func TestSameNamedVariablesAcrossRepositoryAndEnvironmentScopes(t *testing.T) {
	const production = "/environments/production/variables"
	remote := map[string]string{
		"/actions/variables":              "actions-old",
		"/agents/variables":               "agents-old",
		production:                        "production-old",
		"/environments/staging/variables": "staging-old",
	}
	var mu sync.Mutex
	fixture, err := startHTTPFixture(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		path := strings.TrimPrefix(r.URL.Path, "/repos/owner/current")
		if r.Method == http.MethodDelete && path == production+"/REGION" {
			delete(remote, production)
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if servePRAccessFixtureRead(w, r) || serveIssueCreationPolicyFixtureRead(w, r) || serveSponsorshipsFixtureRead(w, r) {
			return
		}
		if isMergeMethodsFixtureRead(r) {
			fmt.Fprint(w, mergeMethodsFixtureResponse)
			return
		}
		if r.Method != http.MethodGet {
			t.Errorf("unexpected write %s %s", r.Method, path)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		switch path {
		case "":
			fmt.Fprint(w, planResponse)
		case "/environments":
			fmt.Fprint(w, `{"total_count":2,"environments":[{"id":1,"name":"production"},{"id":2,"name":"staging"}]}`)
		case "/environments/production":
			fmt.Fprint(w, `{"id":1,"name":"production"}`)
		case "/environments/staging":
			fmt.Fprint(w, `{"id":2,"name":"staging"}`)
		default:
			base, named := strings.CutSuffix(path, "/REGION")
			if value, ok := remote[base]; ok {
				variable := map[string]string{"name": "REGION", "value": value}
				if named {
					json.NewEncoder(w).Encode(variable)
				} else {
					json.NewEncoder(w).Encode(map[string]any{"total_count": 1, "variables": []map[string]string{variable}})
				}
			} else if path == production {
				fmt.Fprint(w, `{"total_count":0,"variables":[]}`)
			} else {
				w.WriteHeader(http.StatusForbidden)
			}
		}
	}, false, "/actions/variables", "/agents/variables", "/environments", "/environments/*/variables")
	if err != nil {
		t.Fatal(err)
	}
	defer fixture.close()
	env := exportFixtureEnv(fixture.cfg)
	if err := setupOrdinaryOrigin(fixture.work, env); err != nil {
		t.Fatal(err)
	}
	binary := acceptanceBinary(t)
	run := func(args ...string) string {
		t.Helper()
		result, err := executeCLI(binary, fixture.work, fixture.root, env, args, false)
		if err != nil || result.runErr != nil {
			t.Fatalf("%v: %v %v %s", args, err, result.runErr, result.stderr.String())
		}
		return result.stdout.String()
	}

	run("export")
	config := repositoryInput(`settings:
  environments:
    - name: production
      variables: [{name: REGION, value: production-old}]
    - name: staging
      variables: [{name: REGION, value: staging-old}]
  secrets_and_variables:
    actions: {variables: [{name: REGION, value: actions-old}]}
    agents: {variables: [{name: REGION, value: agents-old}]}
`)
	assertExportRetains(t, fixture, config, "repository", "settings")
	run("validate")
	if err := writeSettingsFile(fixture.work, strings.ReplaceAll(config, "-old", "-new")); err != nil {
		t.Fatal(err)
	}
	plan := run("plan")
	for path, scope := range map[string]string{
		"secrets_and_variables.actions.variables": "actions",
		"secrets_and_variables.agents.variables":  "agents",
		`environments["production"].variables`:    "production",
		`environments["staging"].variables`:       "staging",
	} {
		want := fmt.Sprintf("repository.settings.%s: update \"REGION\" value: \"%s-old\" → \"%s-new\"\n", path, scope, scope)
		if !strings.Contains(plan, want) {
			t.Fatalf("missing scoped change %q in %s", want, plan)
		}
	}
	if err := writeSettingsFile(fixture.work, repositoryInput("settings: {environments: [{name: production, variables: []}]}\n")); err != nil {
		t.Fatal(err)
	}
	run("apply", "--prune", "environment-variables")
	mu.Lock()
	defer mu.Unlock()
	if _, ok := remote[production]; ok || len(remote) != 3 {
		t.Fatalf("expected only production REGION to be removed: %v", remote)
	}
}
