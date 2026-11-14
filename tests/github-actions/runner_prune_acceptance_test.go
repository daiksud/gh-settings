//go:build acceptance

package githubactions

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
)

// Exercises the supplied release-version binary through the actual composite steps.
func TestBuiltCompositeRunnerPrune(t *testing.T) {
	product, version := suppliedCompositeProduct(t)
	for _, tc := range []struct {
		name, selector string
		unconfirmed    bool
	}{{"named", "runners", false}, {"all", "all", false}, {"unconfirmed", "runners", true}} {
		t.Run(tc.name, func(t *testing.T) {
			f := newCompositeFixture(t)
			f.base["GITHUB_ACTIONS"], f.base["GH_DEBUG"] = "true", "api"
			f.digest = fmt.Sprintf("%x", sha256.Sum256(product))
			f.write(f.base["MOCK_ASSET"], string(product))
			f.write(f.base["MOCK_CHECKSUMS"], f.digest+"  gh-settings-linux-amd64\n")
			f.write(filepath.Join(f.packagePath, "action/cli-version"), version+"\n")
			input := "version: daiksud/gh-settings@v0\nrepository: {settings: {actions: {runners: [{id: 23, custom_labels: [new-label]}]}}}\n"
			path := filepath.Join(f.caller, ".github/settings.yml")
			f.write(path, input)
			dir, err := os.MkdirTemp("/tmp", "ghs-runner-prune-")
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { os.RemoveAll(dir) })
			listener, err := net.Listen("unix", filepath.Join(dir, "api.sock"))
			if err != nil {
				t.Fatal(err)
			}
			var mu sync.Mutex
			labels, present := []string{"old-label"}, true
			var writes []string
			rawLabels := func(id int) []any {
				rows := []any{map[string]any{"id": 1, "name": "readonly", "type": "read-only"}}
				custom := labels
				if id == 24 {
					custom = []string{"candidate-label"}
				}
				for _, n := range custom {
					rows = append(rows, map[string]any{"name": n, "type": "custom"})
				}
				return rows
			}
			runner := func(id int) map[string]any {
				return map[string]any{"id": id, "name": fmt.Sprintf("runner-%d-canary", id), "os": "linux-canary", "status": "online", "busy": false, "labels": rawLabels(id), "ephemeral": false, "version": "v1"}
			}
			server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				w.Header().Set("Content-Type", "application/json")
				if r.Host != "api.github.com" || r.Header.Get("Authorization") != "token prune-fixture" {
					t.Error("credential/target escaped")
					w.WriteHeader(400)
					return
				}
				switch r.URL.Path {
				case "/repos/owner/config":
					if r.Method != "GET" {
						t.Error("unexpected repository write")
						w.WriteHeader(400)
						return
					}
					fmt.Fprint(w, `{"id":1,"full_name":"owner/config","private":false,"archived":false,"owner":{"id":2,"login":"owner","type":"User"}}`)
				case "/repos/owner/config/actions/runners":
					if r.Method != "GET" || r.URL.RawQuery != "per_page=100&page=1" {
						t.Error("unexpected runner list", r.Method, r.URL.String())
						w.WriteHeader(400)
						return
					}
					rows := []any{runner(23)}
					if present {
						rows = append(rows, runner(24))
					}
					json.NewEncoder(w).Encode(map[string]any{"total_count": len(rows), "runners": rows})
				case "/repos/owner/config/actions/runners/23", "/repos/owner/config/actions/runners/24":
					id := 23
					if strings.HasSuffix(r.URL.Path, "/24") {
						id = 24
					}
					if r.Method == "DELETE" && id == 24 && present {
						writes = append(writes, "delete-24")
						if !tc.unconfirmed {
							present = false
						}
						w.WriteHeader(204)
						return
					}
					if r.Method != "GET" || (id == 24 && !present) {
						t.Error("unexpected runner operation", r.Method, r.URL.Path)
						w.WriteHeader(400)
						return
					}
					json.NewEncoder(w).Encode(runner(id))
				case "/repos/owner/config/actions/runners/23/labels", "/repos/owner/config/actions/runners/24/labels":
					id := 23
					if strings.Contains(r.URL.Path, "/24/") {
						id = 24
					}
					if r.Method == "PUT" && id == 23 {
						var body map[string][]string
						if json.NewDecoder(r.Body).Decode(&body) != nil || len(body) != 1 || !reflect.DeepEqual(body["labels"], []string{"new-label"}) {
							t.Error("unexpected label replacement", body)
						}
						labels = []string{"new-label"}
						writes = append(writes, "update-23")
					} else if r.Method != "GET" || (id == 24 && !present) {
						t.Error("unexpected labels operation", r.Method, r.URL.Path)
						w.WriteHeader(400)
						return
					}
					rows := rawLabels(id)
					json.NewEncoder(w).Encode(map[string]any{"total_count": len(rows), "labels": rows})
				default:
					t.Error("unexpected route", r.Method, r.URL.String())
					w.WriteHeader(403)
				}
			})}
			go func() { _ = server.Serve(listener) }()
			t.Cleanup(func() { server.Close() })
			f.write(filepath.Join(dir, "config.yml"), "http_unix_socket: "+filepath.Join(dir, "api.sock")+"\n")
			f.base["GH_CONFIG_DIR"], f.base["GH_PATH_DISABLED"] = dir, "1"
			for _, k := range []string{"HTTP_PROXY", "HTTPS_PROXY", "http_proxy", "https_proxy"} {
				f.base[k] = "http://127.0.0.1:1"
			}
			f.base["NO_PROXY"], f.base["no_proxy"] = "", ""
			opts := map[string]string{"github.action_ref": version, "inputs.prune": tc.selector}
			detail := f.run(t, "plan", "prune-fixture", opts)
			if detail.status != 0 || !strings.Contains(detail.summary, "runner-24-canary") || !strings.Contains(detail.summary, "candidate-label") || !strings.Contains(detail.summary, "new-label") {
				t.Fatal(detail)
			}
			private := f.run(t, "plan", "prune-fixture", map[string]string{"github.action_ref": version, "inputs.prune": tc.selector, "inputs.plan-reporting": "status-only"})
			if private.status != 0 || private.outputs["plan-outcome"] != "changed" || strings.Contains(private.log+private.summary, "canary") || strings.Contains(private.log+private.summary, "candidate-label") || len(private.remainingReports) != 0 {
				t.Fatal(private)
			}
			result := f.run(t, "apply", "prune-fixture", opts)
			if (result.status != 0) != tc.unconfirmed || (tc.unconfirmed && len(result.outputs) != 0) || (!tc.unconfirmed && len(result.outputs) != 4) || strings.Contains(result.log, "canary") || strings.Contains(result.log, "/actions/runners") {
				t.Fatal(result)
			}
			got, err := os.ReadFile(path)
			if err != nil || string(got) != input || f.git("rev-parse", "HEAD") != f.sha {
				t.Fatal("declaration changed", err)
			}
			mu.Lock()
			defer mu.Unlock()
			if !reflect.DeepEqual(writes, []string{"update-23", "delete-24"}) || present != tc.unconfirmed {
				t.Fatal("unexpected mutation order or state", writes, present)
			}
		})
	}
}
