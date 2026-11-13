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

const coverageRulesetInput = `version: daiksud/gh-settings@v0
repository:
  settings:
    rulesets:
      - id: 7
        ruleset_name: coverage
        enforcement_status: active
        target_branches: {include_default_branch: true}
        rules:
          restrict_deletions: true
          restrict_code_coverage:
            minimum_line_coverage_percentage: 90.25
`

const coverageRulesetResponse = `{"id":7,"name":"coverage","source_type":"Repository","source":"owner/current","target":"branch","enforcement":"active","conditions":{"ref_name":{"include":["~DEFAULT_BRANCH"],"exclude":[]}},"bypass_actors":[],"rules":[{"type":"code_coverage","parameters":{"minimum_coverage":80.5,"max_coverage_drop":2.25}},{"type":"deletion"}]}`

func TestRulesetCoverageBinary(t *testing.T) {
	binary := acceptanceBinary(t)
	var mu sync.Mutex
	var state map[string]any
	json.Unmarshal([]byte(coverageRulesetResponse), &state)
	writes, details, requests := 0, 0, 0
	malformed := false
	fixture, err := startHTTPFixture(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		requests++
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
				fmt.Fprint(w, "[]")
			case "/repos/owner/current/rulesets":
				if state == nil {
					fmt.Fprint(w, "[]")
				} else {
					fmt.Fprint(w, `[{"id":7,"source_type":"Repository","source":"owner/current"}]`)
				}
			case "/repos/owner/current/rulesets/7":
				details++
				data, _ := json.Marshal(state)
				if malformed {
					data = []byte(strings.Replace(string(data), `"max_coverage_drop":2.25,`, "", 1))
				}
				fmt.Fprint(w, string(data))
			default:
				t.Error("unexpected read", r.URL)
				w.WriteHeader(400)
			}
			return
		}
		if r.URL.Path != "/repos/owner/current/rulesets/7" {
			t.Error("unexpected mutation", r.Method, r.URL)
			w.WriteHeader(400)
			return
		}
		writes++
		if r.Method == "DELETE" {
			state = nil
			w.WriteHeader(204)
			return
		}
		if r.Method != "PUT" {
			t.Error(r.Method)
			w.WriteHeader(400)
			return
		}
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Error(err)
		}
		state = payload
		if _, present := state["id"]; present {
			t.Error("ID sent in body")
		}
		state["id"], state["source"], state["source_type"] = 7, "owner/current", "Repository"
		json.NewEncoder(w).Encode(state)
	}, false, "/labels", "/rulesets", "/rulesets/7", "/topics")
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
		result, err := executeCLI(binary, fixture.work, fixture.root, env, args, false)
		if err != nil || result.runErr != nil {
			t.Fatalf("%v: %v %v %s", args, err, result.runErr, result.stderr.String())
		}
		return result.stdout.String()
	}
	run("export")
	path := filepath.Join(fixture.work, ".github/settings.yml")
	exported, err := os.ReadFile(path)
	if err != nil || !strings.Contains(string(exported), "minimum_line_coverage_percentage: 80.5") || !strings.Contains(string(exported), "maximum_line_coverage_drop: 2.25") {
		t.Fatal(string(exported), err)
	}
	run("validate")
	mu.Lock()
	malformed = true
	mu.Unlock()
	result, err := executeCLI(binary, fixture.work, fixture.root, env, []string{"export"}, false)
	data, readErr := os.ReadFile(path)
	if err != nil || result.runErr != nil || readErr != nil || !strings.Contains(string(data), "minimum_line_coverage_percentage: 80.5") || !strings.Contains(result.stderr.String(), "repository.settings.rulesets") {
		t.Fatal("unavailable observation lost previous policy", string(data), result, err, readErr)
	}
	mu.Lock()
	malformed = false
	mu.Unlock()
	if err := writeSettingsFile(fixture.work, coverageRulesetInput); err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	beforeRequests := requests
	mu.Unlock()
	run("validate")
	mu.Lock()
	if requests != beforeRequests {
		t.Error("validation used API")
	}
	mu.Unlock()
	if output := run("plan"); !strings.Contains(output, "minimum_line_coverage_percentage") || !strings.Contains(output, `"maximum_line_coverage_drop":0`) {
		t.Fatal(output)
	}
	mu.Lock()
	beforeDetails := details
	mu.Unlock()
	run("apply")
	mu.Lock()
	if writes != 1 || details-beforeDetails != 3 {
		t.Error("missing independent verification", writes, details-beforeDetails)
	}
	rules := state["rules"].([]any)
	if !reflect.DeepEqual(rules[0], map[string]any{"type": "code_coverage", "parameters": map[string]any{"minimum_coverage": 90.25, "max_coverage_drop": float64(0)}}) {
		t.Error(rules)
	}
	mu.Unlock()
	run("apply")
	if output := run("plan"); output != "No differences for managed repository settings.\n" {
		t.Fatal(output)
	}
	if err := checkSettingsUnchanged(fixture.work, coverageRulesetInput); err != nil {
		t.Fatal(err)
	}
	// Group omission removes the selected rule under whole-object ownership.
	without := strings.Split(coverageRulesetInput, "          restrict_code_coverage:")[0]
	if err := writeSettingsFile(fixture.work, without); err != nil {
		t.Fatal(err)
	}
	run("apply")
	mu.Lock()
	if writes != 2 || len(state["rules"].([]any)) != 1 {
		t.Error("omitted group did not remove coverage", state, writes)
	}
	mu.Unlock()
	// Re-add coverage, then explicitly prune the complete supported object.
	if err := writeSettingsFile(fixture.work, coverageRulesetInput); err != nil {
		t.Fatal(err)
	}
	run("apply")
	if err := writeSettingsFile(fixture.work, "version: daiksud/gh-settings@v0\nrepository: {settings: {rulesets: []}}\n"); err != nil {
		t.Fatal(err)
	}
	run("apply")
	mu.Lock()
	if writes != 3 || state == nil {
		t.Error("empty ordinary selection deleted", writes)
	}
	mu.Unlock()
	run("apply", "--prune", "rulesets")
	mu.Lock()
	defer mu.Unlock()
	if writes != 4 || state != nil {
		t.Fatal("explicit prune failed", writes, state)
	}
}

func TestRulesetCoverageCreationIDRequiresReadback(t *testing.T) {
	for _, verified := range []bool{false, true} {
		t.Run(fmt.Sprint(verified), func(t *testing.T) {
			var mu sync.Mutex
			var created map[string]any
			posts, reads := 0, 0
			fixture, err := startHTTPFixture(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				w.Header().Set("Content-Type", "application/json")
				switch {
				case r.Method == "GET" && r.URL.Path == "/repos/owner/current/rulesets":
					if created == nil {
						fmt.Fprint(w, "[]")
					} else {
						fmt.Fprint(w, `[{"id":17,"source_type":"Repository","source":"owner/current"}]`)
					}
				case r.Method == "POST" && r.URL.Path == "/repos/owner/current/rulesets":
					posts++
					if err := json.NewDecoder(r.Body).Decode(&created); err != nil {
						t.Error(err)
					}
					created["id"], created["source_type"], created["source"] = 17, "Repository", "owner/current"
					w.WriteHeader(201)
					json.NewEncoder(w).Encode(created)
				case r.Method == "GET" && r.URL.Path == "/repos/owner/current/rulesets/17":
					reads++
					if !verified {
						w.WriteHeader(403)
						fmt.Fprint(w, "{}")
					} else {
						json.NewEncoder(w).Encode(created)
					}
				default:
					t.Error(r.Method, r.URL)
					w.WriteHeader(400)
				}
			}, false, "/rulesets", "/rulesets/17")
			if err != nil {
				t.Fatal(err)
			}
			defer fixture.close()
			env := labelIDFixtureEnv(fixture.cfg, false)
			if err := setupOrdinaryOrigin(fixture.work, env); err != nil {
				t.Fatal(err)
			}
			input := strings.Replace(coverageRulesetInput, "      - id: 7\n        ruleset_name: coverage", "      - ruleset_name: coverage", 1)
			if err := writeSettingsFile(fixture.work, input); err != nil {
				t.Fatal(err)
			}
			result, err := executeCLI(acceptanceBinary(t), fixture.work, fixture.root, env, []string{"apply"}, false)
			data, readErr := os.ReadFile(filepath.Join(fixture.work, ".github/settings.yml"))
			if err != nil || readErr != nil || (result.runErr == nil) != verified || strings.Contains(string(data), "id: 17") != verified {
				t.Fatal(string(data), result, err, readErr)
			}
			if !verified && (string(data) != input || result.stdout.Len() != 0 || !strings.Contains(result.stderr.String(), "id 17") || !strings.Contains(result.stderr.String(), "may already have applied")) {
				t.Fatal("failed confirmation lost recovery or changed file", result)
			}
			mu.Lock()
			if posts != 1 || reads != 1 {
				t.Error(posts, reads)
			}
			mu.Unlock()
			if verified {
				result, err = executeCLI(acceptanceBinary(t), fixture.work, fixture.root, env, []string{"apply"}, false)
				mu.Lock()
				defer mu.Unlock()
				if err != nil || result.runErr != nil || posts != 1 {
					t.Fatal("created identity did not reconcile", result, posts, err)
				}
			}
		})
	}
}

func TestRulesetCoverageLocalValidation(t *testing.T) {
	binary := acceptanceBinary(t)
	fixture, err := startHTTPFixture(func(w http.ResponseWriter, r *http.Request) {
		t.Error("invalid input contacted API")
		w.WriteHeader(500)
	}, false, "/rulesets")
	if err != nil {
		t.Fatal(err)
	}
	defer fixture.close()
	env := exportFixtureEnv(fixture.cfg)
	if err := setupOrdinaryOrigin(fixture.work, env); err != nil {
		t.Fatal(err)
	}
	for _, input := range []string{
		strings.Replace(coverageRulesetInput, "90.25", "-0.01", 1),
		strings.Replace(coverageRulesetInput, "90.25", "100.01", 1),
		strings.Replace(coverageRulesetInput, "target_branches: {include_default_branch: true}", "target_tags: {include_all_tags: true}", 1),
	} {
		if err := writeSettingsFile(fixture.work, input); err != nil {
			t.Fatal(err)
		}
		for _, command := range []string{"validate", "plan", "apply"} {
			result, err := executeCLI(binary, fixture.work, fixture.root, env, []string{command}, false)
			if err != nil || result.runErr == nil {
				t.Fatal("accepted invalid coverage", command, input, result, err)
			}
		}
	}
}
