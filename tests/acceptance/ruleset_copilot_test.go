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

func TestRulesetCopilotExportValidatePlanApplyReadback(t *testing.T) {
	binary := acceptanceBinary(t)
	const baseline = `{"id":7,"name":"copilot","source_type":"Repository","source":"owner/current","target":"branch","enforcement":"active","conditions":{"ref_name":{"include":["~DEFAULT_BRANCH"],"exclude":[]}},"bypass_actors":[],"rules":[{"type":"copilot_code_review","parameters":{"review_draft_pull_requests":false,"review_on_push":true}},{"type":"deletion"}]}`
	var state map[string]any
	json.Unmarshal([]byte(baseline), &state)
	var mu sync.Mutex
	writes, details := 0, 0
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
				details++
				json.NewEncoder(w).Encode(state)
			default:
				t.Error("unexpected read", r.URL)
				w.WriteHeader(404)
			}
			return
		}
		if r.Method != "PUT" || r.URL.Path != "/repos/owner/current/rulesets/7" {
			t.Error("unexpected write", r.Method, r.URL)
			w.WriteHeader(400)
			return
		}
		writes++
		var got, want map[string]any
		json.NewDecoder(r.Body).Decode(&got)
		json.Unmarshal([]byte(strings.Replace(baseline, `"review_draft_pull_requests":false`, `"review_draft_pull_requests":true`, 1)), &want)
		delete(want, "id")
		delete(want, "source")
		delete(want, "source_type")
		if !reflect.DeepEqual(got, want) {
			t.Error("protection payload", got, want)
		}
		got["id"], got["source"], got["source_type"] = 7, "owner/current", "Repository"
		state = got
		json.NewEncoder(w).Encode(state)
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
	exported, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(exported), "automatically_request_copilot_code_review:") || !strings.Contains(string(exported), "review_new_pushes: true") || !strings.Contains(string(exported), "review_draft_pull_requests: false") {
		t.Fatal(string(exported))
	}
	if out := run("plan"); out != "No differences for managed repository settings.\n" {
		t.Fatal(out)
	}
	// Focus only the requested family; the export above independently proves acquisition.
	desired := `version: daiksud/gh-settings@v0
repository:
  settings:
    rulesets:
      - id: 7
        ruleset_name: copilot
        enforcement_status: active
        target_branches: {include_default_branch: true}
        rules:
          restrict_deletions: true
          automatically_request_copilot_code_review:
            review_new_pushes: true
            review_draft_pull_requests: true
`
	if err := writeSettingsFile(fixture.work, desired); err != nil {
		t.Fatal(err)
	}
	run("validate")
	if out := run("plan"); !strings.Contains(out, "automatically_request_copilot_code_review") {
		t.Fatal(out)
	}
	mu.Lock()
	before := details
	mu.Unlock()
	run("apply")
	mu.Lock()
	if writes != 1 || details-before != 2 {
		t.Error("apply requires inventory detail plus independent readback", writes, details-before)
	}
	mu.Unlock()
	if out := run("plan"); out != "No differences for managed repository settings.\n" {
		t.Fatal(out)
	}
	run("apply")
	mu.Lock()
	defer mu.Unlock()
	if writes != 1 {
		t.Fatal("repeat mutated", writes)
	}
}
