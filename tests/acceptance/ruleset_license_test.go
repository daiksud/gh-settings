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
)

func TestLicenseRulesetProcessRoundTrip(t *testing.T) {
	binary := acceptanceBinary(t)
	var mu sync.Mutex
	const absent = `{"id":7,"name":"license","source_type":"Repository","source":"owner/current","target":"branch","enforcement":"active","bypass_actors":[],"conditions":{"ref_name":{"include":["~DEFAULT_BRANCH"],"exclude":[]}},"rules":[]}`
	var state map[string]any
	if err := json.Unmarshal([]byte(absent), &state); err != nil {
		t.Fatal(err)
	}
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
		if r.Method == "PUT" && r.URL.Path == "/repos/owner/current/rulesets/7" {
			var payload map[string]any
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Error(err)
			}
			expectedRules := []any{map[string]any{"type": "license_compliance_scanning"}}
			if writes == 1 {
				expectedRules = []any{}
			}
			expected := map[string]any{"name": "license", "target": "branch", "enforcement": "active", "bypass_actors": []any{}, "conditions": map[string]any{"ref_name": map[string]any{"include": []any{"~DEFAULT_BRANCH"}, "exclude": []any{}}}, "rules": expectedRules}
			if !reflect.DeepEqual(payload, expected) {
				t.Error("unexpected complete payload", payload, expected)
			}
			writes++
			payload["id"], payload["source"], payload["source_type"] = 7, "owner/current", "Repository"
			state = payload
			json.NewEncoder(w).Encode(state)
			return
		}
		if r.Method != "GET" {
			t.Error("unexpected mutation", r.Method, r.URL)
			w.WriteHeader(400)
			return
		}
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
			readbacks++
			json.NewEncoder(w).Encode(state)
		default:
			t.Error("unexpected path", r.URL)
			w.WriteHeader(404)
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
	run := func(args ...string) string {
		t.Helper()
		r, err := executeCLI(binary, fixture.work, fixture.root, env, args, false)
		if err != nil || r.runErr != nil {
			t.Fatalf("%v: %v %v %s", args, err, r.runErr, r.stderr.String())
		}
		return r.stdout.String()
	}
	run("export", "--rulesets")
	path := filepath.Join(fixture.work, ".github/settings.yml")
	exported, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(exported), "require_license_compliance_results_before_merging: false") {
		t.Fatal(string(exported))
	}
	const config = `version: daiksud/gh-settings@v0
repository:
  settings:
    rulesets:
      - id: 7
        ruleset_name: license
        enforcement_status: active
        target_branches: {include_default_branch: true}
        rules: {require_license_compliance_results_before_merging: true}
`
	for _, enabled := range []bool{true, false} {
		input := config
		if !enabled {
			input = strings.Replace(config, "before_merging: true", "before_merging: false", 1)
		}
		if err := writeSettingsFile(fixture.work, input); err != nil {
			t.Fatal(err)
		}
		run("validate")
		if out := run("plan"); !strings.Contains(out, "require_license_compliance_results_before_merging") {
			t.Fatal(out)
		}
		mu.Lock()
		beforeReads := readbacks
		beforeWrites := writes
		mu.Unlock()
		run("apply")
		mu.Lock()
		if writes != beforeWrites+1 || readbacks != beforeReads+2 {
			t.Error("missing fresh acquisition/readback", writes, readbacks, beforeReads)
		}
		mu.Unlock()
		if out := run("plan"); out != "No differences for managed repository settings.\n" {
			t.Fatal(out)
		}
		run("apply")
		mu.Lock()
		if writes != beforeWrites+1 {
			t.Error("repeat mutated", writes)
		}
		mu.Unlock()
		if err := checkSettingsUnchanged(fixture.work, input); err != nil {
			t.Fatal(err)
		}
	}
}
