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

func TestBuiltCompositeRunnerLabels(t *testing.T) {
	product, version := suppliedCompositeProduct(t)
	for _, scenario := range []string{"matching", "change", "clear", "readback failure", "export"} {
		t.Run(scenario, func(t *testing.T) {
			f := newCompositeFixture(t)
			f.base["GH_DEBUG"] = "api"
			f.digest = fmt.Sprintf("%x", sha256.Sum256(product))
			f.write(f.base["MOCK_ASSET"], string(product))
			f.write(f.base["MOCK_CHECKSUMS"], f.digest+"  gh-settings-linux-amd64\n")
			f.write(filepath.Join(f.packagePath, "action/cli-version"), version+"\n")
			desired := []string{"private-custom-canary"}
			if scenario == "clear" {
				desired = []string{}
			}
			encoded, _ := json.Marshal(desired)
			input := "version: daiksud/gh-settings@v0\nrepository: {settings: {actions: {runners: [{id: 23, custom_labels: " + string(encoded) + "}]}}}\n"
			path := filepath.Join(f.caller, ".github/settings.yml")
			f.write(path, input)
			dir, e := os.MkdirTemp("/tmp", "ghs-runners-")
			if e != nil {
				t.Fatal(e)
			}
			t.Cleanup(func() { os.RemoveAll(dir) })
			listener, e := net.Listen("unix", filepath.Join(dir, "api.sock"))
			if e != nil {
				t.Fatal(e)
			}
			var mu sync.Mutex
			labels := []string{"old"}
			if scenario == "matching" {
				labels = []string{"PRIVATE-CUSTOM-CANARY"}
			}
			writes := 0
			rawLabels := func() []any {
				v := []any{map[string]any{"id": 1, "name": "private-readonly", "type": "read-only"}}
				for _, n := range labels {
					v = append(v, map[string]any{"name": n, "type": "custom"})
				}
				return v
			}
			server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				w.Header().Set("Content-Type", "application/json")
				if r.Host != "api.github.com" || r.Header.Get("Authorization") != "token runners-fixture" {
					t.Error("credential/target escaped")
					w.WriteHeader(400)
					return
				}
				switch r.URL.Path {
				case "/repos/owner/config":
					fmt.Fprint(w, `{"id":1,"full_name":"owner/config","private":false,"archived":false,"owner":{"id":2,"login":"owner","type":"User"}}`)
				case "/repos/owner/config/actions/runners":
					if scenario != "export" {
						t.Error("unselected inventory read")
					}
					json.NewEncoder(w).Encode(map[string]any{"total_count": 1, "runners": []any{map[string]any{"id": 23, "name": "private-runner", "os": "private-os", "status": "online", "busy": false, "labels": rawLabels()}}})
				case "/repos/owner/config/actions/runners/23":
					if scenario == "readback failure" && writes > 0 {
						w.WriteHeader(403)
						fmt.Fprint(w, `{"message":"private-upstream"}`)
						return
					}
					json.NewEncoder(w).Encode(map[string]any{"id": 23, "name": "private-runner", "os": "private-os", "status": "offline", "busy": true, "labels": rawLabels()})
				case "/repos/owner/config/actions/runners/23/labels":
					if r.Method == "PUT" {
						var p map[string][]string
						if json.NewDecoder(r.Body).Decode(&p) != nil || len(p) != 1 || !reflect.DeepEqual(p["labels"], desired) {
							t.Error("wrong replacement", p)
						}
						labels = append([]string{}, desired...)
						writes++
					} else if r.Method != "GET" {
						t.Error("forbidden method")
						w.WriteHeader(400)
						return
					}
					v := rawLabels()
					json.NewEncoder(w).Encode(map[string]any{"total_count": len(v), "labels": v})
				default:
					if scenario != "export" {
						t.Error("unexpected endpoint", r.URL)
					}
					w.WriteHeader(403)
					fmt.Fprint(w, `{"message":"unavailable"}`)
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
			opts := map[string]string{"github.action_ref": version}
			if scenario == "export" {
				os.Remove(path)
				result := f.run(t, "export", "runners-fixture", opts)
				b, _ := os.ReadFile(path)
				if result.status != 0 || result.outputs["config-path"] != ".github/settings.yml" || !strings.Contains(string(b), "custom_labels:") || strings.Contains(string(b)+result.log+result.summary, "private-") {
					t.Fatal(result, string(b))
				}
				mu.Lock()
				defer mu.Unlock()
				if writes != 0 {
					t.Fatal(writes)
				}
				return
			}
			detail := f.run(t, "plan", "runners-fixture", opts)
			if detail.status != 0 || (scenario != "matching" && (!strings.Contains(detail.summary, "custom_labels") || !strings.Contains(detail.summary, "23"))) {
				t.Fatal(detail)
			}
			private := f.run(t, "plan", "runners-fixture", map[string]string{"github.action_ref": version, "inputs.plan-reporting": "status-only"})
			outcome := "changed"
			if scenario == "matching" {
				outcome = "no-difference"
			}
			wantOutputs := map[string]string{"config-sha": f.sha, "target": "owner/config", "cli-version": version, "cli-sha256": f.digest, "plan-outcome": outcome}
			if private.status != 0 || !reflect.DeepEqual(private.outputs, wantOutputs) || strings.Contains(private.log+private.summary, "custom_labels") || strings.Contains(strings.ToLower(private.log+private.summary), "private-") || len(private.remainingReports) != 0 {
				t.Fatal(private)
			}
			failed := scenario == "readback failure"
			n := 2
			if failed {
				n = 1
			}
			for i := 0; i < n; i++ {
				r := f.run(t, "apply", "runners-fixture", opts)
				if (r.status != 0) != failed || (!failed && len(r.outputs) != 4) || (failed && len(r.outputs) != 0) || strings.Contains(r.log, "private-") || strings.Contains(r.log, "/actions/runners") {
					t.Fatal(r)
				}
			}
			b, _ := os.ReadFile(path)
			if string(b) != input || f.git("rev-parse", "HEAD") != f.sha {
				t.Fatal("input changed")
			}
			mu.Lock()
			defer mu.Unlock()
			want := 1
			if scenario == "matching" {
				want = 0
			}
			if writes != want {
				t.Fatal("repeat wrote", writes)
			}
		})
	}
}
