//go:build acceptance

package acceptance

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"sync"
	"testing"

	"go.yaml.in/yaml/v4"
)

func TestPushRulesetExportValidatePlanApplyAndIncompleteSafety(t *testing.T) {
	binary := acceptanceBinary(t)
	data, err := os.ReadFile("../fixtures/ruleset-push.json")
	if err != nil {
		t.Fatal(err)
	}
	var state, baseline map[string]any
	if err = json.Unmarshal(data, &state); err != nil {
		t.Fatal(err)
	}
	json.Unmarshal(data, &baseline)
	var mu sync.Mutex
	writes := 0
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
				fmt.Fprint(w, `[{"id":26024001,"source_type":"Repository","source":"owner/current"}]`)
			case "/repos/owner/current/rulesets/26024001":
				json.NewEncoder(w).Encode(state)
			default:
				t.Error("unexpected GET", r.URL)
				w.WriteHeader(404)
			}
			return
		}
		if r.Method != "PUT" || r.URL.Path != "/repos/owner/current/rulesets/26024001" {
			t.Error("unexpected mutation", r.Method, r.URL)
			w.WriteHeader(400)
			return
		}
		writes++
		var payload, expected map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Error(err)
		}
		copyData, _ := json.Marshal(baseline)
		json.Unmarshal(copyData, &expected)
		delete(expected, "id")
		delete(expected, "source")
		delete(expected, "source_type")
		delete(expected, "conditions")
		acceptanceRuleParams(t, expected, "max_file_size")["max_file_size"] = float64(10)
		for _, value := range []map[string]any{payload, expected} {
			slices.SortFunc(value["rules"].([]any), func(a, b any) int {
				return strings.Compare(a.(map[string]any)["type"].(string), b.(map[string]any)["type"].(string))
			})
		}
		if !reflect.DeepEqual(payload, expected) {
			t.Error("whole-object protection changed", payload, expected)
		}
		payload["id"], payload["source"], payload["source_type"] = float64(26024001), "owner/current", "Repository"
		state = payload
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
			t.Fatalf("%v: %v %s", args, err, out.stderr.String())
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
	if out := run("plan"); out != "No differences for managed repository settings.\n" {
		t.Fatal(out)
	}
	var doc map[string]any
	if err := yaml.Unmarshal(exported, &doc); err != nil {
		t.Fatal(err)
	}
	actualPolicies := doc["repository"].(map[string]any)["settings"].(map[string]any)["rulesets"].([]any)

	policy := map[string]any{
		"id": 26024001, "ruleset_name": "push", "enforcement_status": "active", "target_pushes": true, "bypass_list": []any{},
		"rules": map[string]any{
			"restrict_creations": false, "restrict_deletions": false, "block_force_pushes": false, "require_linear_history": false, "require_signed_commits": false,
			"restrict_file_extensions":  map[string]any{"restricted_file_extensions": []any{".exe"}},
			"restrict_file_paths":       map[string]any{"restricted_file_paths": []any{"secrets/**"}, "allowed_exceptions": []any{"secrets/example"}},
			"restrict_file_path_length": map[string]any{"maximum_file_path_length": 240},
			"restrict_file_size":        map[string]any{"maximum_file_size": 20, "allowed_exceptions": []any{}},
		},
	}

	policies := []any{policy}
	if !reflect.DeepEqual(actualPolicies, policies) {
		t.Fatalf("export=%v want=%v", actualPolicies, policies)
	}
	// Focus the independent UI declaration on Rulesets; ordinary scalar apply is separate.
	document := map[string]any{"version": "daiksud/gh-settings@v0", "repository": map[string]any{"settings": map[string]any{"rulesets": policies}}}
	focused, err := yaml.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	if err := writeSettingsFile(fixture.work, string(focused)); err != nil {
		t.Fatal(err)
	}
	run("apply")
	mu.Lock()
	count := writes
	mu.Unlock()
	if count != 0 {
		t.Fatal("unchanged Ruleset export wrote", count)
	}
	policy["rules"].(map[string]any)["restrict_file_size"].(map[string]any)["maximum_file_size"] = 10
	edited, err := yaml.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	if err := writeSettingsFile(fixture.work, string(edited)); err != nil {
		t.Fatal(err)
	}
	if out := run("plan"); !strings.Contains(out, `"maximum_file_size":10`) {
		t.Fatal("threshold change absent", out)
	}
	run("apply")
	if out := run("plan"); out != "No differences for managed repository settings.\n" {
		t.Fatal(out)
	}
	run("apply")
	mu.Lock()
	count = writes
	mu.Unlock()
	if count != 1 {
		t.Fatal("repeat apply wrote", count)
	}
	unchanged, err := os.ReadFile(path)
	if err != nil || !reflect.DeepEqual(edited, unchanged) {
		t.Fatal("ID-selected update changed declaration", err)
	}
	// Losing acquired exception information or receiving a future rule must fail
	// before a write and preserve the existing export destination bytes.
	complete, err := json.Marshal(state)
	if err != nil {
		t.Fatal(err)
	}
	for _, corrupt := range []func(){
		func() { delete(acceptanceRuleParams(t, state, "file_path_restriction"), "ignored_file_paths") },
		func() {
			state["rules"] = append(state["rules"].([]any), map[string]any{"type": "future_push_protection"})
		},
	} {
		mu.Lock()
		if err := json.Unmarshal(complete, &state); err != nil {
			t.Fatal(err)
		}
		corrupt()
		mu.Unlock()
		for _, command := range [][]string{{"export", "--rulesets"}, {"plan"}, {"apply"}} {
			out, err := executeCLI(binary, fixture.work, fixture.root, env, command, false)
			if err != nil || out.runErr == nil {
				t.Fatal("incomplete push policy accepted", command, err)
			}
		}
		preserved, err := os.ReadFile(path)
		if err != nil || !reflect.DeepEqual(edited, preserved) {
			t.Fatal("failed export replaced destination", err)
		}
		mu.Lock()
		if writes != 1 {
			t.Fatal("incomplete acquisition wrote", writes)
		}
		mu.Unlock()
	}

}
