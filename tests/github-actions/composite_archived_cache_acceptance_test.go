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

func TestBuiltCompositeArchivedCache(t *testing.T) {
	product, version := suppliedCompositeProduct(t)
	for _, scenario := range []string{"matching", "retention differs", "storage differs", "duplicate retention", "alias retention", "alias storage", "alias metadata", "denied"} {
		t.Run(scenario, func(t *testing.T) {
			f := newCompositeFixture(t)
			f.base["GITHUB_ACTIONS"] = "true"
			f.digest = fmt.Sprintf("%x", sha256.Sum256(product))
			f.write(f.base["MOCK_ASSET"], string(product))
			f.write(f.base["MOCK_CHECKSUMS"], f.digest+"  gh-settings-linux-amd64\n")
			f.write(filepath.Join(f.packagePath, "action/cli-version"), version+"\n")
			input := "version: daiksud/gh-settings@v0\nrepository: {settings: {general: {danger_zone: {archive_this_repository: true}}, actions: {general: {cache: {cache_retention: 30, cache_size_eviction_limit: 10}}}}}\n"
			path := filepath.Join(f.caller, ".github/settings.yml")
			f.write(path, input)
			dir, err := os.MkdirTemp("/tmp", "ghs-archive-cache-")
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = os.RemoveAll(dir) })
			listener, err := net.Listen("unix", filepath.Join(dir, "api.sock"))
			if err != nil {
				t.Fatal(err)
			}
			var mu sync.Mutex
			writes, repoReads, cacheReads := 0, 0, 0
			server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				w.Header().Set("Content-Type", "application/json")
				if r.Host != "api.github.com" || r.Header.Get("Authorization") != "token archive-cache-fixture" {
					t.Error("unexpected target or credential")
					w.WriteHeader(400)
					return
				}
				if r.Method != http.MethodGet {
					writes++
					w.WriteHeader(400)
					return
				}
				switch r.URL.Path {
				case "/repos/owner/config":
					repoReads++
					if scenario == "alias metadata" && repoReads == 2 {
						fmt.Fprint(w, `{"id":42,"full_name":"owner/config","visibility":"public","VISIBILITY":"private","archived":true}`)
					} else {
						fmt.Fprint(w, `{"id":42,"full_name":"owner/config","visibility":"public","archived":true}`)
					}
				case "/repos/owner/config/actions/cache/retention-limit":
					cacheReads++
					if scenario == "denied" {
						w.WriteHeader(403)
						return
					}
					if scenario == "duplicate retention" {
						fmt.Fprint(w, `{"max_cache_retention_days":31,"max_cache_retention_days":30}`)
						return
					}
					if scenario == "alias retention" {
						fmt.Fprint(w, `{"MAX_CACHE_RETENTION_DAYS":31,"max_cache_retention_days":30}`)
						return
					}
					v := 30
					if scenario == "retention differs" {
						v = 31
					}
					fmt.Fprintf(w, `{"max_cache_retention_days":%d}`, v)
				case "/repos/owner/config/actions/cache/storage-limit":
					cacheReads++
					if scenario == "alias storage" {
						fmt.Fprint(w, `{"MAX_CACHE_SIZE_GB":11,"max_cache_size_gb":10}`)
						return
					}
					v := 10
					if scenario == "storage differs" {
						v = 11
					}
					fmt.Fprintf(w, `{"max_cache_size_gb":%d}`, v)
				default:
					t.Errorf("unexpected path %s", r.URL.Path)
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
			result := f.run(t, "apply", "archive-cache-fixture", map[string]string{"github.action_ref": version})
			if (result.status == 0) != (scenario == "matching") || (result.status != 0 && len(result.outputs) != 0) {
				t.Fatalf("unexpected result %+v", result)
			}
			mu.Lock()
			defer mu.Unlock()
			data, err := os.ReadFile(path)
			if err != nil || string(data) != input || writes != 0 {
				t.Fatalf("config error=%v writes=%d", err, writes)
			}
			if scenario == "alias metadata" && (repoReads != 2 || cacheReads != 0) || scenario == "alias retention" && cacheReads != 1 || scenario == "alias storage" && cacheReads != 2 {
				t.Fatalf("alias endpoint not reached: repository=%d cache=%d", repoReads, cacheReads)
			}
			if scenario == "matching" && (repoReads != 3 || cacheReads != 2) {
				t.Fatalf("reads=%d %d", repoReads, cacheReads)
			}
		})
	}
}
