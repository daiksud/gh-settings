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

func TestBuiltCompositeArchivedEnvironmentAmbiguousFailsClosed(t *testing.T) {
	product, version := suppliedCompositeProduct(t)
	for _, scenario := range []string{"matching", "inventory duplicate", "detail duplicate", "fresh detail duplicate", "custom duplicate", "fresh custom duplicate", "pattern duplicate", "detail trailing", "detail redirect", "detail partial"} {
		t.Run(scenario, func(t *testing.T) {
			f := newCompositeFixture(t)
			f.base["GITHUB_ACTIONS"] = "true"
			f.digest = fmt.Sprintf("%x", sha256.Sum256(product))
			f.write(f.base["MOCK_ASSET"], string(product))
			f.write(f.base["MOCK_CHECKSUMS"], f.digest+"  gh-settings-linux-amd64\n")
			input := `version: daiksud/gh-settings@v0
repository:
  settings:
    general: {danger_zone: {archive_this_repository: true}}
    environments:
      - name: production
        deployment_protection_rules: {wait_timer: 5, custom_rules: []}
        deployment_branches_and_tags: selected_branches_and_tags
        deployment_branch_or_tag_rules: [{ref_type: branch, name_pattern: main}]
`
			path := filepath.Join(f.caller, ".github/settings.yml")
			f.write(path, input)
			dir, err := os.MkdirTemp("/tmp", "ghs-archive-env-json-")
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = os.RemoveAll(dir) })
			listener, err := net.Listen("unix", filepath.Join(dir, "api.sock"))
			if err != nil {
				t.Fatal(err)
			}
			var mu sync.Mutex
			reads := map[string]int{}
			writes := 0
			base := "/repos/owner/config/environments/production"
			server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				if r.Host != "api.github.com" || r.Header.Get("Authorization") != "token archive-fixture" {
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
				w.Header().Set("Content-Type", "application/json")
				switch r.URL.Path {
				case "/repos/owner/config":
					fmt.Fprint(w, `{"id":71,"full_name":"owner/config","archived":true}`)
				case "/repos/owner/config/environments":
					if scenario == "inventory duplicate" {
						fmt.Fprint(w, `{"total_count":2,"total_count":1,"environments":[{"id":99,"id":7,"name":"production"}]}`)
					} else {
						fmt.Fprint(w, `{"total_count":1,"environments":[{"id":7,"name":"production"}]}`)
					}
				case base:
					if scenario == "detail duplicate" || scenario == "fresh detail duplicate" && reads[base] == 2 {
						fmt.Fprint(w, `{"id":7,"name":"production","protection_rules":[{"type":"wait_timer","wait_timer":4,"wait_timer":5}],"deployment_branch_policy":{"protected_branches":false,"custom_branch_policies":true},"can_admins_bypass":false}`)
					} else {
						if scenario == "detail redirect" {
							w.Header().Set("Location", "https://example.invalid/redirected")
							w.WriteHeader(http.StatusFound)
						}
						if scenario == "detail partial" {
							w.WriteHeader(http.StatusPartialContent)
						}
						fmt.Fprint(w, `{"id":7,"name":"production","protection_rules":[{"type":"wait_timer","wait_timer":5}],"deployment_branch_policy":{"protected_branches":false,"custom_branch_policies":true},"can_admins_bypass":false}`)
						if scenario == "detail trailing" {
							fmt.Fprint(w, ` {}`)
						}
					}
				case base + "/deployment_protection_rules":
					if scenario == "custom duplicate" || scenario == "fresh custom duplicate" && reads[r.URL.Path] == 2 {
						fmt.Fprint(w, `{"total_count":1,"total_count":0,"custom_deployment_protection_rules":[{"id":91}],"custom_deployment_protection_rules":[]}`)
					} else {
						fmt.Fprint(w, `{"total_count":0,"custom_deployment_protection_rules":[]}`)
					}
				case base + "/deployment-branch-policies":
					if scenario == "pattern duplicate" {
						fmt.Fprint(w, `{"total_count":1,"branch_policies":[{"id":17,"name":"other","name":"main","type":"branch"}]}`)
					} else {
						fmt.Fprint(w, `{"total_count":1,"branch_policies":[{"id":17,"name":"main","type":"branch"}]}`)
					}
				default:
					t.Errorf("unexpected route %s", r.URL.Path)
					w.WriteHeader(http.StatusBadRequest)
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
			if (result.status == 0) != (scenario == "matching") || result.status != 0 && len(result.outputs) != 0 {
				t.Fatalf("unexpected result: %+v", result)
			}
			mu.Lock()
			defer mu.Unlock()
			if writes != 0 {
				t.Fatal("unexpected writes", writes)
			}
			target, wantReads := "", 1
			switch scenario {
			case "inventory duplicate":
				target = "/repos/owner/config/environments"
			case "detail duplicate", "detail trailing", "detail redirect", "detail partial":
				target = base
			case "fresh detail duplicate":
				target, wantReads = base, 2
			case "custom duplicate":
				target = base + "/deployment_protection_rules"
			case "fresh custom duplicate":
				target, wantReads = base+"/deployment_protection_rules", 2
			case "pattern duplicate":
				target = base + "/deployment-branch-policies"
			}
			if target != "" && reads[target] != wantReads {
				t.Fatalf("targeted response not reached: %s reads=%d, want=%d", target, reads[target], wantReads)
			}
			if scenario == "matching" && (reads[base] != 2 || reads[base+"/deployment_protection_rules"] != 2 || reads[base+"/deployment-branch-policies"] != 2) {
				t.Fatal("selected controls not read twice", reads)
			}
			data, err := os.ReadFile(path)
			if err != nil || string(data) != input || f.git("rev-parse", "HEAD") != f.sha {
				t.Fatal("configuration or commit changed", err)
			}
		})
	}
}
