//go:build acceptance

package githubactions

import (
	"crypto/sha256"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestBuiltCompositeArchivedCopilotToggles(t *testing.T) {
	product, version := suppliedCompositeProduct(t)
	for _, scenario := range []string{"matching", "instructions differ", "mcp differs", "fresh differs", "denied", "missing", "identity drift"} {
		t.Run(scenario, func(t *testing.T) {
			f := newCompositeFixture(t)
			f.base["GITHUB_ACTIONS"] = "true"
			f.digest = fmt.Sprintf("%x", sha256.Sum256(product))
			f.write(f.base["MOCK_ASSET"], string(product))
			f.write(f.base["MOCK_CHECKSUMS"], f.digest+"  gh-settings-linux-amd64\n")
			input := "version: daiksud/gh-settings@v0\nrepository: {settings: {general: {danger_zone: {archive_this_repository: true}}, copilot: {code_review: {use_custom_instructions: true, allow_copilot_to_use_mcp_tools_when_reviewing_pull_requests: false}}}}\n"
			config := filepath.Join(f.caller, ".github/settings.yml")
			f.write(config, input)
			dir, err := os.MkdirTemp("/tmp", "ghs-archived-copilot-")
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = os.RemoveAll(dir) })
			listener, err := net.Listen("unix", filepath.Join(dir, "api.sock"))
			if err != nil {
				t.Fatal(err)
			}
			var mu sync.Mutex
			repoReads, toggleReads, writes := 0, 0, 0
			server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				if r.Method != http.MethodGet {
					writes++
					w.WriteHeader(400)
					return
				}
				if r.Host != "api.github.com" || r.Header.Get("Authorization") != "token archive-copilot-fixture" {
					t.Error("unexpected target or credential")
					w.WriteHeader(400)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				switch r.URL.Path {
				case "/repos/owner/config":
					repoReads++
					id := 42
					if scenario == "identity drift" && repoReads == 2 {
						id = 43
					}
					fmt.Fprintf(w, `{"id":%d,"full_name":"owner/config","archived":true}`, id)
				case "/repos/owner/config/copilot/code-review/settings":
					toggleReads++
					if scenario == "denied" {
						w.WriteHeader(403)
						return
					}
					if scenario == "missing" {
						fmt.Fprint(w, `{"custom_instructions_enabled":true}`)
						return
					}
					instructions := scenario != "instructions differ"
					mcp := scenario == "mcp differs" || scenario == "fresh differs" && toggleReads == 2
					fmt.Fprintf(w, `{"custom_instructions_enabled":%t,"mcp_enabled":%t}`, instructions, mcp)
				default:
					t.Errorf("unexpected route %s", r.URL.Path)
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
			result := f.run(t, "apply", "archive-copilot-fixture", map[string]string{"github.action_ref": version})
			if (result.status == 0) != (scenario == "matching") || result.status != 0 && len(result.outputs) != 0 {
				t.Fatalf("unexpected Action result: %+v", result)
			}
			mu.Lock()
			defer mu.Unlock()
			if writes != 0 || toggleReads == 0 || scenario == "matching" && (repoReads != 2 || toggleReads != 2) {
				t.Fatalf("unexpected reads/writes: repo=%d toggles=%d writes=%d", repoReads, toggleReads, writes)
			}
			data, err := os.ReadFile(config)
			if err != nil || string(data) != input || f.git("rev-parse", "HEAD") != f.sha {
				t.Fatal("configuration or commit changed", err)
			}
		})
	}
}
