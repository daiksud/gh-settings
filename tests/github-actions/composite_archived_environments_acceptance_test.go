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

func TestBuiltCompositeArchivedEnvironments(t *testing.T) {
	product, version := suppliedCompositeProduct(t)
	for _, scenario := range []string{"matching", "about differs", "app differs", "unknown pattern", "identity changed"} {
		t.Run(scenario, func(t *testing.T) {
			f := newCompositeFixture(t)
			f.base["GITHUB_ACTIONS"] = "true"
			f.digest = fmt.Sprintf("%x", sha256.Sum256(product))
			f.write(f.base["MOCK_ASSET"], string(product))
			f.write(f.base["MOCK_CHECKSUMS"], f.digest+"  gh-settings-linux-amd64\n")
			input := `version: daiksud/gh-settings@v0
repository:
  about: {description: same}
  settings:
    general: {danger_zone: {archive_this_repository: true}}
    environments:
      - name: production
        deployment_protection_rules: {wait_timer: 5, custom_rules: [101]}
        deployment_branches_and_tags: selected_branches_and_tags
        deployment_branch_or_tag_rules: [{ref_type: branch, name_pattern: main}]
`
			path := filepath.Join(f.caller, ".github/settings.yml")
			f.write(path, input)
			dir, err := os.MkdirTemp("/tmp", "ghs-archive-env-")
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
			base := "/repos/owner/config/environments/production"
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
				counts[r.URL.Path]++
				w.Header().Set("Content-Type", "application/json")
				switch r.URL.Path {
				case "/repos/owner/config":
					description := "same"
					if scenario == "about differs" {
						description = "other"
					}
					fmt.Fprintf(w, `{"id":71,"full_name":"owner/config","archived":true,"description":%q,"homepage":"","visibility":"public","default_branch":"main","has_issues":true,"has_projects":false,"has_wiki":false,"has_discussions":false,"web_commit_signoff_required":false,"is_template":false}`, description)
				case "/repos/owner/config/environments":
					id := 7
					if scenario == "identity changed" && counts[r.URL.Path] > 1 {
						id = 8
					}
					fmt.Fprintf(w, `{"total_count":1,"environments":[{"id":%d,"name":"production"}]}`, id)
				case base:
					id := 7
					if scenario == "identity changed" && counts[r.URL.Path] > 1 {
						id = 8
					}
					fmt.Fprintf(w, `{"id":%d,"name":"production","protection_rules":[{"type":"wait_timer","wait_timer":5}],"deployment_branch_policy":{"protected_branches":false,"custom_branch_policies":true},"can_admins_bypass":false}`, id)
				case base + "/deployment_protection_rules":
					id := 101
					if scenario == "app differs" {
						id = 102
					}
					fmt.Fprintf(w, `{"total_count":1,"custom_deployment_protection_rules":[{"id":91,"node_id":"rule91","enabled":true,"app":{"id":%d,"node_id":"app101","slug":"custom-app","integration_url":"https://api.github.com/apps/custom-app"}}]}`, id)
				case base + "/deployment-branch-policies":
					kind := "branch"
					if scenario == "unknown pattern" {
						kind = "future"
					}
					fmt.Fprintf(w, `{"total_count":1,"branch_policies":[{"id":17,"name":"main","type":%q}]}`, kind)
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
			if (result.status == 0) != (scenario == "matching") || (result.status != 0 && len(result.outputs) != 0) {
				t.Fatalf("unexpected result: %+v", result)
			}
			mu.Lock()
			defer mu.Unlock()
			if writes != 0 {
				t.Fatal("unexpected writes", writes)
			}
			if scenario == "matching" && (counts[base] != 2 || counts[base+"/deployment_protection_rules"] != 2 || counts[base+"/deployment-branch-policies"] != 2) {
				t.Fatal("selected controls not re-read", counts)
			}
			data, err := os.ReadFile(path)
			if err != nil || string(data) != input || f.git("rev-parse", "HEAD") != f.sha {
				t.Fatal("configuration or commit changed", err)
			}
		})
	}
}
