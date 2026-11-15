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

func TestArchivedEnvironmentVariablesCLIReadOnly(t *testing.T) {
	const desired = "FIXTURE_ENV_VAR_DESIRED_CANARY"
	const other = "FIXTURE_ENV_VAR_OTHER_CANARY"
	for _, scenario := range []string{"matching", "different", "absent", "denied", "malformed", "parent drift", "repository drift", "prune"} {
		t.Run(scenario, func(t *testing.T) {
			var mu sync.Mutex
			counts := map[string]int{}
			writes := 0
			f, err := startHTTPFixture(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				w.Header().Set("Content-Type", "application/json")
				if r.Method != http.MethodGet {
					writes++
					w.WriteHeader(400)
					return
				}
				counts[r.URL.Path]++
				switch r.URL.Path {
				case "/repos/owner/current":
					id := 71
					if scenario == "repository drift" && counts[r.URL.Path] > 1 {
						id = 72
					}
					fmt.Fprintf(w, `{"id":%d,"full_name":"owner/current","archived":true}`, id)
				case "/repos/owner/current/environments/production", "/repos/owner/current/environments/staging":
					name, id := "production", 7
					if strings.HasSuffix(r.URL.Path, "/staging") {
						name, id = "staging", 8
					}
					if scenario == "parent drift" && name == "production" && counts[r.URL.Path] > 1 {
						id = 9
					}
					fmt.Fprintf(w, `{"id":%d,"name":%q}`, id, name)
				case "/repos/owner/current/environments/production/variables", "/repos/owner/current/environments/staging/variables":
					fmt.Fprintf(w, `{"total_count":1,"variables":[{"name":"CHANNEL","value":%q}]}`, desired)
				case "/repos/owner/current/environments/production/variables/CHANNEL", "/repos/owner/current/environments/staging/variables/CHANNEL":
					if scenario == "denied" {
						w.WriteHeader(403)
						fmt.Fprintf(w, `{"message":%q}`, other)
						return
					}
					if scenario == "absent" {
						w.WriteHeader(404)
						return
					}
					if scenario == "malformed" {
						fmt.Fprint(w, `{"name":"CHANNEL"}`)
						return
					}
					value := desired
					if scenario == "different" {
						value = other
					}
					fmt.Fprintf(w, `{"name":"CHANNEL","value":%q}`, value)
				default:
					t.Error("unexpected request", r.URL.Path)
					w.WriteHeader(400)
				}
			}, false, "/environments")
			if err != nil {
				t.Fatal(err)
			}
			defer f.close()
			env := append(exportFixtureEnv(f.cfg), "GH_DEBUG=api")
			if err := setupOrdinaryOrigin(f.work, env); err != nil {
				t.Fatal(err)
			}
			input := repositoryInput("settings:\n  general: {danger_zone: {archive_this_repository: true}}\n  environments:\n    - name: production\n      variables: [{name: CHANNEL, value: " + desired + "}]\n    - name: staging\n      variables: [{name: CHANNEL, value: " + desired + "}]\n")
			if err := writeSettingsFile(f.work, input); err != nil {
				t.Fatal(err)
			}
			args := []string{"apply"}
			if scenario == "prune" {
				args = append(args, "--prune", "environment-variables")
			}
			result, err := executeCLI(acceptanceBinary(t), f.work, f.root, env, args, false)
			if err != nil || (result.runErr == nil) != (scenario == "matching") {
				t.Fatal("unexpected CLI result", err, result.runErr)
			}
			if scenario == "prune" && !strings.Contains(result.stdout.String()+result.stderr.String(), "archived repository cannot update") {
				t.Fatal("prune did not reach archived guard")
			}
			for _, output := range []string{result.stdout.String(), result.stderr.String()} {
				if strings.Contains(output, desired) || strings.Contains(output, other) {
					t.Fatal("variable value leaked into CLI output")
				}
			}
			data, err := os.ReadFile(filepath.Join(f.work, ".github/settings.yml"))
			if err != nil || string(data) != input {
				t.Fatal("settings changed", err)
			}
			mu.Lock()
			defer mu.Unlock()
			if writes != 0 {
				t.Fatal("unexpected writes", writes)
			}
			if (scenario == "absent" || scenario == "malformed") && counts["/repos/owner/current/environments/production/variables/CHANNEL"] == 0 {
				t.Fatal("selected variable failure not exercised")
			}
			if scenario == "repository drift" && counts["/repos/owner/current"] != 2 {
				t.Fatal("repository drift not exercised")
			}
			if scenario == "parent drift" && counts["/repos/owner/current/environments/production"] != 2 {
				t.Fatal("parent drift not exercised")
			}
			if scenario == "matching" && (counts["/repos/owner/current"] != 2 || counts["/repos/owner/current/environments/production"] != 2 || counts["/repos/owner/current/environments/staging"] != 2 || counts["/repos/owner/current/environments/production/variables/CHANNEL"] != 1 || counts["/repos/owner/current/environments/staging/variables/CHANNEL"] != 1) {
				t.Fatal("missing read-only evidence", counts)
			}
		})
	}
}
