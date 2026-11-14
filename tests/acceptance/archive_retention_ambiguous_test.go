//go:build acceptance

package acceptance

import (
	"fmt"
	"net/http"
	"sync"
	"testing"
)

func TestArchivedRetentionRejectsAmbiguousObservation(t *testing.T) {
	for _, scenario := range []string{"matching", "duplicate days", "duplicate maximum", "alias"} {
		t.Run(scenario, func(t *testing.T) {
			var mu sync.Mutex
			reads, writes := 0, 0
			fixture, err := startHTTPFixture(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				if r.Method != http.MethodGet {
					writes++
					w.WriteHeader(400)
					return
				}
				switch r.URL.Path {
				case "/repos/owner/current":
					fmt.Fprint(w, `{"id":42,"full_name":"owner/current","archived":true}`)
				case "/repos/owner/current/actions/permissions/artifact-and-log-retention":
					reads++
					body := `{"days":30,"maximum_allowed_days":90}`
					switch scenario {
					case "duplicate days":
						body = `{"days":10,"days":30,"maximum_allowed_days":90}`
					case "duplicate maximum":
						body = `{"days":30,"maximum_allowed_days":10,"maximum_allowed_days":90}`
					case "alias":
						body = `{"Days":10,"days":30,"maximum_allowed_days":90}`
					}
					fmt.Fprint(w, body)
				default:
					t.Errorf("unexpected path %s", r.URL.Path)
					w.WriteHeader(400)
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
			input := repositoryInput("settings: {general: {danger_zone: {archive_this_repository: true}}, actions: {general: {check_workflow_run_status_artifact_and_log_retention: 30}}}\n")
			if err := writeSettingsFile(fixture.work, input); err != nil {
				t.Fatal(err)
			}
			result, err := executeCLI(acceptanceBinary(t), fixture.work, fixture.root, env, []string{"apply"}, false)
			if err != nil || (result.runErr == nil) != (scenario == "matching") {
				t.Fatalf("err=%v result=%v stderr=%s", err, result.runErr, result.stderr.String())
			}
			mu.Lock()
			defer mu.Unlock()
			if writes != 0 || reads != 1 {
				t.Fatalf("reads=%d writes=%d", reads, writes)
			}
		})
	}
}
