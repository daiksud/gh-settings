//go:build acceptance

package acceptance

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestArchivedVariablesReadOnlyApply(t *testing.T) {
	const desired = "FIXTURE_ARCHIVED_DESIRED_CANARY"
	const remote = "FIXTURE_ARCHIVED_REMOTE_CANARY"
	for _, scenario := range []string{"matching", "different", "absent", "denied", "malformed", "wrong name", "duplicate value", "value alias", "duplicate name", "name alias", "identity drift"} {
		t.Run(scenario, func(t *testing.T) {
			var mu sync.Mutex
			writes, repoReads, variableReads := 0, 0, 0
			fixture, err := startHTTPFixture(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				w.Header().Set("Content-Type", "application/json")
				if r.Method != http.MethodGet {
					writes++
					w.WriteHeader(400)
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
				case "/repos/owner/current/actions/variables/BUILD_CHANNEL", "/repos/owner/current/agents/variables/MODEL_FLAG":
					variableReads++
					if scenario == "denied" {
						w.WriteHeader(403)
						fmt.Fprintf(w, `{"message":%q}`, remote)
						return
					}
					if scenario == "absent" {
						w.WriteHeader(404)
						return
					}
					if scenario == "malformed" {
						fmt.Fprint(w, `{"name":"BUILD_CHANNEL"}`)
						return
					}
					name := "BUILD_CHANNEL"
					if strings.Contains(r.URL.Path, "/agents/") {
						name = "MODEL_FLAG"
					}
					if scenario == "wrong name" {
						name = "OTHER"
					}
					if scenario == "duplicate value" {
						fmt.Fprintf(w, `{"name":%q,"value":%q,"value":%q}`, name, remote, desired)
						return
					}
					if scenario == "value alias" {
						fmt.Fprintf(w, `{"name":%q,"Value":%q,"value":%q}`, name, remote, desired)
						return
					}
					if scenario == "duplicate name" {
						fmt.Fprintf(w, `{"name":"OTHER","name":%q,"value":%q}`, name, desired)
						return
					}
					if scenario == "name alias" {
						fmt.Fprintf(w, `{"Name":"OTHER","name":%q,"value":%q}`, name, desired)
						return
					}
					value := desired
					if scenario == "different" {
						value = remote
					}
					fmt.Fprintf(w, `{"name":%q,"value":%q}`, name, value)
				default:
					t.Error("unexpected request path")
					w.WriteHeader(400)
				}
			}, false, "/labels", "/topics")
			if err != nil {
				t.Fatal(err)
			}
			defer fixture.close()
			env := append(exportFixtureEnv(fixture.cfg), "GH_DEBUG=api")
			if err := setupOrdinaryOrigin(fixture.work, env); err != nil {
				t.Fatal(err)
			}
			input := repositoryInput("settings:\n  general: {danger_zone: {archive_this_repository: true}}\n  actions: {variables: [{name: BUILD_CHANNEL, value: " + desired + "}]}\n  agents: {variables: [{name: MODEL_FLAG, value: " + desired + "}]}\n")
			if err := writeSettingsFile(fixture.work, input); err != nil {
				t.Fatal(err)
			}
			result, err := executeCLI(acceptanceBinary(t), fixture.work, fixture.root, env, []string{"apply"}, false)
			if err != nil || (result.runErr == nil) != (scenario == "matching") {
				t.Fatal("unexpected CLI result", err, result.runErr)
			}
			if strings.Contains(result.stdout.String(), desired) || strings.Contains(result.stderr.String(), desired) || strings.Contains(result.stderr.String(), remote) {
				t.Fatal("variable value leaked into CLI output")
			}
			mu.Lock()
			defer mu.Unlock()
			if writes != 0 || variableReads == 0 || scenario == "matching" && (repoReads != 2 || variableReads != 2) {
				t.Fatal("unexpected request counts", writes, repoReads, variableReads)
			}
			data, err := os.ReadFile(filepath.Join(fixture.work, ".github/settings.yml"))
			if err != nil || string(data) != input {
				t.Fatal("declaration changed", err)
			}
		})
	}
}
