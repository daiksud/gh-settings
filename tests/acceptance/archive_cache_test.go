//go:build acceptance

package acceptance

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestArchivedCacheCLIReadOnly(t *testing.T) {
	for _, scenario := range []string{"matching", "retention differs", "storage differs", "duplicate retention", "duplicate metadata", "alias retention", "alias storage", "alias metadata", "denied", "identity drift"} {
		t.Run(scenario, func(t *testing.T) {
			var mu sync.Mutex
			writes, repoReads, cacheReads := 0, 0, 0
			fixture, err := startHTTPFixture(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				w.Header().Set("Content-Type", "application/json")
				if r.Method != http.MethodGet {
					writes++
					w.WriteHeader(400)
					return
				}
				switch r.URL.Path {
				case "/repos/owner/current":
					repoReads++
					id := 42
					if scenario == "identity drift" && repoReads == 3 {
						id = 43
					}
					if scenario == "duplicate metadata" && repoReads == 2 {
						fmt.Fprint(w, `{"id":42,"full_name":"owner/current","visibility":"private","visibility":"public","archived":true}`)
					} else if scenario == "alias metadata" && repoReads == 2 {
						fmt.Fprint(w, `{"id":42,"full_name":"owner/current","visibility":"public","VISIBILITY":"private","archived":true}`)
					} else {
						fmt.Fprintf(w, `{"id":%d,"full_name":"owner/current","visibility":"public","archived":true}`, id)
					}
				case "/repos/owner/current/actions/cache/retention-limit":
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
					value := 30
					if scenario == "retention differs" {
						value = 31
					}
					fmt.Fprintf(w, `{"max_cache_retention_days":%d}`, value)
				case "/repos/owner/current/actions/cache/storage-limit":
					cacheReads++
					if scenario == "alias storage" {
						fmt.Fprint(w, `{"MAX_CACHE_SIZE_GB":11,"max_cache_size_gb":10}`)
						return
					}
					value := 10
					if scenario == "storage differs" {
						value = 11
					}
					fmt.Fprintf(w, `{"max_cache_size_gb":%d}`, value)
				default:
					t.Errorf("unexpected path %s", r.URL.Path)
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
			input := repositoryInput("settings: {general: {danger_zone: {archive_this_repository: true}}, actions: {general: {cache: {cache_retention: 30, cache_size_eviction_limit: 10}}}}\n")
			if err := writeSettingsFile(fixture.work, input); err != nil {
				t.Fatal(err)
			}
			result, err := executeCLI(acceptanceBinary(t), fixture.work, fixture.root, env, []string{"apply"}, false)
			if err != nil || (result.runErr == nil) != (scenario == "matching") {
				t.Fatalf("err=%v result=%v stderr=%s", err, result.runErr, result.stderr.String())
			}
			mu.Lock()
			defer mu.Unlock()
			data, err := os.ReadFile(filepath.Join(fixture.work, ".github/settings.yml"))
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
