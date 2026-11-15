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

func TestBuiltCompositeArchivedEnvironmentSelectedVariables(t *testing.T) {
	const desired = "COMPOSITE_ENV_DESIRED_CANARY"
	const remote = "COMPOSITE_ENV_REMOTE_CANARY"
	const base = "/repos/owner/config/environments/production"
	product, version := suppliedCompositeProduct(t)
	for _, scenario := range []string{"matching", "different", "denied", "parent drift", "prune"} {
		t.Run(scenario, func(t *testing.T) {
			f := newCompositeFixture(t)
			f.base["GITHUB_ACTIONS"] = "true"
			f.digest = fmt.Sprintf("%x", sha256.Sum256(product))
			f.write(f.base["MOCK_ASSET"], string(product))
			f.write(f.base["MOCK_CHECKSUMS"], f.digest+"  gh-settings-linux-amd64\n")
			f.write(filepath.Join(f.packagePath, "action/cli-version"), version+"\n")
			input := "version: daiksud/gh-settings@v0\nrepository:\n  settings:\n    general: {danger_zone: {archive_this_repository: true}}\n    environments:\n      - name: production\n        variables: [{name: FIRST, value: " + desired + "}, {name: SECOND, value: " + desired + "}]\n"
			config := filepath.Join(f.caller, ".github/settings.yml")
			f.write(config, input)
			dir, err := os.MkdirTemp("/tmp", "ghs-archive-env-vars-")
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = os.RemoveAll(dir) })
			listener, err := net.Listen("unix", filepath.Join(dir, "api.sock"))
			if err != nil {
				t.Fatal(err)
			}
			var mu sync.Mutex
			parentReads, writes := 0, 0
			reads := map[string]int{}
			server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				w.Header().Set("Content-Type", "application/json")
				if r.Host != "api.github.com" || r.Header.Get("Authorization") != "token env-vars-fixture" {
					t.Error("unexpected target or credential")
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				if r.Method != http.MethodGet {
					writes++
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				reads[r.URL.Path]++
				switch r.URL.Path {
				case "/repos/owner/config":
					fmt.Fprint(w, `{"id":42,"full_name":"owner/config","archived":true}`)
				case base:
					parentReads++
					id := 7
					if scenario == "parent drift" && parentReads > 1 {
						id = 8
					}
					fmt.Fprintf(w, `{"id":%d,"name":"production"}`, id)
				case base + "/variables":
					fmt.Fprintf(w, `{"total_count":2,"variables":[{"name":"FIRST","value":%q},{"name":"SECOND","value":%q}]}`, desired, desired)
				case base + "/variables/FIRST", base + "/variables/SECOND":
					name := filepath.Base(r.URL.Path)
					if name == "SECOND" && scenario == "denied" {
						w.WriteHeader(http.StatusForbidden)
						fmt.Fprintf(w, `{"message":%q}`, remote)
						return
					}
					value := desired
					if name == "SECOND" && scenario == "different" {
						value = remote
					}
					fmt.Fprintf(w, `{"name":%q,"value":%q}`, name, value)
				default:
					t.Error("unexpected API route", r.URL.Path)
					w.WriteHeader(http.StatusBadRequest)
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
			extra := map[string]string{"github.action_ref": version, "GH_DEBUG": "api"}
			if scenario == "prune" {
				extra["inputs.prune"] = "environment-variables"
			}
			result := f.run(t, "apply", "env-vars-fixture", extra)
			if (result.status == 0) != (scenario == "matching") {
				t.Error("unexpected Action success status", result.status)
			}
			if scenario == "prune" && !strings.Contains(result.log, "archived repository cannot update") {
				t.Error("prune did not reach archived guard")
			}
			if scenario != "matching" && len(result.outputs) != 0 {
				t.Error("failed Action emitted outputs")
			}
			for _, output := range []string{result.log, result.summary, result.api, result.calls, fmt.Sprint(result.outputs)} {
				if strings.Contains(output, desired) || strings.Contains(output, remote) {
					t.Error("variable value leaked in Action output")
				}
			}
			data, err := os.ReadFile(config)
			if err != nil || string(data) != input {
				t.Fatal("declaration changed", err)
			}
			mu.Lock()
			defer mu.Unlock()
			if writes != 0 {
				t.Error("unexpected API write", writes)
			}
			if scenario == "matching" && (parentReads != 2 || reads[base+"/variables/FIRST"] != 1 || reads[base+"/variables/SECOND"] != 1) {
				t.Error("matching read counts", parentReads, reads)
			}
			if (scenario == "different" || scenario == "denied") && reads[base+"/variables/SECOND"] == 0 {
				t.Error("second variable not observed")
			}
			if scenario == "parent drift" && parentReads < 2 {
				t.Error("parent drift not observed")
			}
		})
	}
}
