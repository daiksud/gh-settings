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
	"strings"
	"sync"
	"testing"
)

func TestBuiltCompositeApplyProgressDiagnostic(t *testing.T) {
	product, version := suppliedCompositeProduct(t)
	for _, scenario := range []string{"initial-read-failure", "confirmed-removal-before-failure"} {
		t.Run(scenario, func(t *testing.T) {
			f := newCompositeFixture(t)
			f.digest = fmt.Sprintf("%x", sha256.Sum256(product))
			f.write(f.base["MOCK_ASSET"], string(product))
			f.write(f.base["MOCK_CHECKSUMS"], f.digest+"  gh-settings-linux-amd64\n")
			f.write(filepath.Join(f.packagePath, "action/cli-version"), version+"\n")
			f.write(filepath.Join(f.caller, ".github/settings.yml"), "version: daiksud/gh-settings@v0\nrepository: {settings: {interaction_limits: {pull_request_limits: {exemption_list: [alice]}}}}\n")
			dir, err := os.MkdirTemp("/tmp", "ghs-progress-")
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = os.RemoveAll(dir) })
			listener, err := net.Listen("unix", filepath.Join(dir, "api.sock"))
			if err != nil {
				t.Fatal(err)
			}
			var mu sync.Mutex
			members := true
			var methods []string
			server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				w.Header().Set("Content-Type", "application/json")
				if r.Host != "api.github.com" || r.Header.Get("Authorization") != "token progress-fixture" {
					t.Error("unexpected target or credential")
					w.WriteHeader(400)
					return
				}
				if r.URL.Path == "/users/alice" && r.Method == "GET" {
					fmt.Fprint(w, `{"login":"alice","id":15,"type":"User"}`)
					return
				}
				if r.URL.Path != "/repos/owner/config/interaction-limits/pulls/bypass-list" {
					t.Error("unexpected route", r.Method, r.URL.Path)
					w.WriteHeader(400)
					return
				}
				methods = append(methods, r.Method)
				switch r.Method {
				case "GET":
					if scenario == "initial-read-failure" {
						w.WriteHeader(403)
						fmt.Fprint(w, `{}`)
						return
					}
					users := []map[string]any{}
					if members {
						users = append(users, map[string]any{"id": 2, "login": "bob", "type": "User"})
					}
					json.NewEncoder(w).Encode(users)
				case "DELETE":
					members = false
					w.WriteHeader(204)
				case "PUT":
					w.WriteHeader(422)
					fmt.Fprint(w, `{}`)
				default:
					t.Error("unexpected method", r.Method)
					w.WriteHeader(400)
				}
			})}
			go func() { _ = server.Serve(listener) }()
			t.Cleanup(func() { _ = server.Close() })
			f.write(filepath.Join(dir, "config.yml"), "http_unix_socket: "+filepath.Join(dir, "api.sock")+"\n")
			f.base["GH_CONFIG_DIR"], f.base["GH_PATH_DISABLED"] = dir, "1"
			for _, key := range []string{"HTTP_PROXY", "HTTPS_PROXY", "http_proxy", "https_proxy"} {
				f.base[key] = "http://127.0.0.1:1"
			}
			f.base["NO_PROXY"], f.base["no_proxy"] = "", ""
			result := f.run(t, "apply", "progress-fixture", map[string]string{"github.action_ref": version})
			if result.status == 0 || len(result.outputs) != 0 {
				t.Fatal(result)
			}
			gotProgress := strings.Contains(result.log, "earlier successful writes remain applied")
			if gotProgress != (scenario == "confirmed-removal-before-failure") || !strings.Contains(result.log, "repository.settings.interaction_limits.pull_request_limits.exemption_list") {
				t.Fatal(result)
			}
			if strings.Contains(result.log, "write effects may already have applied") != (scenario == "confirmed-removal-before-failure") {
				t.Fatal("failed-write uncertainty was lost or invented", result)
			}
			if strings.Contains(result.log, "write effects are unconfirmed") != (scenario == "confirmed-removal-before-failure") {
				t.Fatal("failed operation was not classified correctly", result)
			}
			mu.Lock()
			defer mu.Unlock()
			if scenario == "initial-read-failure" && (len(methods) != 1 || methods[0] != "GET" || !members) {
				t.Fatal(methods, members)
			}
			if scenario == "confirmed-removal-before-failure" && (members || strings.Join(methods, ",") != "GET,DELETE,GET,PUT") {
				t.Fatal(methods, members)
			}
		})
	}
}
