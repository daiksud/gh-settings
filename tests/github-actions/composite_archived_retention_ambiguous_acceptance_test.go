//go:build acceptance

package githubactions

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestBuiltCompositeArchivedRetentionRejectsAmbiguous(t *testing.T) {
	product := suppliedCompositeProduct(t)
	for _, scenario := range []string{"matching", "duplicate days", "duplicate maximum", "alias"} {
		t.Run(scenario, func(t *testing.T) {
			f := product.newFixture(t)
			f.base["GITHUB_ACTIONS"] = "true"

			input := "version: daiksud/gh-settings@v0\nrepository: {settings: {general: {danger_zone: {archive_this_repository: true}}, actions: {general: {check_workflow_run_status_artifact_and_log_retention: 30}}}}\n"
			path := filepath.Join(f.caller, ".github/settings.yml")
			f.write(path, input)
			dir, err := os.MkdirTemp("/tmp", "ghs-archive-retention-")
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = os.RemoveAll(dir) })
			listener, err := net.Listen("unix", filepath.Join(dir, "api.sock"))
			if err != nil {
				t.Fatal(err)
			}
			var mu sync.Mutex
			reads, writes := 0, 0
			server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				if r.Host != "api.github.com" || r.Header.Get("Authorization") != "token archive-retention-fixture" {
					t.Error("unexpected target or token")
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
					fmt.Fprint(w, `{"id":42,"full_name":"owner/config","archived":true}`)
				case "/repos/owner/config/actions/permissions/artifact-and-log-retention":
					reads++
					body := `{"days":30,"maximum_allowed_days":90}`
					switch scenario {
					case "duplicate days":
						body = `{"days":10,"days":30,"maximum_allowed_days":90}`
					case "duplicate maximum":
						body = `{"days":30,"maximum_allowed_days":10,"maximum_allowed_days":90}`
					case "alias":
						body = `{"Days":10,"days":30,"maximum_allowed_days":90}`
					}
					fmt.Fprint(w, body)
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
			result := f.run(t, "apply", "archive-retention-fixture", nil)
			if (result.status == 0) != (scenario == "matching") || result.status != 0 && len(result.outputs) != 0 {
				t.Fatalf("unexpected result %+v", result)
			}
			mu.Lock()
			defer mu.Unlock()
			data, err := os.ReadFile(path)
			if err != nil || string(data) != input || writes != 0 || reads != 1 {
				t.Fatalf("config error=%v reads=%d writes=%d", err, reads, writes)
			}
		})
	}
}
