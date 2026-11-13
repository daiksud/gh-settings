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
	"reflect"
	"strings"
	"sync"
	"testing"
)

func TestBuiltCompositeActionsExecution(t *testing.T) {
	product, version := suppliedCompositeProduct(t)
	for _, scenario := range []string{"matching", "selected change", "disabled activation", "selected failure", "companion drift"} {
		t.Run(scenario, func(t *testing.T) {
			f := newCompositeFixture(t)
			f.digest = fmt.Sprintf("%x", sha256.Sum256(product))
			f.write(f.base["MOCK_ASSET"], string(product))
			f.write(f.base["MOCK_CHECKSUMS"], f.digest+"  gh-settings-linux-amd64\n")
			f.write(filepath.Join(f.packagePath, "action/cli-version"), version+"\n")
			input := "version: daiksud/gh-settings@v0\nrepository:\n  settings:\n    actions:\n      general:\n        actions_permissions:\n          policy: allow_owner_and_select_non_owner_actions_and_reusable_workflows\n          require_actions_to_be_pinned_to_a_full_length_commit_sha: true\n          allow_actions_created_by_github: true\n          allow_actions_by_marketplace_verified_creators: true\n          allow_or_block_specified_actions_and_reusable_workflows: ['keep/*']\n"
			path := filepath.Join(f.caller, ".github/settings.yml")
			f.write(path, input)
			state, err := os.MkdirTemp("/tmp", "ghs-execution-")
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = os.RemoveAll(state) })
			listener, err := net.Listen("unix", filepath.Join(state, "api.sock"))
			if err != nil {
				t.Fatal(err)
			}
			var mu sync.Mutex
			base := map[string]any{"enabled": scenario != "disabled activation" && scenario != "selected failure", "allowed_actions": "selected", "sha_pinning_required": true}
			selected := map[string]any{"github_owned_allowed": true, "verified_allowed": scenario == "matching", "patterns_allowed": []any{"keep/*"}}
			var operations []string
			var payloads []map[string]any
			server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				prefix := "/repos/owner/config/actions/permissions"
				if r.Host != "api.github.com" || r.Header.Get("Authorization") != "token execution-fixture-write" || (r.URL.Path != prefix && r.URL.Path != prefix+"/selected-actions") {
					t.Error("unexpected target/credential boundary", r.Method, r.URL.Path)
					w.WriteHeader(400)
					return
				}
				suffix := strings.TrimPrefix(r.URL.Path, prefix)
				operations = append(operations, r.Method+suffix)
				target := base
				if suffix != "" {
					target = selected
				}
				w.Header().Set("Content-Type", "application/json")
				switch r.Method {
				case "GET":
					json.NewEncoder(w).Encode(target)
				case "PUT":
					var body map[string]any
					if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
						t.Error(err)
					}
					payloads = append(payloads, body)
					if scenario == "selected failure" {
						w.WriteHeader(403)
						fmt.Fprint(w, `{"message":"denied"}`)
						return
					}
					if scenario == "disabled activation" && suffix != "" && base["enabled"] != false {
						t.Error("enabled before selected policy")
					}
					for k, v := range body {
						target[k] = v
					}
					if scenario == "companion drift" {
						base["sha_pinning_required"] = false
					}
					w.WriteHeader(204)
				default:
					t.Error("unexpected method", r.Method)
					w.WriteHeader(400)
				}
			})}
			go func() { _ = server.Serve(listener) }()
			t.Cleanup(func() { _ = server.Close() })
			f.write(filepath.Join(state, "config.yml"), "http_unix_socket: "+filepath.Join(state, "api.sock")+"\n")
			f.base["GH_CONFIG_DIR"], f.base["GH_PATH_DISABLED"] = state, "1"
			for _, key := range []string{"HTTP_PROXY", "HTTPS_PROXY", "http_proxy", "https_proxy"} {
				f.base[key] = "http://127.0.0.1:1"
			}
			f.base["NO_PROXY"], f.base["no_proxy"] = "", ""
			success := scenario != "selected failure" && scenario != "companion drift"
			attempts := 1
			if success {
				attempts = 2
			}
			for i := 0; i < attempts; i++ {
				result := f.run(t, "apply", "execution-fixture-write", map[string]string{"github.action_ref": version})
				if (result.status == 0) != success || success && len(result.outputs) != 4 || !success && len(result.outputs) != 0 {
					t.Fatalf("incorrect success publication: %+v", result)
				}
				if !success && !strings.Contains(result.log, "may already have applied") {
					t.Fatal("uncertainty lost", result.log)
				}
				if strings.Contains(result.log, "earlier resource writes remain applied") {
					t.Fatal("fabricated earlier resource progress", result.log)
				}
			}
			mu.Lock()
			defer mu.Unlock()
			want := []string{"GET", "GET/selected-actions"}
			var wantPayloads []map[string]any
			if scenario != "matching" {
				want = append(want, "PUT/selected-actions")
				wantPayloads = append(wantPayloads, map[string]any{"github_owned_allowed": true, "verified_allowed": true, "patterns_allowed": []any{"keep/*"}})
				if scenario == "disabled activation" {
					want = append(want, "PUT")
					wantPayloads = append(wantPayloads, map[string]any{"enabled": true})
				}
				if scenario != "selected failure" {
					want = append(want, "GET", "GET/selected-actions")
				}
			}
			if success {
				want = append(want, "GET", "GET/selected-actions")
			}
			if !reflect.DeepEqual(operations, want) || !reflect.DeepEqual(payloads, wantPayloads) {
				t.Fatal(operations, want, payloads, wantPayloads)
			}
			if scenario == "selected failure" && base["enabled"] != false {
				t.Fatal("failed selection enabled Actions")
			}
			data, err := os.ReadFile(path)
			if err != nil || string(data) != input || f.git("rev-parse", "HEAD") != f.sha {
				t.Fatal("Action modified settings or commit", err)
			}
		})
	}
}
