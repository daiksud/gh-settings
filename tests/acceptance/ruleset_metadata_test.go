//go:build acceptance

package acceptance

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"

	"go.yaml.in/yaml/v4"
)

func TestMetadataRulesExportValidatePlanApplyReadbackAndRepeat(t *testing.T) {
	binary := acceptanceBinary(t)
	for _, test := range []struct{ kind, control, target string }{
		{"commit_message_pattern", "restrict_commit_message", "branch"},
		{"commit_author_email_pattern", "restrict_commit_author_email", "branch"},
		{"committer_email_pattern", "restrict_committer_email", "branch"},
		{"branch_name_pattern", "restrict_branch_names", "branch"},
		{"tag_name_pattern", "restrict_tag_names", "tag"},
	} {
		t.Run(test.kind, func(t *testing.T) {
			policy := map[string]any{"id": 7, "name": "metadata", "source_type": "Repository", "source": "owner/current", "target": test.target, "enforcement": "active", "conditions": map[string]any{"ref_name": map[string]any{"include": []any{"~ALL"}, "exclude": []any{}}}, "bypass_actors": []any{}, "rules": []any{map[string]any{"type": "deletion"}, map[string]any{"type": test.kind, "parameters": map[string]any{"operator": "regex", "pattern": "^before$", "name": "exact description", "negate": true}}}}
			var mu sync.Mutex
			writes, readbacks := 0, 0
			fixture, err := startHTTPFixture(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				w.Header().Set("Content-Type", "application/json")
				if servePRAccessFixtureRead(w, r) || serveIssueCreationPolicyFixtureRead(w, r) || serveSponsorshipsFixtureRead(w, r) {
					return
				}
				if isMergeMethodsFixtureRead(r) {
					fmt.Fprint(w, mergeMethodsFixtureResponse)
					return
				}
				if r.Method == "GET" {
					switch r.URL.Path {
					case "/repos/owner/current":
						fmt.Fprint(w, planResponse)
					case "/repos/owner/current/topics":
						fmt.Fprint(w, `{"names":[]}`)
					case "/repos/owner/current/labels":
						fmt.Fprint(w, `[]`)
					case "/repos/owner/current/rulesets":
						fmt.Fprint(w, `[{"id":7,"source_type":"Repository","source":"owner/current"}]`)
					case "/repos/owner/current/rulesets/7":
						if writes > 0 {
							readbacks++
						}
						json.NewEncoder(w).Encode(policy)
					default:
						t.Error("unexpected GET", r.URL)
						w.WriteHeader(404)
					}
					return
				}
				if r.Method != "PUT" || r.URL.Path != "/repos/owner/current/rulesets/7" {
					t.Error("unexpected mutation", r.Method, r.URL)
					w.WriteHeader(400)
					return
				}
				writes++
				var payload map[string]any
				if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
					t.Error(err)
				}
				expected := map[string]any{"name": "metadata", "target": test.target, "enforcement": "active", "conditions": map[string]any{"ref_name": map[string]any{"include": []any{"~ALL"}, "exclude": []any{}}}, "bypass_actors": []any{}, "rules": []any{map[string]any{"type": test.kind, "parameters": map[string]any{"operator": "regex", "pattern": "^after$", "name": "exact description", "negate": true}}, map[string]any{"type": "deletion"}}}
				// Canonical rule order is by type; each expected rule has independent literal parameters.
				if test.kind == "tag_name_pattern" {
					expected["rules"] = []any{map[string]any{"type": "deletion"}, map[string]any{"type": test.kind, "parameters": map[string]any{"operator": "regex", "pattern": "^after$", "name": "exact description", "negate": true}}}
				}
				if !reflect.DeepEqual(payload, expected) {
					t.Errorf("whole-object payload=%v want=%v", payload, expected)
				}
				payload["id"], payload["source_type"], payload["source"] = 7, "Repository", "owner/current"
				policy = payload
				json.NewEncoder(w).Encode(policy)
			}, false)
			if err != nil {
				t.Fatal(err)
			}
			defer fixture.close()
			env := exportFixtureEnv(fixture.cfg)
			if err := setupOrdinaryOrigin(fixture.work, env); err != nil {
				t.Fatal(err)
			}
			run := func(args ...string) string {
				t.Helper()
				out, err := executeCLI(binary, fixture.work, fixture.root, env, args, false)
				if err != nil || out.runErr != nil {
					t.Fatalf("%v: %v %v %s", args, err, out.runErr, out.stderr.String())
				}
				return out.stdout.String()
			}
			run("export", "--rulesets")
			run("validate")
			path := filepath.Join(fixture.work, ".github/settings.yml")
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(data), test.control+":") || !strings.Contains(string(data), "must_not_match_given_regex_pattern") {
				t.Fatal("UI export lost restriction", string(data))
			}
			if out := run("plan"); out != "No differences for managed repository settings.\n" {
				t.Fatal(out)
			}
			var doc map[string]any
			if err := yaml.Unmarshal(data, &doc); err != nil {
				t.Fatal(err)
			}
			entries := doc["repository"].(map[string]any)["settings"].(map[string]any)["rulesets"].([]any)
			focused := map[string]any{"version": "daiksud/gh-settings@v0", "repository": map[string]any{"settings": map[string]any{"rulesets": entries}}}
			encoded, _ := yaml.Marshal(focused)
			if err := writeSettingsFile(fixture.work, string(encoded)); err != nil {
				t.Fatal(err)
			}
			run("apply")
			mu.Lock()
			n := writes
			mu.Unlock()
			if n != 0 {
				t.Fatal("unchanged export wrote", n)
			}
			entries[0].(map[string]any)["rules"].(map[string]any)[test.control].(map[string]any)["pattern"] = "^after$"
			encoded, _ = yaml.Marshal(focused)
			if err := writeSettingsFile(fixture.work, string(encoded)); err != nil {
				t.Fatal(err)
			}
			run("validate")
			if out := run("plan"); !strings.Contains(out, `"pattern":"^after$"`) {
				t.Fatal(out)
			}
			run("apply")
			mu.Lock()
			n, reads := writes, readbacks
			mu.Unlock()
			if n != 1 || reads != 1 {
				t.Fatal("missing independent readback", n, reads)
			}
			if out := run("plan"); out != "No differences for managed repository settings.\n" {
				t.Fatal(out)
			}
			run("apply")
			mu.Lock()
			n = writes
			mu.Unlock()
			if n != 1 {
				t.Fatal("repeat wrote", n)
			}
		})
	}
}
