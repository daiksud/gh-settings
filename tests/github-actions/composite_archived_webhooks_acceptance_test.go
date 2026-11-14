//go:build acceptance

package githubactions

import (
	"crypto/sha256"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestBuiltCompositeArchivedWebhooks(t *testing.T) {
	product, version := suppliedCompositeProduct(t)
	for _, mode := range []string{"matching", "config differs", "created drift", "missing config"} {
		t.Run(mode, func(t *testing.T) {
			f := newCompositeFixture(t)
			f.base["GITHUB_ACTIONS"] = "true"
			f.digest = fmt.Sprintf("%x", sha256.Sum256(product))
			f.write(f.base["MOCK_ASSET"], string(product))
			f.write(f.base["MOCK_CHECKSUMS"], f.digest+"  gh-settings-linux-amd64\n")
			f.write(filepath.Join(f.packagePath, "action/cli-version"), version+"\n")
			input := "version: daiksud/gh-settings@v0\nrepository: {settings: {general: {danger_zone: {archive_this_repository: true}}, webhooks: [{id: 7, content_type: application/json, active: true, events: [push, issues]}]}}\n"
			path := filepath.Join(f.caller, ".github/settings.yml")
			f.write(path, input)
			dir, err := os.MkdirTemp("/tmp", "ghs-archive-webhooks-")
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = os.RemoveAll(dir) })
			listener, err := net.Listen("unix", filepath.Join(dir, "api.sock"))
			if err != nil {
				t.Fatal(err)
			}
			var mu sync.Mutex
			hookReads, writes := 0, 0
			server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				if r.Host != "api.github.com" || r.Header.Get("Authorization") != "token archive-fixture" {
					t.Error("unexpected target or credential")
					w.WriteHeader(400)
					return
				}
				if r.Method != "GET" {
					writes++
					w.WriteHeader(400)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				switch r.URL.Path {
				case "/repos/owner/config":
					fmt.Fprint(w, `{"id":1,"full_name":"owner/config","archived":true,"private":false,"visibility":"public","owner":{"id":2,"login":"owner","type":"User"}}`)
				case "/repos/owner/config/hooks/7":
					hookReads++
					created := "2026-11-14T00:00:00Z"
					if mode == "created drift" && hookReads == 2 {
						created = "2026-11-14T01:00:00Z"
					}
					fmt.Fprintf(w, `{"id":7,"name":"web","type":"Repository","url":"https://api.github.com/repos/owner/config/hooks/7","created_at":%q,"active":true,"events":["issues","push"],"config":{"url":"https://example.com/hook","content_type":"json","insecure_ssl":"0","secret":"SECRET_CANARY"}}`, created)
				case "/repos/owner/config/hooks/7/config":
					content := "json"
					if mode == "config differs" && hookReads == 2 {
						content = "form"
					}
					if mode == "missing config" && hookReads == 2 {
						fmt.Fprint(w, `{}`)
						return
					}
					fmt.Fprintf(w, `{"url":"https://example.com/hook","content_type":%q,"insecure_ssl":"0","secret":"SECRET_CANARY"}`, content)
				default:
					t.Error("unexpected route", r.URL.Path)
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
			result := f.run(t, "apply", "archive-fixture", map[string]string{"github.action_ref": version})
			if (result.status == 0) != (mode == "matching") || (result.status != 0 && len(result.outputs) != 0) {
				t.Fatalf("unexpected result: %+v", result)
			}
			if strings.Contains(fmt.Sprintf("%+v", result), "SECRET_CANARY") {
				t.Fatal("secret leaked")
			}
			mu.Lock()
			defer mu.Unlock()
			data, err := os.ReadFile(path)
			if err != nil || string(data) != input || f.git("rev-parse", "HEAD") != f.sha || writes != 0 || hookReads != 2 {
				t.Fatalf("declaration/writes/reads: %v %d %d", err, writes, hookReads)
			}
		})
	}
}
