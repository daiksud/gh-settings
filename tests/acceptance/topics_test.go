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
)

func TestTopicsPlanApplyAcceptance(t *testing.T) {
	binary := acceptanceBinary(t)
	for _, test := range []struct {
		name, command, input, body, want, errorText string
		status                                      int
	}{
		{name: "sorted plan", command: "plan", input: "names: [z, a]", body: `{"names":["old"]}`, want: "settings.topics.names: [\"old\"] → [\"a\",\"z\"]\n"},
		{name: "order independent", command: "plan", input: "names: [z, a]", body: `{"names":["a","z"]}`, want: "No differences for managed repository settings.\n"},
		{name: "clear plan", command: "plan", input: "names: []", body: `{"names":["old"]}`, want: "settings.topics.names: [\"old\"] → []\n"},
		{name: "exact sorted PUT", command: "apply", input: "names: [z, a]", want: `{"names":["a","z"]}`},
		{name: "exact empty PUT", command: "apply", input: "names: []", want: `{"names":[]}`},
		{name: "forbidden", command: "apply", input: "names: []", status: 403, errorText: "403"},
		{name: "redirect rejected", command: "apply", input: "names: []", status: 302, errorText: "302"},
		{name: "null rejected", command: "apply", input: "names: null", errorText: "array of strings"},
		{name: "invalid entry", command: "plan", input: "names: [true]", errorText: "array of strings"},
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
			}, false)
			if err != nil {
				t.Fatal(err)
			}
			defer fixture.close()
			env := exportFixtureEnv(fixture.cfg)
			if err := setupOrdinaryOrigin(fixture.work, env); err != nil {
				t.Fatal(err)
			}
			input := "schemaVersion: 1\nsettings:\n  topics: {" + test.input + "}\n"
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
			} else {
				want := test.want
				if test.command == "apply" && test.input != "" {
					want = "Applied repository settings.\n"
				}
				if result.runErr != nil || result.stdout.String() != want || result.stderr.Len() != 0 {
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
				return
			}
			if result.runErr != nil || result.stdout.Len() != 0 || result.stderr.Len() != 0 {
				t.Fatalf("result=%+v", result)
			}
			data, err := os.ReadFile(filepath.Join(fixture.work, ".github", "settings.yml"))
			if err != nil || !strings.Contains(string(data), "names:\n            - a\n            - z") {
				t.Fatalf("export=%s error=%v", data, err)
			}
			input := "schemaVersion: 1\nsettings: {topics: {names: []}}\n"
			if err := writeSettingsFile(fixture.work, input); err != nil {
				t.Fatal(err)
			}
			report := filepath.Join(fixture.root, "topics.json")
			result, err = executeCLI(binary, fixture.work, fixture.root, env, []string{"action", "plan", report}, false)
			if err != nil || result.runErr != nil || result.stderr.Len() != 0 {
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
			if !reflect.DeepEqual(paths, []string{"GET /repos/owner/current", "POST /graphql", "POST /graphql", "POST /graphql", "POST /graphql", "GET /repos/owner/current/topics", "GET /repos/owner/current/labels", "GET /repos/owner/current/topics"}) {
				t.Fatalf("requests=%v", paths)
			}
		})
	}
}
