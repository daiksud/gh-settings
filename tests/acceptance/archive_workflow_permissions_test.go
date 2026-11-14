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

func TestArchivedWorkflowPermissionsCLIReadOnly(t *testing.T) {
	for _, scenario := range []string{"matching", "default only", "approval only", "approval only unknown default", "default differs", "approval differs", "selected normalized", "missing approval", "denied", "duplicate default", "identity drift"} {
		t.Run(scenario, func(t *testing.T) {
			var mu sync.Mutex
			writes, repoReads, workflowReads := 0, 0, 0
			fixture, err := startHTTPFixture(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				w.Header().Set("Content-Type", "application/json")
				if r.Method != http.MethodGet {
					writes++
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				switch r.URL.Path {
				case "/repos/owner/current":
					repoReads++
					id := 42
					if scenario == "identity drift" && repoReads == 2 {
						id = 43
					}
					fmt.Fprintf(w, `{"id":%d,"full_name":"owner/current","archived":true}`, id)
				case "/repos/owner/current/actions/permissions/workflow":
					workflowReads++
					if scenario == "denied" {
						w.WriteHeader(http.StatusForbidden)
						return
					}
					if scenario == "duplicate default" {
						fmt.Fprint(w, `{"default_workflow_permissions":"write","default_workflow_permissions":"read","can_approve_pull_request_reviews":false}`)
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
			}, false)
			if err != nil {
				t.Fatal(err)
			}
			defer fixture.close()
			env := exportFixtureEnv(fixture.cfg)
			if err := setupOrdinaryOrigin(fixture.work, env); err != nil {
				t.Fatal(err)
			}
			selection := "default_permissions: read_repository_contents_and_packages_permissions, allow_github_actions_to_create_and_approve_pull_requests: false"
			if scenario == "default only" || scenario == "selected normalized" {
				selection = "default_permissions: read_repository_contents_and_packages_permissions"
			}
			if scenario == "approval only" || scenario == "approval only unknown default" {
				selection = "allow_github_actions_to_create_and_approve_pull_requests: false"
			}
			input := repositoryInput("settings: {general: {danger_zone: {archive_this_repository: true}}, actions: {general: {workflow_permissions: {" + selection + "}}}}\n")
			if err := writeSettingsFile(fixture.work, input); err != nil {
				t.Fatal(err)
			}
			result, err := executeCLI(acceptanceBinary(t), fixture.work, fixture.root, env, []string{"apply"}, false)
			want := scenario == "matching" || scenario == "default only" || scenario == "approval only" || scenario == "approval only unknown default"
			if err != nil || (result.runErr == nil) != want {
				t.Fatalf("err=%v result=%v stderr=%s", err, result.runErr, result.stderr.String())
			}
			mu.Lock()
			defer mu.Unlock()
			data, err := os.ReadFile(filepath.Join(fixture.work, ".github/settings.yml"))
			if err != nil || string(data) != input || writes != 0 || workflowReads != 1 || want && repoReads != 2 {
				t.Fatalf("config error=%v writes=%d repo reads=%d workflow reads=%d", err, writes, repoReads, workflowReads)
			}
		})
	}
}
