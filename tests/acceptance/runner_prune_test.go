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

func TestRunnerPruneBuiltCLI(t *testing.T) {
	binary := acceptanceBinary(t)
	for _, selector := range []string{"runners", "all"} {
		t.Run(selector, func(t *testing.T) {
			var mu sync.Mutex
			labels := []string{"old"}
			present := true
			events := []string{}
			unconfirmed := false
			rawLabels := func(id int) []any {
				out := []any{map[string]any{"id": 1, "name": "self-hosted", "type": "read-only"}}
				if id == 23 {
					for _, n := range labels {
						out = append(out, map[string]any{"name": n, "type": "custom"})
					}
				} else {
					out = append(out, map[string]any{"name": "candidate-label", "type": "custom"})
				}
				return out
			}
			row := func(id int) map[string]any {
				return map[string]any{"id": id, "name": fmt.Sprintf("runner-%d-canary", id), "os": "os-canary", "status": "offline", "busy": false, "ephemeral": false, "version": "1", "labels": rawLabels(id)}
			}
			f, e := startHTTPFixture(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				w.Header().Set("Content-Type", "application/json")
				switch r.URL.Path {
				case "/repos/owner/current":
					if r.Method != "GET" {
						t.Error("unexpected target write")
						w.WriteHeader(403)
						return
					}
					fmt.Fprint(w, `{"id":1,"full_name":"owner/current","private":false,"archived":false,"owner":{"id":2,"login":"owner","type":"User"}}`)
				case "/repos/owner/current/actions/runners":
					if r.Method != "GET" {
						t.Error("unexpected list mutation")
						w.WriteHeader(403)
						return
					}
					rows := []any{row(23)}
					if present {
						rows = append(rows, row(24))
					}
					json.NewEncoder(w).Encode(map[string]any{"total_count": len(rows), "runners": rows})
				case "/repos/owner/current/actions/runners/23", "/repos/owner/current/actions/runners/24":
					id := 23
					if strings.HasSuffix(r.URL.Path, "/24") {
						id = 24
					}
					if r.Method == "DELETE" && id == 24 && present {
						events = append(events, "delete")
						if !unconfirmed {
							present = false
						}
						w.WriteHeader(204)
						return
					}
					if r.Method != "GET" || (id == 24 && !present) {
						t.Error("unexpected runner request", r.Method, r.URL)
						w.WriteHeader(403)
						return
					}
					json.NewEncoder(w).Encode(row(id))
				case "/repos/owner/current/actions/runners/23/labels", "/repos/owner/current/actions/runners/24/labels":
					id := 23
					if strings.Contains(r.URL.Path, "/24/") {
						id = 24
					}
					if r.Method == "PUT" && id == 23 {
						var body map[string][]string
						if json.NewDecoder(r.Body).Decode(&body) != nil || !reflect.DeepEqual(body, map[string][]string{"labels": {"new"}}) {
							t.Error("unexpected payload", body)
						}
						labels = []string{"new"}
						events = append(events, "labels")
					} else if r.Method != "GET" {
						t.Error("unexpected label mutation", r.Method)
						w.WriteHeader(403)
						return
					}
					v := rawLabels(id)
					json.NewEncoder(w).Encode(map[string]any{"total_count": len(v), "labels": v})
				default:
					t.Error("unexpected route", r.Method, r.URL)
					w.WriteHeader(403)
				}
			}, false, "/actions/runners", "/actions/runners/23", "/actions/runners/24", "/actions/runners/23/labels", "/actions/runners/24/labels")
			if e != nil {
				t.Fatal(e)
			}
			defer f.close()
			env := append(exportFixtureEnv(f.cfg), "GITHUB_ACTIONS=true")
			if e = setupOrdinaryOrigin(f.work, env); e != nil {
				t.Fatal(e)
			}
			path := filepath.Join(f.work, ".github/settings.yml")
			if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
				t.Fatal(err)
			}
			save := func(body string) {
				t.Helper()
				if e := os.WriteFile(path, []byte("version: daiksud/gh-settings@v0\nrepository: {settings: {actions: {"+body+"}}}\n"), 0600); e != nil {
					t.Fatal(e)
				}
			}
			run := func(args ...string) *executionResult {
				t.Helper()
				v, e := executeCLI(binary, f.work, f.root, env, args, false)
				if e != nil {
					t.Fatal(e)
				}
				return v
			}
			save("runners: []")
			r := run("apply")
			if r.runErr != nil || len(events) != 0 {
				t.Fatal(r.runErr, events)
			}
			save("runners: [{id: 23, custom_labels: [new]}]")
			r = run("plan", "--prune", selector)
			if r.runErr != nil || !strings.Contains(r.stdout.String(), "runner-24-canary") || !strings.Contains(r.stdout.String(), "candidate-label") {
				t.Fatal(r.runErr, r.stdout.String(), r.stderr.String())
			}
			mu.Lock()
			unconfirmed = true
			mu.Unlock()
			r = run("apply", "--prune", selector)
			if r.runErr == nil || !strings.Contains(r.stderr.String(), "effects unknown") || strings.Contains(r.stderr.String(), "canary") {
				t.Fatal(r.runErr, r.stderr.String())
			}
			mu.Lock()
			if !reflect.DeepEqual(events, []string{"labels", "delete"}) || !present {
				t.Fatal(events, present)
			}
			events = nil
			unconfirmed = false
			mu.Unlock()
			r = run("apply", "--prune", selector)
			if r.runErr != nil {
				t.Fatal(r.runErr, r.stderr.String())
			}
			mu.Lock()
			defer mu.Unlock()
			if !reflect.DeepEqual(events, []string{"delete"}) || present {
				t.Fatal(events, present)
			}
		})
	}
}
