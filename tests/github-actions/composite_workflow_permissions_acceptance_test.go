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

func TestBuiltCompositeWorkflowPermissions(t *testing.T) {
	product, version := suppliedCompositeProduct(t)
	for _, scenario := range []string{"matching", "changed", "matching declaration drift", "denied readback"} {
		t.Run(scenario, func(t *testing.T) {
			f := newCompositeFixture(t)
			f.digest = fmt.Sprintf("%x", sha256.Sum256(product))
			f.write(f.base["MOCK_ASSET"], string(product))
			f.write(f.base["MOCK_CHECKSUMS"], f.digest+"  gh-settings-linux-amd64\n")
			f.write(filepath.Join(f.packagePath, "action/cli-version"), version+"\n")
			const input = "version: daiksud/gh-settings@v0\nrepository:\n  settings:\n    actions:\n      general:\n        workflow_permissions:\n          default_permissions: read_repository_contents_and_packages_permissions\n          allow_github_actions_to_create_and_approve_pull_requests: true\n"
			path := filepath.Join(f.caller, ".github/settings.yml")
			f.write(path, input)
			state, err := os.MkdirTemp("/tmp", "ghs-workflow-")
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = os.RemoveAll(state) })
			listener, err := net.Listen("unix", filepath.Join(state, "api.sock"))
			if err != nil {
				t.Fatal(err)
			}
			var mu sync.Mutex
			permission, approval := "read", scenario == "matching"
			var operations []string
			var payloads []map[string]any
			server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				if r.Host != "api.github.com" || r.URL.Path != "/repos/owner/config/actions/permissions/workflow" || r.Header.Get("Authorization") != "token workflow-fixture-write" {
					t.Error("unexpected target or credential boundary", r.Method, r.URL.Path)
					w.WriteHeader(400)
					return
				}
				operations = append(operations, r.Method)
				w.Header().Set("Content-Type", "application/json")
				switch r.Method {
				case "GET":
					if scenario == "denied readback" && len(payloads) > 0 {
						w.WriteHeader(403)
						fmt.Fprint(w, `{"message":"denied"}`)
						return
					}
					json.NewEncoder(w).Encode(map[string]any{"default_workflow_permissions": permission, "can_approve_pull_request_reviews": approval})
				case "PUT":
					var body map[string]any
					if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
						t.Error(err)
					}
					payloads = append(payloads, body)
					approval = true
					if scenario == "matching declaration drift" {
						permission = "write"
					}
					w.WriteHeader(204)
				default:
					t.Error("unexpected method", r.Method)
					w.WriteHeader(400)
				}
			})}
			go func() { _ = server.Serve(listener) }()
			t.Cleanup(func() { _ = server.Close() })
			f.write(filepath.Join(state, "config.yml"), "http_unix_socket: "+filepath.Join(state, "api.sock")+"\n")
			f.base["GH_CONFIG_DIR"], f.base["GH_PATH_DISABLED"] = state, "1"
			for _, key := range []string{"HTTP_PROXY", "HTTPS_PROXY", "http_proxy", "https_proxy"} {
				f.base[key] = "http://127.0.0.1:1"
			}
			f.base["NO_PROXY"], f.base["no_proxy"] = "", ""
			success := scenario == "matching" || scenario == "changed"
			attempts := 1
			if success {
				attempts = 2
			}
			for i := 0; i < attempts; i++ {
				result := f.run(t, "apply", "workflow-fixture-write", map[string]string{"github.action_ref": version})
				if (result.status == 0) != success || success && len(result.outputs) != 4 || !success && len(result.outputs) != 0 {
					t.Fatalf("incorrect Action success publication: %+v", result)
				}
				if !success && !strings.Contains(result.log, "may already have applied") {
					t.Fatal("write uncertainty lost", result.log)
				}
				if strings.Contains(result.log, "earlier successful writes remain applied") {
					t.Fatal("fabricated prior progress", result.log)
				}
			}
			mu.Lock()
			defer mu.Unlock()
			want := []string{"GET", "PUT", "GET"}
			if scenario == "matching" {
				want = []string{"GET", "GET"}
				if len(payloads) != 0 {
					t.Fatal(payloads)
				}
			} else {
				if success {
					want = append(want, "GET")
				}
				if !reflect.DeepEqual(payloads, []map[string]any{{"can_approve_pull_request_reviews": true}}) {
					t.Fatal("unchanged declared field replayed", payloads)
				}
			}
			if !reflect.DeepEqual(operations, want) {
				t.Fatal(operations, want)
			}
			data, err := os.ReadFile(path)
			if err != nil || string(data) != input || f.git("rev-parse", "HEAD") != f.sha {
				t.Fatal("Action modified configuration or commit", err)
			}
		})
	}
}
