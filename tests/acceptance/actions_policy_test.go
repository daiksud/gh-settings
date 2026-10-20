//go:build acceptance

package acceptance

import (
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
)

func actionsPolicyInput(body string) string {
	return "schemaVersion: 1\nsettings:\n  actions: " + body + "\n"
}

func TestActionsPolicyExportTransitionAndReconciliation(t *testing.T) {
	binary := acceptanceBinary(t)
	var mu sync.Mutex
	base := map[string]any{"enabled": false, "allowed_actions": "all", "sha_pinning_required": true}
	selected := map[string]any{"github_owned_allowed": true, "verified_allowed": true, "patterns_allowed": []any{"remembered/*"}}
	var operations []string
	var writes []map[string]any
	failSelected := false
	corruptRead := false
	fixture, err := startHTTPFixture(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		if strings.HasPrefix(r.URL.Path, "/repos/owner/current/actions/permissions") {
			operations = append(operations, r.Method+" "+r.URL.Path)
			isSelected := r.URL.Path == "/repos/owner/current/actions/permissions/selected-actions"
			if r.URL.Path != "/repos/owner/current/actions/permissions" && !isSelected {
				t.Error("unexpected Actions path", r.URL)
				w.WriteHeader(400)
				return
			}
			if isSelected && base["allowed_actions"] != "selected" {
				t.Error("inactive selected request")
				w.WriteHeader(409)
				return
			}
			if isSelected && failSelected {
				w.WriteHeader(403)
				fmt.Fprint(w, `{"message":"organization policy denies this operation"}`)
				return
			}
			if corruptRead && r.Method == "GET" {
				fmt.Fprint(w, `{"enabled":false,"allowed_actions":"all"}`)
				return
			}
			target := base
			if isSelected {
				target = selected
			}
			switch r.Method {
			case "GET":
				json.NewEncoder(w).Encode(target)
			case "PUT":
				var payload map[string]any
				if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
					t.Error(err)
				}
				writes = append(writes, payload)
				for key, value := range payload {
					target[key] = value
				}
				w.WriteHeader(204)
			default:
				t.Error("unexpected Actions method", r.Method)
				w.WriteHeader(400)
			}
			return
		}
		if servePRAccessFixtureRead(w, r) || serveIssueCreationPolicyFixtureRead(w, r) {
			return
		}
		if isMergeMethodsFixtureRead(r) {
			fmt.Fprint(w, mergeMethodsFixtureResponse)
			return
		}
		if r.Method != "GET" {
			t.Error("unexpected write", r.Method, r.URL)
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
		default:
			t.Error("unexpected read", r.URL)
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
	run := func(args ...string) string {
		t.Helper()
		r, err := executeCLI(binary, fixture.work, fixture.root, env, args, false)
		if err != nil || r.runErr != nil || r.stderr.Len() != 0 {
			t.Fatal(args, err, r.stderr.String())
		}
		return r.stdout.String()
	}
	run("export")
	mu.Lock()
	calls := len(operations)
	mu.Unlock()
	if calls != 0 {
		t.Fatal("ordinary export requires additional Administration read")
	}
	run("export", "--actions")
	run("validate")
	data, err := os.ReadFile(filepath.Join(fixture.work, ".github/settings.yml"))
	if err != nil || !strings.Contains(string(data), "enabled: false") || strings.Contains(string(data), "selected_actions:") {
		t.Fatal(string(data), err)
	}
	if got := run("plan"); got != "No differences for managed repository settings.\n" {
		t.Fatal(got)
	}
	full := actionsPolicyInput(`{allowed_actions: selected, selected_actions: {github_owned_allowed: false, verified_allowed: false, patterns_allowed: ['org/*', '!org/bad@*', 'org/*']}}`)
	if err := writeSettingsFile(fixture.work, full); err != nil {
		t.Fatal(err)
	}
	plan := run("plan")
	if !strings.Contains(plan, `settings.actions.allowed_actions: "all" → "selected"`) || strings.Count(plan, "unavailable while policy is not selected") != 3 {
		t.Fatal(plan)
	}
	report := filepath.Join(fixture.root, "actions-plan.json")
	run("action", "plan", report)
	data, _ = os.ReadFile(report)
	var native map[string]any
	if err := json.Unmarshal(data, &native); err != nil {
		t.Fatal(err)
	}
	if native["managed"] != float64(4) || native["outcome"] != "changed" {
		t.Fatal(string(data))
	}
	entries := native["actions"].([]any)
	if entries[1].(map[string]any)["before"] != nil || entries[1].(map[string]any)["after"] != false {
		t.Fatal(string(data))
	}
	summary := filepath.Join(fixture.root, "actions-summary.md")
	summaryEnv := append(fixtureEnv(fixture.cfg), "SETTINGS_ACTOR=user", "SETTINGS_TRIGGERING_ACTOR=user", "SETTINGS_EVENT=push", "SETTINGS_RUN_ID=123", "SETTINGS_RUN_ATTEMPT=1", "SETTINGS_WORKFLOW_REF=owner/current/.github/workflows/settings.yml@refs/heads/main", "SETTINGS_WORKFLOW_SHA="+strings.Repeat("a", 40), "SETTINGS_CONFIG_SHA="+strings.Repeat("b", 40), "SETTINGS_TARGET=owner/current", "SETTINGS_CLI_VERSION=dev", "SETTINGS_CLI_SHA256="+strings.Repeat("c", 64))
	reads := fixture.reads.Load()
	r, err := executeCLI(binary, fixture.root, fixture.root, summaryEnv, []string{"action", "summary", report, summary}, false)
	if err != nil || r.runErr != nil || fixture.reads.Load() != reads {
		t.Fatal(err, r.stderr.String())
	}
	data, _ = os.ReadFile(summary)
	if !strings.Contains(html.UnescapeString(string(data)), strings.TrimSpace(plan)) {
		t.Fatal(string(data))
	}
	run("apply")
	if got := run("plan"); got != "No differences for managed repository settings.\n" {
		t.Fatal(got)
	}
	run("export", "--actions")
	run("validate")
	if got := run("plan"); got != "No differences for managed repository settings.\n" {
		t.Fatal(got)
	}
	mu.Lock()
	if base["enabled"] != false || len(writes) != 2 || !reflect.DeepEqual(writes[0], map[string]any{"enabled": false, "allowed_actions": "selected"}) || !reflect.DeepEqual(writes[1], map[string]any{"github_owned_allowed": false, "verified_allowed": false, "patterns_allowed": []any{"org/*", "!org/bad@*", "org/*"}}) {
		t.Fatal("inferred/overbroad values", base, writes)
	}
	mu.Unlock()
	// Partial selected updates preserve fresh companions and exact sequence order.
	writeSettingsFile(fixture.work, actionsPolicyInput(`{selected_actions: {patterns_allowed: []}}`))
	run("apply")
	run("apply")
	mu.Lock()
	if selected["github_owned_allowed"] != false || selected["verified_allowed"] != false || !reflect.DeepEqual(selected["patterns_allowed"], []any{}) || len(writes) != 4 {
		t.Fatal(selected, writes)
	}
	mu.Unlock()
	writeSettingsFile(fixture.work, actionsPolicyInput(`{allowed_actions: local_only, sha_pinning_required: false}`))
	run("apply")
	// An incomplete transition fails before any write; errors do not invent defaults.
	writeSettingsFile(fixture.work, actionsPolicyInput(`{allowed_actions: selected}`))
	mu.Lock()
	beforeWrites := len(writes)
	mu.Unlock()
	r, err = executeCLI(binary, fixture.work, fixture.root, env, []string{"apply"}, false)
	if err != nil || r.runErr == nil || !strings.Contains(r.stderr.String(), "all three") {
		t.Fatal(err, r.stderr.String())
	}
	mu.Lock()
	if len(writes) != beforeWrites {
		t.Fatal("incomplete transition wrote")
	}
	mu.Unlock()
	// The prerequisite policy update remains applied if the selected endpoint fails.
	writeSettingsFile(fixture.work, full)
	mu.Lock()
	failSelected = true
	mu.Unlock()
	r, err = executeCLI(binary, fixture.work, fixture.root, env, []string{"apply"}, false)
	if err != nil || r.runErr == nil || !strings.Contains(r.stderr.String(), "may already have applied") {
		t.Fatal(err, r.stderr.String())
	}
	mu.Lock()
	if base["allowed_actions"] != "selected" {
		t.Fatal("failure claimed rollback")
	}
	failSelected = false
	corruptRead = true
	mu.Unlock()
	prior := full
	r, err = executeCLI(binary, fixture.work, fixture.root, env, []string{"export", "--actions"}, false)
	if err != nil || r.runErr == nil {
		t.Fatal("incomplete export succeeded", err, r.stderr.String())
	}
	if err := checkSettingsUnchanged(fixture.work, prior); err != nil {
		t.Fatal(err)
	}
}

func TestActionsInvalidDeclarationsNeverContactAPI(t *testing.T) {
	binary := acceptanceBinary(t)
	fixture, err := startHTTPFixture(func(w http.ResponseWriter, r *http.Request) {
		t.Error("invalid declaration invoked API", r.Method, r.URL)
		w.WriteHeader(400)
	}, false)
	if err != nil {
		t.Fatal(err)
	}
	defer fixture.close()
	env := exportFixtureEnv(fixture.cfg)
	if err := setupOrdinaryOrigin(fixture.work, env); err != nil {
		t.Fatal(err)
	}
	for _, body := range []string{`{enabled: null}`, `{allowed_actions: unknown}`, `{selected_actions: {patterns_allowed: null}}`, `{allowed_actions: all, selected_actions: {verified_allowed: false}}`} {
		writeSettingsFile(fixture.work, actionsPolicyInput(body))
		for _, command := range []string{"validate", "plan", "apply"} {
			r, err := executeCLI(binary, fixture.work, fixture.root, env, []string{command}, false)
			if err != nil || r.runErr == nil {
				t.Fatal(body, command, err, r.stderr.String())
			}
		}
	}
	if fixture.reads.Load() != 0 {
		t.Fatal("invalid inputs contacted API")
	}
}

func TestActionsReadFailuresDiscardPlansAndKeepEarlierWrites(t *testing.T) {
	binary := acceptanceBinary(t)
	for _, test := range []struct {
		name, body string
		status     int
	}{
		{"denied", `{"message":"organization policy denies access"}`, 403},
		{"missing", `{"enabled":false,"allowed_actions":"all"}`, 200},
		{"null", `{"enabled":false,"allowed_actions":"all","sha_pinning_required":null}`, 200},
		{"unknown", `{"enabled":false,"allowed_actions":"unknown","sha_pinning_required":true}`, 200},
		{"redirect", `{}`, 301},
	} {
		t.Run(test.name, func(t *testing.T) {
			var mu sync.Mutex
			writes := 0
			redirects := 0
			fixture, err := startHTTPFixture(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				w.Header().Set("Content-Type", "application/json")
				if r.Method == "PATCH" && r.URL.Path == "/repos/owner/current" {
					writes++
					fmt.Fprint(w, `{}`)
					return
				}
				if r.Method == "GET" && r.URL.Path == "/repos/owner/current" {
					fmt.Fprint(w, planResponse)
					return
				}
				if r.Method == "GET" && r.URL.Path == "/repos/owner/current/actions/permissions" {
					if test.status == 301 {
						w.Header().Set("Location", "/repos/owner/other/actions/permissions")
					}
					w.WriteHeader(test.status)
					fmt.Fprint(w, test.body)
					return
				}
				redirects++
				t.Error("unexpected later or redirected request", r.Method, r.URL)
				w.WriteHeader(400)
			}, false)
			if err != nil {
				t.Fatal(err)
			}
			defer fixture.close()
			env := exportFixtureEnv(fixture.cfg)
			if err := setupOrdinaryOrigin(fixture.work, env); err != nil {
				t.Fatal(err)
			}
			input := "schemaVersion: 1\nsettings:\n  repository: {description: changed}\n  actions: {enabled: false}\n"
			if err := writeSettingsFile(fixture.work, input); err != nil {
				t.Fatal(err)
			}
			report := filepath.Join(fixture.root, "failed-plan.json")
			r, err := executeCLI(binary, fixture.work, fixture.root, env, []string{"action", "plan", report}, false)
			if err != nil || r.runErr == nil {
				t.Fatal("read failure accepted", err, r.stderr.String())
			}
			data, err := os.ReadFile(report)
			if err != nil {
				t.Fatal(err)
			}
			var typed map[string]any
			if err := json.Unmarshal(data, &typed); err != nil {
				t.Fatal(err)
			}
			if typed["outcome"] != "read-failure" || typed["managed"] != float64(0) || len(typed["changes"].([]any)) != 0 || typed["actions"] != nil {
				t.Fatal("partial comparison escaped", string(data))
			}
			mu.Lock()
			if writes != 0 {
				t.Fatal("plan wrote")
			}
			mu.Unlock()
			r, err = executeCLI(binary, fixture.work, fixture.root, env, []string{"apply"}, false)
			if err != nil || r.runErr == nil || !strings.Contains(r.stderr.String(), "earlier resource writes remain applied") {
				t.Fatal(err, r.stderr.String())
			}
			mu.Lock()
			if writes != 1 || redirects != 0 {
				t.Fatal(writes, redirects)
			}
			mu.Unlock()
			if err := checkSettingsUnchanged(fixture.work, input); err != nil {
				t.Fatal(err)
			}
		})
	}
}
