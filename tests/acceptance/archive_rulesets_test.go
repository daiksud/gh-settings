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

func TestArchivedRulesetsReadOnlyApply(t *testing.T) {
	binary := acceptanceBinary(t)
	for _, scenario := range []string{"matching", "different", "unknown"} {
		t.Run(scenario, func(t *testing.T) {
			var mu sync.Mutex
			writes, listReads, detailReads := 0, 0, 0
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
					fmt.Fprint(w, `{"id":42,"full_name":"owner/current","archived":true}`)
				case "/repos/owner/current/rulesets":
					listReads++
					fmt.Fprint(w, `[{"id":7,"source_type":"Repository","source":"owner/current"}]`)
				case "/repos/owner/current/rulesets/7":
					detailReads++
					mode := "active"
					if scenario == "different" {
						mode = "disabled"
					}
					if scenario == "unknown" {
						mode = "FUTURE"
					}
					fmt.Fprintf(w, `{"id":7,"name":"coverage","source_type":"Repository","source":"owner/current","target":"branch","enforcement":%q,"conditions":{"ref_name":{"include":["~DEFAULT_BRANCH"],"exclude":[]}},"bypass_actors":[],"rules":[{"type":"code_coverage","parameters":{"minimum_coverage":90.25,"max_coverage_drop":0}}]}`, mode)
				default:
					t.Error("unexpected read", r.URL)
					w.WriteHeader(400)
				}
			}, false, "/labels", "/topics")
			if err != nil {
				t.Fatal(err)
			}
			defer fixture.close()
			env := exportFixtureEnv(fixture.cfg)
			if err := setupOrdinaryOrigin(fixture.work, env); err != nil {
				t.Fatal(err)
			}
			input := repositoryInput("settings:\n  general: {danger_zone: {archive_this_repository: true}}\n  rulesets:\n    - id: 7\n      ruleset_name: coverage\n      enforcement_status: active\n      target_branches: {include_default_branch: true}\n      rules: {restrict_code_coverage: {minimum_line_coverage_percentage: 90.25}}\n")
			if err := writeSettingsFile(fixture.work, input); err != nil {
				t.Fatal(err)
			}
			result, err := executeCLI(binary, fixture.work, fixture.root, env, []string{"apply"}, false)
			if err != nil || (result.runErr == nil) != (scenario == "matching") {
				t.Fatalf("error=%v run=%v stderr=%s", err, result.runErr, result.stderr.String())
			}
			if scenario != "matching" && !strings.Contains(result.stderr.String(), "Ruleset") {
				t.Fatalf("missing Rulesets diagnostic: %s", result.stderr.String())
			}
			mu.Lock()
			defer mu.Unlock()
			if writes != 0 || listReads != 1 || detailReads != 1 {
				t.Fatalf("writes=%d list=%d detail=%d", writes, listReads, detailReads)
			}
			data, err := os.ReadFile(filepath.Join(fixture.work, ".github/settings.yml"))
			if err != nil || string(data) != input {
				t.Fatalf("declaration changed: %s %v", data, err)
			}
		})
	}
}
