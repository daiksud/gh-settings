//go:build acceptance

package githubactions

import (
	"crypto/sha256"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestBuiltCompositeArchivedAutolinks(t *testing.T) {
	const target = "https://fixture-private.example/<num>?q=A%2FB"
	const private = "UPSTREAM_PRIVATE_AUTOLINK_CANARY"
	const remoteOnly = "REMOTE_ONLY_AUTOLINK_CANARY"
	product, version := suppliedCompositeProduct(t)
	for _, scenario := range []string{"matching", "different", "missing", "denied", "duplicate member", "idless", "prune", "owner drift", "archive drift", "mixed", "export flow"} {
		t.Run(scenario, func(t *testing.T) {
			f := newCompositeFixture(t)
			f.base["GITHUB_ACTIONS"], f.base["GH_DEBUG"] = "true", "api"
			f.digest = fmt.Sprintf("%x", sha256.Sum256(product))
			f.write(f.base["MOCK_ASSET"], string(product))
			f.write(f.base["MOCK_CHECKSUMS"], f.digest+"  gh-settings-linux-amd64\n")
			f.write(filepath.Join(f.packagePath, "action/cli-version"), version+"\n")
			id := "id: 7, "
			if scenario == "idless" {
				id = ""
			}
			selected := "autolinks: [{" + id + "reference_prefix: TICKET-, target_url: '" + target + "', reference_format: numeric}]"
			if scenario == "mixed" {
				selected += ", actions: {variables: [{name: FLAG, value: FIXTURE_VALUE}]}"
			}
			input := "version: daiksud/gh-settings@v0\nrepository: {settings: {general: {danger_zone: {archive_this_repository: true}}, " + selected + "}}\n"
			path := filepath.Join(f.caller, ".github/settings.yml")
			f.write(path, input)
			dir, err := os.MkdirTemp("/tmp", "ghs-archive-autolinks-")
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = os.RemoveAll(dir) })
			listener, err := net.Listen("unix", filepath.Join(dir, "api.sock"))
			if err != nil {
				t.Fatal(err)
			}
			var mu sync.Mutex
			repoReads, reads, variableReads, writes := 0, 0, 0, 0
			exporting := scenario == "export flow"
			server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				w.Header().Set("Content-Type", "application/json")
				if r.Host != "api.github.com" || r.Header.Get("Authorization") != "token autolink-fixture" {
					t.Error("unexpected target or credential")
					w.WriteHeader(400)
					return
				}
				if exporting && r.Method == http.MethodPost && r.URL.Path == "/graphql" {
					body, _ := io.ReadAll(r.Body)
					if strings.Contains(strings.ToLower(string(body)), "mutation") {
						writes++
					}
					w.WriteHeader(403)
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
					owner, archived := 2, true
					if repoReads >= 3 {
						if scenario == "owner drift" {
							owner = 3
						}
						if scenario == "archive drift" {
							archived = false
						}
					}
					fmt.Fprintf(w, `{"id":1,"full_name":"owner/config","private":false,"archived":%t,"owner":{"id":%d,"login":"owner","type":"User"}}`, archived, owner)
				case "/repos/owner/config/autolinks":
					reads++
					if scenario == "denied" {
						w.WriteHeader(403)
						fmt.Fprintf(w, `{"message":%q}`, private)
						return
					}
					if scenario == "missing" {
						fmt.Fprint(w, `[]`)
						return
					}
					if scenario == "duplicate member" {
						fmt.Fprintf(w, `[{"id":8,"id":7,"key_prefix":"TICKET-","url_template":%q,"is_alphanumeric":false}]`, target)
						return
					}
					prefix := "TICKET-"
					if scenario == "different" {
						prefix = "OTHER-"
					}
					if scenario == "export flow" {
						fmt.Fprintf(w, `[{"id":7,"key_prefix":%q,"url_template":%q,"is_alphanumeric":false}]`, prefix, target)
					} else {
						fmt.Fprintf(w, `[{"id":7,"key_prefix":%q,"url_template":%q,"is_alphanumeric":false},{"id":9,"key_prefix":"UNSELECTED-","url_template":"https://fixture.example/%s/<num>","is_alphanumeric":false}]`, prefix, target, remoteOnly)
					}
				case "/repos/owner/config/actions/variables/FLAG":
					variableReads++
					fmt.Fprint(w, `{"name":"FLAG","value":"FIXTURE_VALUE"}`)
				default:
					if scenario != "export flow" {
						t.Error("unexpected route", r.URL.Path)
					}
					w.WriteHeader(403)
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
			options := map[string]string{"github.action_ref": version, "GH_DEBUG": "api"}
			if scenario == "prune" {
				options["inputs.prune"] = "autolinks"
			}
			if scenario == "export flow" {
				if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
				exported := f.run(t, "export", "autolink-fixture", options)
				if exported.status != 0 {
					t.Fatal("export failed", exported.status)
				}
				mu.Lock()
				exporting = false
				mu.Unlock()
				data, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				input = string(data)
				if !strings.Contains(input, "id: 7") || !strings.Contains(input, "reference_prefix: TICKET-") || !strings.Contains(input, "archive_this_repository: true") {
					t.Fatal("export omitted selected identity or archive state")
				}
				plan := f.run(t, "plan", "autolink-fixture", map[string]string{"github.action_ref": version, "inputs.plan-reporting": "status-only"})
				if plan.status != 0 {
					t.Error("plan failed", plan.status)
				}
				if strings.Contains(plan.log+plan.summary+plan.api+plan.calls+fmt.Sprint(plan.outputs), target) {
					t.Error("status-only plan leaked target")
				}
			}
			mu.Lock()
			beforeReads, beforeRepos := reads, repoReads
			mu.Unlock()
			result := f.run(t, "apply", "autolink-fixture", options)
			success := scenario == "matching" || scenario == "mixed" || scenario == "export flow"
			if (result.status == 0) != success {
				t.Error("unexpected Action success", result.status)
			}
			if scenario == "idless" && !strings.Contains(result.log, "existing IDs required") || scenario == "prune" && !strings.Contains(result.log, "autolink prune requires explicit unarchive") {
				t.Error("unexpected rejection", result.log)
			}
			if !success && len(result.outputs) != 0 {
				t.Error("failure published outputs")
			}
			for _, output := range []string{result.log, result.summary, result.api, result.calls, fmt.Sprint(result.outputs)} {
				if strings.Contains(output, private) || strings.Contains(output, remoteOnly) {
					t.Error("upstream body leaked")
				}
			}
			data, err := os.ReadFile(path)
			if err != nil || string(data) != input || f.git("rev-parse", "HEAD") != f.sha {
				t.Error("Action changed checkout", err)
			}
			mu.Lock()
			defer mu.Unlock()
			if writes != 0 {
				t.Error("remote mutation", writes)
			}
			if scenario == "export flow" && (reads-beforeReads != 2 || repoReads-beforeRepos != 4) {
				t.Errorf("apply matching reads: inventory=%d repository=%d", reads-beforeReads, repoReads-beforeRepos)
			}
			if scenario == "mixed" && variableReads == 0 {
				t.Error("mixed variable not observed")
			}
			if scenario != "idless" && scenario != "prune" && reads == 0 {
				t.Error("autolink inventory not observed")
			}
			if success && reads < 2 {
				t.Error("did not verify second inventory", reads)
			}
			if (scenario == "owner drift" || scenario == "archive drift") && repoReads < 3 {
				t.Error("drift not observed", repoReads)
			}
		})
	}
}
