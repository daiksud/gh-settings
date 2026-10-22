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

	"go.yaml.in/yaml/v4"
)

func TestLabelsPlanAndApplyAcceptance(t *testing.T) {
	binary := acceptanceBinary(t)
	current := `[{"name":"keep","color":"abcdef","description":"external"},{"name":"old","color":"123456","description":null}]`
	for _, test := range []struct {
		name, command, declaration, body, want, errorText string
		requests                                          []string
		failMethod                                        string
		status                                            int
	}{
		{name: "omitted", command: "apply", declaration: `{}`, want: "No managed repository settings.\n"},
		{name: "empty clear plan", command: "plan", declaration: `{labels: []}`, body: current, want: "settings.labels: delete \"keep\" (removes issue/PR associations)\nsettings.labels: delete \"old\" (removes issue/PR associations)\n", requests: []string{"GET /repos/owner/current/labels "}},
		{name: "name identity and omitted attrs", command: "apply", declaration: `{labels: [{name: KEEP}, {name: old}]}`, body: current, want: "Applied repository settings.\n", requests: []string{"GET /repos/owner/current/labels "}},
		{name: "create update delete", command: "apply", declaration: `{labels: [{name: keep, description: ""}, {name: new}]}`, body: current, want: "Applied repository settings.\n", requests: []string{"GET /repos/owner/current/labels ", `PATCH /repos/owner/current/labels/keep {"description":""}`, `POST /repos/owner/current/labels {"name":"new"}`, "DELETE /repos/owner/current/labels/old "}},
		{name: "name change creates and deletes", command: "apply", declaration: `{labels: [{name: replacement, color: AABBCC}]}`, body: `[{"name":"original","color":"abcdef","description":""}]`, want: "Applied repository settings.\n", requests: []string{"GET /repos/owner/current/labels ", `POST /repos/owner/current/labels {"name":"replacement","color":"aabbcc"}`, "DELETE /repos/owner/current/labels/original "}},
		{name: "clear all", command: "apply", declaration: `{labels: []}`, body: current, want: "Applied repository settings.\n", requests: []string{"GET /repos/owner/current/labels ", "DELETE /repos/owner/current/labels/keep ", "DELETE /repos/owner/current/labels/old "}},
		{name: "malformed state", command: "apply", declaration: `{labels: []}`, body: `[{"name":"old"}]`, errorText: "missing color", requests: []string{"GET /repos/owner/current/labels "}},
		{name: "invalid null before API", command: "apply", declaration: `{labels: null}`, errorText: "array of label mappings"},
		{name: "duplicate before API", command: "plan", declaration: `{labels: [{name: bug}, {name: BUG}]}`, errorText: "duplicate label"},
		{name: "create failure stops delete", command: "apply", declaration: `{labels: [{name: new}]}`, body: current, failMethod: "POST", status: 403, errorText: "403", requests: []string{"GET /repos/owner/current/labels ", `POST /repos/owner/current/labels {"name":"new"}`}},
		{name: "update failure stops delete", command: "apply", declaration: `{labels: [{name: keep, description: ""}]}`, body: current, failMethod: "PATCH", status: 422, errorText: "422", requests: []string{"GET /repos/owner/current/labels ", `PATCH /repos/owner/current/labels/keep {"description":""}`}},
		{name: "post redirect", command: "apply", declaration: `{labels: [{name: new}]}`, body: current, failMethod: "POST", status: 302, errorText: "302", requests: []string{"GET /repos/owner/current/labels ", `POST /repos/owner/current/labels {"name":"new"}`}},
		{name: "patch redirect", command: "apply", declaration: `{labels: [{name: keep, description: ""}]}`, body: current, failMethod: "PATCH", status: 307, errorText: "307", requests: []string{"GET /repos/owner/current/labels ", `PATCH /repos/owner/current/labels/keep {"description":""}`}},
		{name: "delete redirect", command: "apply", declaration: `{labels: []}`, body: current, failMethod: "DELETE", status: 308, errorText: "308", requests: []string{"GET /repos/owner/current/labels ", "DELETE /repos/owner/current/labels/keep "}},
		{name: "get redirect on apply", command: "apply", declaration: `{labels: []}`, body: current, failMethod: "GET", status: 302, errorText: "302", requests: []string{"GET /repos/owner/current/labels "}},
	} {
		t.Run(test.name, func(t *testing.T) {
			var mu sync.Mutex
			var requests []string
			fixture, err := startHTTPFixture(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				body, _ := io.ReadAll(r.Body)
				requests = append(requests, r.Method+" "+r.URL.EscapedPath()+" "+string(body))
				w.Header().Set("Content-Type", "application/json")
				if r.Method == test.failMethod {
					w.Header().Set("Location", "https://api.github.com/repos/other/wrong/labels")
					w.WriteHeader(test.status)
					fmt.Fprint(w, `{"message":"fixture denied"}`)
					return
				}
				if r.Method == "GET" {
					fmt.Fprint(w, test.body)
				} else {
					w.WriteHeader(204)
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
			input := "schemaVersion: 1\nsettings: " + test.declaration + "\n"
			if err := writeSettingsFile(fixture.work, input); err != nil {
				t.Fatal(err)
			}
			result, err := executeCLI(binary, fixture.work, fixture.root, env, []string{test.command}, false)
			if err != nil {
				t.Fatal(err)
			}
			if test.errorText != "" {
				if result.runErr == nil || !strings.Contains(result.stderr.String(), test.errorText) || result.stdout.Len() != 0 {
					t.Fatalf("result=%+v", result)
				}
			} else if result.runErr != nil || result.stderr.Len() != 0 || result.stdout.String() != test.want {
				t.Fatalf("result=%+v want=%q", result, test.want)
			}
			mu.Lock()
			defer mu.Unlock()
			if !reflect.DeepEqual(requests, test.requests) {
				t.Fatalf("requests=%q want=%q", requests, test.requests)
			}
			if err := checkSettingsUnchanged(fixture.work, input); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestLabelsExportPaginationAndFailure(t *testing.T) {
	binary := acceptanceBinary(t)
	for _, fail := range []bool{false, true} {
		t.Run(fmt.Sprint(fail), func(t *testing.T) {
			var mu sync.Mutex
			var paths []string
			fixture, err := startHTTPFixture(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				paths = append(paths, r.Method+" "+r.URL.RequestURI())
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
					t.Error("export wrote")
					w.WriteHeader(400)
					return
				}
				switch r.URL.Path {
				case "/repos/owner/current":
					fmt.Fprint(w, planResponse)
				case "/repos/owner/current/topics":
					fmt.Fprint(w, `{"names":[]}`)
				case "/repos/owner/current/labels":
					if r.URL.Query().Get("page") == "1" {
						labels := make([]map[string]any, 100)
						for i := range labels {
							labels[i] = map[string]any{"name": fmt.Sprintf("label-%03d", 99-i), "color": "ABCDEF", "description": nil, "id": i, "default": true}
						}
						_ = json.NewEncoder(w).Encode(labels)
					} else if fail {
						w.WriteHeader(403)
						fmt.Fprint(w, `{"message":"denied"}`)
					} else {
						fmt.Fprint(w, `[]`)
					}
				default:
					t.Errorf("path=%s", r.URL)
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
			before := "previous configuration\n"
			if err := writeSettingsFile(fixture.work, before); err != nil {
				t.Fatal(err)
			}
			result, err := executeCLI(binary, fixture.work, fixture.root, env, []string{"export"}, false)
			if err != nil {
				t.Fatal(err)
			}
			if fail {
				if result.runErr == nil || !strings.Contains(result.stderr.String(), "403") {
					t.Fatalf("result=%+v", result)
				}
				if err := checkSettingsUnchanged(fixture.work, before); err != nil {
					t.Fatal(err)
				}
			} else {
				if result.runErr != nil || result.stderr.Len() != 0 || result.stdout.Len() != 0 {
					t.Fatalf("result=%+v", result)
				}
				data, err := os.ReadFile(filepath.Join(fixture.work, ".github", "settings.yml"))
				if err != nil {
					t.Fatal(err)
				}
				var document struct {
					Settings struct {
						Labels []map[string]any `yaml:"labels"`
					} `yaml:"settings"`
				}
				if err := yaml.Unmarshal(data, &document); err != nil {
					t.Fatal(err)
				}
				if len(document.Settings.Labels) != 100 {
					t.Fatalf("labels=%d", len(document.Settings.Labels))
				}
				for i, label := range document.Settings.Labels {
					if !reflect.DeepEqual(label, map[string]any{"name": fmt.Sprintf("label-%03d", i), "color": "abcdef", "description": ""}) {
						t.Fatalf("label=%v", label)
					}
				}
			}
			mu.Lock()
			defer mu.Unlock()
			want := []string{"GET /repos/owner/current", "POST /graphql", "POST /graphql", "POST /graphql", "POST /graphql", "GET /repos/owner/current/topics", "GET /repos/owner/current/labels?per_page=100&page=1", "GET /repos/owner/current/labels?per_page=100&page=2"}
			if !reflect.DeepEqual(paths, want) {
				t.Fatalf("paths=%v", paths)
			}
		})
	}
}

func TestLabelsActionReportAndCredentialFreeSummary(t *testing.T) {
	binary := acceptanceBinary(t)
	fixture, err := startHTTPFixture(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.Path != "/repos/owner/current/labels" {
			t.Errorf("unexpected %s %s", r.Method, r.URL)
			w.WriteHeader(400)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[{"name":"old","color":"abcdef","description":null}]`)
	}, true)
	if err != nil {
		t.Fatal(err)
	}
	defer fixture.close()
	env := exportFixtureEnv(fixture.cfg)
	if err := setupOrdinaryOrigin(fixture.work, env); err != nil {
		t.Fatal(err)
	}
	if err := writeSettingsFile(fixture.work, "schemaVersion: 1\nsettings: {labels: [{name: '<new>', description: ''}]}\n"); err != nil {
		t.Fatal(err)
	}
	report := filepath.Join(fixture.root, "plan.json")
	result, err := executeCLI(binary, fixture.work, fixture.root, env, []string{"action", "plan", report}, false)
	if err != nil || result.runErr != nil || result.stderr.Len() != 0 {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	data, err := os.ReadFile(report)
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	want := map[string]any{"outcome": "changed", "target": "owner/current", "managed": float64(1), "changes": []any{}, "labels": []any{
		map[string]any{"operation": "create", "name": "<new>", "after": map[string]any{"description": ""}},
		map[string]any{"operation": "delete", "name": "old", "before": map[string]any{"color": "abcdef", "description": ""}},
	}}
	if !reflect.DeepEqual(decoded, want) {
		t.Fatalf("report=%s", data)
	}
	summary := filepath.Join(fixture.root, "summary.md")
	summaryEnv := append(fixtureEnv(fixture.cfg), "SETTINGS_ACTOR=user", "SETTINGS_TRIGGERING_ACTOR=user", "SETTINGS_EVENT=push", "SETTINGS_RUN_ID=123", "SETTINGS_RUN_ATTEMPT=1", "SETTINGS_WORKFLOW_REF=owner/current/.github/workflows/settings.yml@refs/heads/main", "SETTINGS_WORKFLOW_SHA="+strings.Repeat("a", 40), "SETTINGS_CONFIG_SHA="+strings.Repeat("b", 40), "SETTINGS_TARGET=owner/current", "SETTINGS_CLI_VERSION=dev", "SETTINGS_CLI_SHA256="+strings.Repeat("c", 64))
	result, err = executeCLI(binary, fixture.root, fixture.root, summaryEnv, []string{"action", "summary", report, summary}, false)
	if err != nil || result.runErr != nil || result.stderr.Len() != 0 {
		t.Fatalf("summary=%+v err=%v", result, err)
	}
	data, err = os.ReadFile(summary)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "<new>") || !strings.Contains(html.UnescapeString(string(data)), `settings.labels: create "<new>" description=""`) || !strings.Contains(string(data), "removes issue/PR associations") || fixture.reads.Load() != 1 {
		t.Fatalf("summary=%s requests=%d", data, fixture.reads.Load())
	}
}
