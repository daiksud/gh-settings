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

func TestBuiltCompositeArchivedWorkflowPermissions(t *testing.T) {
	product := suppliedCompositeProduct(t)
	for _, scenario := range []string{"matching", "approval only unknown default", "default differs", "approval differs", "selected normalized", "missing approval", "duplicate default", "alias default", "alias approval", "denied", "identity drift"} {
		t.Run(scenario, func(t *testing.T) {
			f := product.newFixture(t)
			f.base["GITHUB_ACTIONS"] = "true"

			selection := "default_permissions: read_repository_contents_and_packages_permissions, allow_github_actions_to_create_and_approve_pull_requests: false"
			if scenario == "approval only unknown default" {
				selection = "allow_github_actions_to_create_and_approve_pull_requests: false"
			}
			input := "version: daiksud/gh-settings@v0\nrepository: {settings: {general: {danger_zone: {archive_this_repository: true}}, actions: {general: {workflow_permissions: {" + selection + "}}}}}\n"
			path := filepath.Join(f.caller, ".github/settings.yml")
			f.write(path, input)
			dir, err := os.MkdirTemp("/tmp", "ghs-archive-workflow-")
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = os.RemoveAll(dir) })
			listener, err := net.Listen("unix", filepath.Join(dir, "api.sock"))
			if err != nil {
				t.Fatal(err)
			}
			var mu sync.Mutex
			writes, repoReads, workflowReads := 0, 0, 0
			server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				w.Header().Set("Content-Type", "application/json")
				if r.Host != "api.github.com" || r.Header.Get("Authorization") != "token archive-workflow-fixture" {
					t.Error("unexpected target or credential")
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				if r.Method != http.MethodGet {
					writes++
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				switch r.URL.Path {
				case "/repos/owner/config":
					repoReads++
					id := 42
					if scenario == "identity drift" && repoReads == 2 {
						id = 43
					}
					fmt.Fprintf(w, `{"id":%d,"full_name":"owner/config","archived":true}`, id)
				case "/repos/owner/config/actions/permissions/workflow":
					workflowReads++
					if scenario == "denied" {
						w.WriteHeader(http.StatusForbidden)
						return
					}
					if scenario == "duplicate default" {
						fmt.Fprint(w, `{"default_workflow_permissions":"write","default_workflow_permissions":"read","can_approve_pull_request_reviews":false}`)
						return
					}
					if scenario == "alias default" {
						fmt.Fprint(w, `{"DEFAULT_WORKFLOW_PERMISSIONS":"write","default_workflow_permissions":"read","can_approve_pull_request_reviews":false}`)
						return
					}
					if scenario == "alias approval" {
						fmt.Fprint(w, `{"default_workflow_permissions":"read","CAN_APPROVE_PULL_REQUEST_REVIEWS":true,"can_approve_pull_request_reviews":false}`)
						return
					}
					if scenario == "missing approval" {
						fmt.Fprint(w, `{"default_workflow_permissions":"read"}`)
						return
					}
					permission, approval := "read", false
					if scenario == "default differs" {
						permission = "write"
					}
					if scenario == "approval differs" {
						approval = true
					}
					if scenario == "selected normalized" || scenario == "approval only unknown default" {
						permission = "FUTURE"
					}
					fmt.Fprintf(w, `{"default_workflow_permissions":%q,"can_approve_pull_request_reviews":%t}`, permission, approval)
				default:
					t.Errorf("unexpected path %s", r.URL.Path)
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
			result := f.run(t, "apply", "archive-workflow-fixture", nil)
			want := scenario == "matching" || scenario == "approval only unknown default"
			if (result.status == 0) != want || result.status != 0 && len(result.outputs) != 0 {
				t.Fatalf("unexpected result %+v", result)
			}
			mu.Lock()
			defer mu.Unlock()
			data, err := os.ReadFile(path)
			if err != nil || string(data) != input || writes != 0 || workflowReads != 1 || want && repoReads != 2 {
				t.Fatalf("config error=%v writes=%d repo reads=%d workflow reads=%d", err, writes, repoReads, workflowReads)
			}
		})
	}
}
