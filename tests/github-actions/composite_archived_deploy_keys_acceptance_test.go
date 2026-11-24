//go:build acceptance

package githubactions

import (
	"encoding/json"
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

// This executes the supplied product through the real composite against a local API.
// It does not establish that an archived GitHub repository permits these reads.
func TestBuiltCompositeArchivedDeployKeys(t *testing.T) {
	const key = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIAEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEB"
	const otherKey = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIAICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgIC"
	const private = "UPSTREAM_PRIVATE_DEPLOY_KEY_CANARY"
	product := suppliedCompositeProduct(t)
	for _, scenario := range []string{"matching", "title mismatch", "key mismatch", "permission mismatch", "missing", "denied", "duplicate member", "idless", "prune", "mixed", "export flow"} {
		t.Run(scenario, func(t *testing.T) {
			f := product.newFixture(t)
			f.base["GITHUB_ACTIONS"], f.base["GH_DEBUG"] = "true", "api"

			id := "id: 7, "
			if scenario == "idless" {
				id = ""
			}
			selected := "deploy_keys: [{" + id + "title: build, key: '" + key + " source-comment', allow_write_access: false}]"
			if scenario == "mixed" {
				selected += ", actions: {variables: [{name: FLAG, value: FIXTURE_VALUE}]}"
			}
			input := "version: daiksud/gh-settings@v0\nrepository: {settings: {general: {danger_zone: {archive_this_repository: true}}, " + selected + "}}\n"
			path := filepath.Join(f.caller, ".github/settings.yml")
			f.write(path, input)
			head, index := f.git("rev-parse", "HEAD"), f.git("write-tree")
			dir, err := os.MkdirTemp("/tmp", "ghs-archive-deploy-key-")
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = os.RemoveAll(dir) })
			listener, err := net.Listen("unix", filepath.Join(dir, "api.sock"))
			if err != nil {
				t.Fatal(err)
			}
			var mu sync.Mutex
			inventoryReads, detailReads, variableReads, writes := 0, 0, 0, 0
			exporting := scenario == "export flow"
			server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				w.Header().Set("Content-Type", "application/json")
				if r.Host != "api.github.com" || r.Header.Get("Authorization") != "token deploy-key-fixture" {
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
					fmt.Fprint(w, `{"id":1,"full_name":"owner/config","private":false,"visibility":"public","archived":true,"owner":{"id":2,"login":"owner","type":"User"}}`)
				case "/repos/owner/config/keys", "/repos/owner/config/keys/7":
					if r.URL.Path == "/repos/owner/config/keys" {
						inventoryReads++
					} else {
						detailReads++
					}
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
						fmt.Fprintf(w, `[{"id":8,"id":7,"title":"build","key":%q,"read_only":true,"verified":false,"enabled":true,"created_at":"2026-11-14T00:00:00Z","url":"https://api.github.com/repos/owner/config/keys/7"}]`, key)
						return
					}
					title, material, readOnly := "build", key, true
					if scenario == "title mismatch" {
						title = "other"
					}
					if scenario == "key mismatch" {
						material = otherKey
					}
					if scenario == "permission mismatch" {
						readOnly = false
					}
					row := map[string]any{"id": 7, "title": title, "key": material, "read_only": readOnly, "verified": false, "enabled": true, "created_at": "2026-11-14T00:00:00Z", "url": "https://api.github.com/repos/owner/config/keys/7", "added_by": private, "last_used": nil}
					if r.URL.Path == "/repos/owner/config/keys" {
						_ = json.NewEncoder(w).Encode([]any{row})
					} else {
						_ = json.NewEncoder(w).Encode(row)
					}
				case "/repos/owner/config/actions/variables/FLAG":
					variableReads++
					fmt.Fprint(w, `{"name":"FLAG","value":"FIXTURE_VALUE"}`)
				default:
					if !exporting {
						t.Error("unexpected route", r.URL.Path)
					}
					w.WriteHeader(403)
				}
			})}
			go func() { _ = server.Serve(listener) }()
			t.Cleanup(func() { _ = server.Close() })
			f.write(filepath.Join(dir, "config.yml"), "http_unix_socket: "+filepath.Join(dir, "api.sock")+"\n")
			f.base["GH_CONFIG_DIR"], f.base["GH_PATH_DISABLED"] = dir, "1"
			for _, name := range []string{"HTTP_PROXY", "HTTPS_PROXY", "http_proxy", "https_proxy"} {
				f.base[name] = "http://127.0.0.1:1"
			}
			f.base["NO_PROXY"], f.base["no_proxy"] = "", ""
			options := map[string]string{"GH_DEBUG": "api"}
			if scenario == "prune" {
				options["inputs.prune"] = "deploy-keys"
			}
			checkPrivacy := func(r compositeResult) {
				t.Helper()
				text := r.log + r.summary + r.api + r.calls + fmt.Sprint(r.outputs)
				for _, forbidden := range []string{key, otherKey, private} {
					if strings.Contains(text, forbidden) {
						t.Error("raw key or private response appeared in Action output")
					}
				}
			}
			if scenario == "export flow" {
				if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
				exported := f.run(t, "export", "deploy-key-fixture", options)
				checkPrivacy(exported)
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
				if !strings.Contains(input, "id: 7") || !strings.Contains(input, key) || !strings.Contains(input, "archive_this_repository: true") {
					t.Fatal("export omitted selected identity, key or archive state")
				}
				plan := f.run(t, "plan", "deploy-key-fixture", options)
				checkPrivacy(plan)
				if plan.status != 0 || len(plan.remainingReports) != 0 {
					t.Error("plan failed or retained a report", plan.status)
				}
			}
			mu.Lock()
			beforeInventory := inventoryReads
			mu.Unlock()
			result := f.run(t, "apply", "deploy-key-fixture", options)
			checkPrivacy(result)
			success := scenario == "matching" || scenario == "mixed" || scenario == "export flow"
			if (result.status == 0) != success {
				t.Error("unexpected Action status", result.status)
			}
			if !success && len(result.outputs) != 0 {
				t.Error("failed apply published outputs")
			}
			data, err := os.ReadFile(path)
			if err != nil || string(data) != input || f.git("rev-parse", "HEAD") != head || f.git("write-tree") != index {
				t.Error("apply changed declaration, HEAD or index", err)
			}
			mu.Lock()
			defer mu.Unlock()
			if writes != 0 {
				t.Error("unexpected remote mutation", writes)
			}
			if scenario == "mixed" && variableReads == 0 {
				t.Error("mixed variable was not observed")
			}
			if success && inventoryReads-beforeInventory < 2 {
				t.Error("apply did not independently reread the complete inventory", inventoryReads-beforeInventory)
			}
		})
	}
}
