//go:build acceptance

package acceptance

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"

	"go.yaml.in/yaml/v4"
)

func TestTopicsPlanApplyAcceptance(t *testing.T) {
	binary := acceptanceBinary(t)
	for _, test := range []struct {
		name, command, input, body, want, errorText string
		status                                      int
	}{
		{name: "sorted plan", command: "plan", input: "[z, a]", body: `{"names":["old"]}`, want: "repository.about.topics: [\"old\"] → [\"a\",\"z\"]\n"},
		{name: "order independent", command: "plan", input: "[z, a]", body: `{"names":["a","z"]}`, want: "No differences for managed repository settings.\n"},
		{name: "clear plan", command: "plan", input: "[]", body: `{"names":["old"]}`, want: "repository.about.topics: [\"old\"] → []\n"},
		{name: "null rejected", command: "apply", input: "null", errorText: "repository.about.topics must be a string array"},
		{name: "invalid entry", command: "plan", input: "[true]", errorText: "repository.about.topics must contain strings"},
		{name: "empty group", command: "apply", input: "", want: "No managed repository settings.\n"},
	} {
		t.Run(test.name, func(t *testing.T) {
			var mu sync.Mutex
			var requests []string
			fixture, err := startHTTPFixture(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				var body strings.Builder
				if r.Body != nil {
					_, _ = io.Copy(&body, r.Body)
				}
				requests = append(requests, r.Method+" "+r.URL.Path+" "+body.String())
				w.Header().Set("Content-Type", "application/json")
				if test.status >= 300 {
					w.Header().Set("Location", "https://api.github.com/repos/other/wrong/topics")
					w.WriteHeader(test.status)
					fmt.Fprint(w, `{"message":"fixture denied"}`)
					return
				}
				if r.Method == "GET" {
					fmt.Fprint(w, test.body)
				} else {
					fmt.Fprint(w, `{}`)
				}
			}, false, "/labels", "/topics")
			if err != nil {
				t.Fatal(err)
			}
			defer fixture.close()
			env := exportFixtureEnv(fixture.cfg)
			if err := setupOrdinaryOrigin(fixture.work, env); err != nil {
				t.Fatal(err)
			}
			input := repositoryInput("about: {}\n")
			if test.input != "" {
				input = aboutInput("topics: " + test.input + "\n")
			}
			if err := writeSettingsFile(fixture.work, input); err != nil {
				t.Fatal(err)
			}
			result, err := executeCLI(binary, fixture.work, fixture.root, env, []string{test.command}, false)
			if err != nil {
				t.Fatal(err)
			}
			if test.errorText != "" {
				if result.runErr == nil || !strings.Contains(withoutExportInfo(result.stderr.String()), test.errorText) || result.stdout.Len() != 0 {
					t.Fatalf("result=%+v", result)
				}
			} else {
				want := test.want
				if test.command == "apply" && test.input != "" {
					want = "Applied repository settings.\n"
				}
				if result.runErr != nil || result.stdout.String() != want || hasNonInfoDiagnostic(result.stderr.String()) {
					t.Fatalf("result=%+v want=%q", result, want)
				}
			}
			mu.Lock()
			defer mu.Unlock()
			count := 1
			if test.input == "" || test.name == "null rejected" || test.name == "invalid entry" {
				count = 0
			}
			if len(requests) != count {
				t.Fatalf("requests=%v want count=%d", requests, count)
			}
			if count == 1 {
				method := "GET"
				body := ""
				if test.command == "apply" {
					method = "PUT"
					body = test.want
					if test.errorText != "" {
						body = `{"names":[]}`
					}
				}
				want := method + " /repos/owner/current/topics " + body
				if requests[0] != want {
					t.Fatalf("request=%q want=%q", requests[0], want)
				}
			}
			if err := checkSettingsUnchanged(fixture.work, input); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestTopicsExportAndActionReport(t *testing.T) {
	binary := acceptanceBinary(t)
	for _, fail := range []bool{false, true} {
		t.Run(fmt.Sprint(fail), func(t *testing.T) {
			var mu sync.Mutex
			var paths []string
			fixture, err := startHTTPFixture(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				paths = append(paths, r.Method+" "+r.URL.Path)
				w.Header().Set("Content-Type", "application/json")
				if servePRAccessFixtureRead(w, r) {
					return
				}
				if serveIssueCreationPolicyFixtureRead(w, r) || serveSponsorshipsFixtureRead(w, r) {
					return
				}
				if isMergeMethodsFixtureRead(r) {
					fmt.Fprint(w, mergeMethodsFixtureResponse)
					return
				}
				if r.Method != "GET" {
					t.Errorf("unexpected write: %s", r.Method)
					w.WriteHeader(400)
					return
				}
				if r.URL.Path == "/repos/owner/current" {
					fmt.Fprint(w, planResponse)
					return
				}
				if r.URL.Path == "/repos/owner/current/labels" {
					fmt.Fprint(w, `[]`)
					return
				}
				if r.URL.Path != "/repos/owner/current/topics" {
					t.Errorf("unexpected path: %s", r.URL.Path)
					w.WriteHeader(400)
					return
				}
				if fail {
					w.WriteHeader(403)
					fmt.Fprint(w, `{"message":"fixture denied"}`)
					return
				}
				fmt.Fprint(w, `{"names":["z","a"]}`)
			}, false, "/labels", "/topics")
			if err != nil {
				t.Fatal(err)
			}
			defer fixture.close()
			env := exportFixtureEnv(fixture.cfg)
			if err := setupOrdinaryOrigin(fixture.work, env); err != nil {
				t.Fatal(err)
			}
			before := "version: daiksud/gh-settings@v0\nrepository: {}\n"
			if err := writeSettingsFile(fixture.work, before); err != nil {
				t.Fatal(err)
			}
			result, err := executeCLI(binary, fixture.work, fixture.root, env, []string{"export"}, false)
			if err != nil {
				t.Fatal(err)
			}
			if fail {
				if result.runErr != nil || !strings.Contains(result.stderr.String(), "info: export ") || !strings.Contains(result.stderr.String(), "HTTP 403") {
					t.Fatalf("result=%+v", result)
				}
				assertExportRetains(t, fixture, before, "repository", "about", "topics")
				return
			}
			if result.runErr != nil || result.stdout.Len() != 0 || hasNonInfoDiagnostic(result.stderr.String()) {
				t.Fatalf("result=%+v", result)
			}
			data, err := os.ReadFile(filepath.Join(fixture.work, ".github", "settings.yml"))
			if err != nil {
				t.Fatal(err)
			}
			var document struct {
				Version    string `yaml:"version"`
				Repository struct {
					About struct {
						Topics []string `yaml:"topics"`
					} `yaml:"about"`
				} `yaml:"repository"`
			}
			if err := yaml.Unmarshal(data, &document); err != nil {
				t.Fatal(err)
			}
			if document.Version != "daiksud/gh-settings@v0" || !reflect.DeepEqual(document.Repository.About.Topics, []string{"a", "z"}) {
				t.Fatalf("About topics export=%s", data)
			}
			input := aboutInput("topics: []\n")
			if err := writeSettingsFile(fixture.work, input); err != nil {
				t.Fatal(err)
			}
			report := filepath.Join(fixture.root, "topics.json")
			result, err = executeCLI(binary, fixture.work, fixture.root, env, []string{"action", "plan", report}, false)
			if err != nil || result.runErr != nil || hasNonInfoDiagnostic(result.stderr.String()) {
				t.Fatalf("plan=%+v error=%v", result, err)
			}
			data, err = os.ReadFile(report)
			if err != nil {
				t.Fatal(err)
			}
			var got map[string]any
			if err := json.Unmarshal(data, &got); err != nil {
				t.Fatal(err)
			}
			want := map[string]any{"outcome": "changed", "target": "owner/current", "managed": float64(1), "changes": []any{map[string]any{"field": "topics.names", "before": []any{"a", "z"}, "after": []any{}}}}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("report=%v", got)
			}
			mu.Lock()
			defer mu.Unlock()
			if !reflect.DeepEqual(paths, []string{"GET /repos/owner/current", "POST /graphql", "POST /graphql", "POST /graphql", "POST /graphql", "GET /repos/owner/current/topics", "GET /repos/owner/current", "GET /repos/owner/current/labels", "GET /repos/owner/current", "GET /repos/owner/current", "GET /repos/owner/current", "GET /repos/owner/current", "GET /repos/owner/current", "GET /repos/owner/current", "GET /repos/owner/current", "GET /repos/owner/current", "GET /repos/owner/current/topics"}) {
				t.Fatalf("requests=%v", paths)
			}
		})
	}
}

func TestTopicsApplyReconciliationAcceptance(t *testing.T) {
	binary := acceptanceBinary(t)
	for _, test := range []struct {
		name, input, initial, after, errorText string
		readStatus, putStatus, verifyStatus    int
		wantMethods                            []string
		repeat, repositoryWrite                bool
	}{
		{name: "exact match", input: "[a, z]", initial: `{"names":["a","z"]}`, wantMethods: []string{"GET topics", "GET labels"}},
		{name: "reordered match", input: "[z, a]", initial: `{"names":["a","z"]}`, wantMethods: []string{"GET topics", "GET labels"}},
		{name: "empty match", input: "[]", initial: `{"names":[]}`, wantMethods: []string{"GET topics", "GET labels"}},
		{name: "replace and repeat", input: "[z, a]", initial: `{"names":["old"]}`, after: `{"names":["z","a"]}`, repeat: true, wantMethods: []string{"GET topics", "PUT topics", "GET topics", "GET labels", "GET topics", "GET labels"}},
		{name: "clear and repeat", input: "[]", initial: `{"names":["old"]}`, after: `{"names":[]}`, repeat: true, wantMethods: []string{"GET topics", "PUT topics", "GET topics", "GET labels", "GET topics", "GET labels"}},
		{name: "missing names", input: "[]", initial: `{}`, errorText: "missing topics names array", wantMethods: []string{"GET topics"}},
		{name: "null names", input: "[]", initial: `{"names":null}`, errorText: "missing topics names array", wantMethods: []string{"GET topics"}},
		{name: "invalid name", input: "[]", initial: `{"names":["UPPER"]}`, errorText: "decode repository topics", wantMethods: []string{"GET topics"}},
		{name: "denied read", input: "[]", readStatus: 403, errorText: "403", wantMethods: []string{"GET topics"}},
		{name: "redirected read", input: "[]", readStatus: 302, errorText: "302", wantMethods: []string{"GET topics"}},
		{name: "rejected PUT", input: "[]", initial: `{"names":["old"]}`, putStatus: 422, errorText: "writes may already have applied", wantMethods: []string{"GET topics", "PUT topics"}},
		{name: "denied verification", input: "[]", initial: `{"names":["old"]}`, verifyStatus: 403, errorText: "verify repository topics", wantMethods: []string{"GET topics", "PUT topics", "GET topics"}},
		{name: "mismatching verification", input: "[]", initial: `{"names":["old"]}`, after: `{"names":["old"]}`, errorText: "does not match declared membership", wantMethods: []string{"GET topics", "PUT topics", "GET topics"}},
		{name: "earlier repository write", input: "[]", readStatus: 403, repositoryWrite: true, errorText: "earlier repository settings update remains applied", wantMethods: []string{"GET current", "PATCH current", "GET current", "GET topics"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			var mu sync.Mutex
			var requests []string
			put := false
			var repositoryState map[string]any
			if err := json.Unmarshal([]byte(planResponse), &repositoryState); err != nil {
				t.Fatal(err)
			}
			fixture, err := startHTTPFixture(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				kind := filepath.Base(r.URL.Path)
				requests = append(requests, r.Method+" "+kind)
				w.Header().Set("Content-Type", "application/json")
				if r.URL.Path == "/repos/owner/current" && test.repositoryWrite {
					if r.Method == "PATCH" {
						var payload map[string]any
						if err := json.NewDecoder(r.Body).Decode(&payload); err != nil || !reflect.DeepEqual(payload, map[string]any{"has_issues": false}) {
							t.Errorf("repository PATCH=%v error=%v", payload, err)
						}
						for key, value := range payload {
							repositoryState[key] = value
						}
					} else if r.Method != "GET" {
						t.Errorf("unexpected repository method: %s", r.Method)
					}
					_ = json.NewEncoder(w).Encode(repositoryState)
					return
				}
				if r.URL.Path == "/repos/owner/current/labels" && r.Method == "GET" {
					fmt.Fprint(w, `[{"id":7,"name":"bug","color":"abcdef","description":""}]`)
					return
				}
				if r.URL.Path != "/repos/owner/current/topics" {
					t.Errorf("unexpected request: %s %s", r.Method, r.URL)
					w.WriteHeader(400)
					return
				}
				body, status := test.initial, test.readStatus
				if r.Method == "PUT" {
					put = true
					payload, _ := io.ReadAll(r.Body)
					want := `{"names":[]}`
					if test.input != "[]" {
						want = `{"names":["a","z"]}`
					}
					if string(payload) != want {
						t.Errorf("PUT=%s want=%s", payload, want)
					}
					// Even a matching acknowledgement must be followed by an independent GET.
					body, status = want, test.putStatus
				} else if r.Method == "GET" {
					if put {
						body, status = test.after, test.verifyStatus
					}
				} else {
					t.Errorf("unexpected method: %s", r.Method)
					w.WriteHeader(400)
					return
				}
				if status >= 300 {
					w.Header().Set("Location", "https://api.github.com/repos/other/wrong/topics")
					w.WriteHeader(status)
					fmt.Fprint(w, `{"message":"fixture denied"}`)
					return
				}
				fmt.Fprint(w, body)
			}, false, "/labels", "/topics")
			if err != nil {
				t.Fatal(err)
			}
			defer fixture.close()
			env := exportFixtureEnv(fixture.cfg)
			if err := setupOrdinaryOrigin(fixture.work, env); err != nil {
				t.Fatal(err)
			}
			contents := "about: {topics: " + test.input + "}\nlabels: [{id: 7, name: bug, color: '#abcdef'}]\n"
			if test.repositoryWrite {
				contents += "settings: {general: {features: {issues: false}}}\n"
			}
			input := repositoryInput(contents)
			if err := writeSettingsFile(fixture.work, input); err != nil {
				t.Fatal(err)
			}
			attempts := 1
			if test.repeat {
				attempts = 2
			}
			for range attempts {
				result, err := executeCLI(binary, fixture.work, fixture.root, env, []string{"apply"}, false)
				if err != nil {
					t.Fatal(err)
				}
				if test.errorText != "" {
					if result.runErr == nil || result.stdout.Len() != 0 || !strings.Contains(result.stderr.String(), test.errorText) {
						t.Fatalf("result=%+v", result)
					}
					if !put && !test.repositoryWrite && strings.Contains(result.stderr.String(), "writes may already have applied") {
						t.Fatalf("preflight claimed writes: %s", result.stderr.String())
					}
				} else if result.runErr != nil || result.stdout.String() != "Applied repository settings.\n" || hasNonInfoDiagnostic(result.stderr.String()) {
					t.Fatalf("result=%+v", result)
				}
			}
			mu.Lock()
			defer mu.Unlock()
			if !reflect.DeepEqual(requests, test.wantMethods) {
				t.Fatalf("requests=%v want=%v", requests, test.wantMethods)
			}
			if err := checkSettingsUnchanged(fixture.work, input); err != nil {
				t.Fatal(err)
			}
		})
	}
}
