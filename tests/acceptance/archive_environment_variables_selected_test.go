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

func TestArchivedEnvironmentVariablesReadOnlyApply(t *testing.T) {
	const desired = "ENV_ARCHIVED_DESIRED_CANARY"
	const remote = "ENV_ARCHIVED_REMOTE_CANARY"
	const base = "/repos/owner/current/environments/production"
	for _, actions := range []bool{false, true} {
		for _, scenario := range []string{"matching", "different", "denied", "parent drift", "prune"} {
			t.Run(fmt.Sprintf("actions=%t/%s", actions, scenario), func(t *testing.T) {
				var mu sync.Mutex
				parentReads, writes := 0, 0
				reads := map[string]int{}
				fixture, err := startHTTPFixture(func(w http.ResponseWriter, r *http.Request) {
					mu.Lock()
					defer mu.Unlock()
					w.Header().Set("Content-Type", "application/json")
					if r.Method != http.MethodGet {
						writes++
						w.WriteHeader(http.StatusBadRequest)
						return
					}
					reads[r.URL.Path]++
					switch r.URL.Path {
					case "/repos/owner/current":
						fmt.Fprint(w, `{"id":42,"full_name":"owner/current","archived":true}`)
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
				}, false)
				if err != nil {
					t.Fatal(err)
				}
				defer fixture.close()
				env := append(labelIDFixtureEnv(fixture.cfg, actions), "GH_DEBUG=api")
				if err := setupOrdinaryOrigin(fixture.work, env); err != nil {
					t.Fatal(err)
				}
				input := "version: daiksud/gh-settings@v0\nrepository:\n  settings:\n    general: {danger_zone: {archive_this_repository: true}}\n    environments:\n      - name: production\n        variables: [{name: FIRST, value: " + desired + "}, {name: SECOND, value: " + desired + "}]\n"
				if err := writeSettingsFile(fixture.work, input); err != nil {
					t.Fatal(err)
				}
				args := []string{"apply"}
				if scenario == "prune" {
					args = append(args, "--prune", "environment-variables")
				}
				result, err := executeCLI(acceptanceBinary(t), fixture.work, fixture.root, env, args, false)
				if err != nil {
					t.Fatal(err)
				}
				if (result.runErr == nil) != (scenario == "matching") {
					t.Error("unexpected CLI success status", result.runErr)
				}
				output := result.stdout.String() + result.stderr.String()
				if scenario == "prune" && !strings.Contains(output, "archived repository cannot update") {
					t.Error("prune did not reach archived guard")
				}
				if strings.Contains(output, desired) || strings.Contains(output, remote) {
					t.Error("variable value leaked in CLI output")
				}
				data, err := os.ReadFile(filepath.Join(fixture.work, ".github/settings.yml"))
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
}
