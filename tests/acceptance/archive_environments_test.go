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

func TestArchivedEnvironmentBinaryReadOnly(t *testing.T) {
	for _, scenario := range []string{"matching", "about differs", "timer differs", "app differs", "unknown pattern", "second read denied", "identity changed"} {
		t.Run(scenario, func(t *testing.T) {
			var mu sync.Mutex
			counts := map[string]int{}
			writes := 0
			base := "/repos/owner/current/environments/production"
			f, err := startHTTPFixture(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				if r.Method != "GET" {
					writes++
					w.WriteHeader(400)
					return
				}
				counts[r.URL.Path]++
				w.Header().Set("Content-Type", "application/json")
				switch r.URL.Path {
				case "/repos/owner/current":
					description := "same"
					if scenario == "about differs" {
						description = "other"
					}
					fmt.Fprintf(w, `{"id":71,"full_name":"owner/current","archived":true,"description":%q,"homepage":"","visibility":"public","default_branch":"main","has_issues":true,"has_projects":false,"has_wiki":false,"has_discussions":false,"web_commit_signoff_required":false,"is_template":false}`, description)
				case "/repos/owner/current/environments":
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
					timer := 5
					if scenario == "timer differs" {
						timer = 4
					}
					fmt.Fprintf(w, `{"id":%d,"name":"production","protection_rules":[{"type":"wait_timer","wait_timer":%d}],"deployment_branch_policy":{"protected_branches":false,"custom_branch_policies":true},"can_admins_bypass":false}`, id, timer)
				case base + "/deployment_protection_rules":
					if scenario == "second read denied" && counts[r.URL.Path] > 1 {
						w.WriteHeader(403)
						fmt.Fprint(w, `{"message":"denied"}`)
						return
					}
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
					t.Error("unexpected request", r.URL.Path)
					w.WriteHeader(400)
				}
			}, false, "/environments", "/environments/production")
			if err != nil {
				t.Fatal(err)
			}
			defer f.close()
			env := labelIDFixtureEnv(f.cfg, false)
			if err := setupOrdinaryOrigin(f.work, env); err != nil {
				t.Fatal(err)
			}
			input := `version: daiksud/gh-settings@v0
repository:
  about: {description: same}
  settings:
    general: {danger_zone: {archive_this_repository: true}}
    environments:
      - name: production
        deployment_protection_rules: {wait_timer: 5, enable_custom_rules_with_github_apps: [101]}
        deployment_branches_and_tags: selected_branches_and_tags
        deployment_branch_or_tag_rules: [{ref_type: branch, name_pattern: main}]
`
			if err := writeSettingsFile(f.work, input); err != nil {
				t.Fatal(err)
			}
			got, err := executeCLI(acceptanceBinary(t), f.work, f.root, env, []string{"apply"}, false)
			if err != nil {
				t.Fatal(err)
			}
			if (got.runErr == nil) != (scenario == "matching") {
				t.Fatalf("unexpected result: %v %s %s", got.runErr, got.stdout.String(), got.stderr.String())
			}
			data, err := os.ReadFile(filepath.Join(f.work, ".github/settings.yml"))
			if err != nil || string(data) != input {
				t.Fatal("settings changed", err, string(data))
			}
			mu.Lock()
			defer mu.Unlock()
			if writes != 0 {
				t.Fatal("unexpected writes", writes)
			}
			if scenario == "matching" && (counts[base] != 2 || counts[base+"/deployment_protection_rules"] != 2 || counts[base+"/deployment-branch-policies"] != 2) {
				t.Fatal("selected controls not re-read", counts)
			}
		})
	}
}
