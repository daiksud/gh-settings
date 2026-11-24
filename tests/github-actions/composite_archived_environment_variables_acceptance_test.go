//go:build acceptance

package githubactions

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestBuiltCompositeArchivedEnvironmentVariables(t *testing.T) {
	const desired = "FIXTURE_ENV_VAR_DESIRED_CANARY"
	const other = "FIXTURE_ENV_VAR_OTHER_CANARY"
	product := suppliedCompositeProduct(t)
	for _, scenario := range []string{"matching", "different", "denied", "parent drift"} {
		t.Run(scenario, func(t *testing.T) {
			f := product.newFixture(t)
			f.base["GITHUB_ACTIONS"] = "true"

			input := "version: daiksud/gh-settings@v0\nrepository: {settings: {general: {danger_zone: {archive_this_repository: true}}, environments: [{name: production, variables: [{name: CHANNEL, value: " + desired + "}]}]}}\n"
			config := filepath.Join(f.caller, ".github/settings.yml")
			f.write(config, input)
			dir, err := os.MkdirTemp("/tmp", "ghs-archived-env-vars-")
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = os.RemoveAll(dir) })
			listener, err := net.Listen("unix", filepath.Join(dir, "api.sock"))
			if err != nil {
				t.Fatal(err)
			}
			var mu sync.Mutex
			counts := map[string]int{}
			writes := 0
			parent := "/repos/owner/config/environments/production"
			server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				w.Header().Set("Content-Type", "application/json")
				if r.Host != "api.github.com" || r.Header.Get("Authorization") != "token archive-env-vars-fixture" {
					t.Error("unexpected target or credential")
					w.WriteHeader(400)
					return
				}
				if r.Method != http.MethodGet {
					writes++
					w.WriteHeader(400)
					return
				}
				counts[r.URL.Path]++
				switch r.URL.Path {
				case "/repos/owner/config":
					fmt.Fprint(w, `{"id":71,"full_name":"owner/config","archived":true}`)
				case parent:
					id := 7
					if scenario == "parent drift" && counts[parent] > 1 {
						id = 8
					}
					fmt.Fprintf(w, `{"id":%d,"name":"production"}`, id)
				case parent + "/variables/CHANNEL":
					if scenario == "denied" {
						w.WriteHeader(403)
						fmt.Fprintf(w, `{"message":%q}`, other)
						return
					}
					value := desired
					if scenario == "different" {
						value = other
					}
					fmt.Fprintf(w, `{"name":"CHANNEL","value":%q}`, value)
				default:
					t.Error("unexpected route", r.URL.Path)
					w.WriteHeader(400)
				}
			})}
			go func() { _ = server.Serve(listener) }()
			t.Cleanup(func() { _ = server.Close() })
			f.write(filepath.Join(dir, "config.yml"), "http_unix_socket: "+filepath.Join(dir, "api.sock")+"\n")
			f.base["GH_CONFIG_DIR"], f.base["GH_PATH_DISABLED"], f.base["GH_DEBUG"] = dir, "1", "api"
			for _, key := range []string{"HTTP_PROXY", "HTTPS_PROXY", "http_proxy", "https_proxy"} {
				f.base[key] = "http://127.0.0.1:1"
			}
			f.base["NO_PROXY"], f.base["no_proxy"] = "", ""
			result := f.run(t, "apply", "archive-env-vars-fixture", nil)
			if (result.status == 0) != (scenario == "matching") || (result.status != 0 && len(result.outputs) != 0) {
				t.Fatal("unexpected Action outcome", result.status, strings.ReplaceAll(strings.ReplaceAll(result.log, desired, "[redacted]"), other, "[redacted]"))
			}
			for _, output := range []string{result.log, result.summary, result.api, result.calls, fmt.Sprint(result.outputs)} {
				if strings.Contains(output, desired) || strings.Contains(output, other) {
					t.Fatal("variable value leaked into Action output")
				}
			}
			data, err := os.ReadFile(config)
			if err != nil || string(data) != input || f.git("rev-parse", "HEAD") != f.sha {
				t.Fatal("configuration or commit changed", err)
			}
			mu.Lock()
			defer mu.Unlock()
			if writes != 0 {
				t.Fatal("unexpected writes", writes)
			}
			if scenario == "parent drift" && counts[parent] != 2 {
				t.Fatal("parent drift not exercised")
			}
			if scenario == "matching" && (counts["/repos/owner/config"] != 2 || counts[parent] != 2 || counts[parent+"/variables/CHANNEL"] != 1) {
				t.Fatal("missing read-only evidence", counts)
			}
		})
	}
}
