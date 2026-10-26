//go:build acceptance

package acceptance

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"go.yaml.in/yaml/v4"
)

func TestOptionalDismissalExportThenPlanApplyFailSafely(t *testing.T) {
	binary := acceptanceBinary(t)
	const restriction = `"dismissal_restriction":{"enabled":true,"allowed_actors":[{"type":"Team","id":21}]},`
	body := strings.Replace(acceptanceReviewRuleset, restriction, "", 1)
	var mu sync.Mutex
	writes := 0
	fixture, err := startHTTPFixture(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if servePRAccessFixtureRead(w, r) || serveIssueCreationPolicyFixtureRead(w, r) || serveSponsorshipsFixtureRead(w, r) {
			return
		}
		if isMergeMethodsFixtureRead(r) {
			fmt.Fprint(w, mergeMethodsFixtureResponse)
			return
		}
		if r.Method != http.MethodGet {
			mu.Lock()
			writes++
			mu.Unlock()
			t.Errorf("unexpected mutation: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusBadRequest)
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
			fmt.Fprint(w, body)
		default:
			t.Errorf("unexpected read: %s", r.URL)
			w.WriteHeader(http.StatusNotFound)
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
	result, err := executeCLI(binary, fixture.work, fixture.root, env, []string{"export", "--rulesets"}, false)
	if err != nil || result.runErr != nil {
		t.Fatalf("optional absence prevented export: %v %v %s", err, result.runErr, result.stderr.String())
	}
	data, err := os.ReadFile(filepath.Join(fixture.work, ".github/settings.yml"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "dismissal_restriction") {
		t.Fatalf("export invented a protection value: %s", data)
	}
	var document map[string]any
	if err := yaml.Unmarshal(data, &document); err != nil {
		t.Fatal(err)
	}
	members := document["settings"].(map[string]any)["rulesets"].([]any)
	if len(members) != 1 || members[0].(map[string]any)["id"] != 7 {
		t.Fatal("export lost observed identity", members)
	}
	// Isolate the selected Ruleset from unrelated direct scalar reapplication.
	selected, err := yaml.Marshal(map[string]any{"schemaVersion": 1, "settings": map[string]any{"rulesets": members}})
	if err != nil {
		t.Fatal(err)
	}
	if err := writeSettingsFile(fixture.work, string(selected)); err != nil {
		t.Fatal(err)
	}
	result, err = executeCLI(binary, fixture.work, fixture.root, env, []string{"validate"}, false)
	if err != nil || result.runErr != nil {
		t.Fatalf("valid export rejected locally: %v %v %s", err, result.runErr, result.stderr.String())
	}
	reportPath := filepath.Join(fixture.root, "unknown-plan.json")
	for _, args := range [][]string{{"plan"}, {"apply"}, {"action", "plan", reportPath}} {
		result, err := executeCLI(binary, fixture.work, fixture.root, env, args, false)
		if err != nil || result.runErr == nil || result.stdout.Len() != 0 || !strings.Contains(result.stderr.String(), "dismissal_restriction") {
			t.Fatalf("unknown state became success/diff for %v: %v %v %s %s", args, err, result.runErr, result.stdout.String(), result.stderr.String())
		}
	}
	data, err = os.ReadFile(reportPath)
	if err != nil {
		t.Fatal(err)
	}
	var report map[string]any
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatal(err)
	}
	if report["outcome"] != "read-failure" || strings.Contains(string(data), "dismissal_restriction") {
		t.Fatalf("report fabricated values or exposed raw diagnostics: %s", data)
	}
	mu.Lock()
	defer mu.Unlock()
	if writes != 0 {
		t.Fatalf("guard allowed %d mutations", writes)
	}
	if err := checkSettingsUnchanged(fixture.work, string(selected)); err != nil {
		t.Fatal(err)
	}
}
