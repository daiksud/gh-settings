//go:build acceptance

package acceptance

import (
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
)

func TestPagesModeNormalizationAcrossExecutableCommands(t *testing.T) {
	binary := acceptanceBinary(t)
	var mu sync.Mutex
	state := map[string]any{"build_type": "FUTURE_MODE", "source": map[string]any{"branch": "main", "path": "/"}, "cname": nil, "https_enforced": true, "public": true}
	ignored := true
	description := "current"
	generalWrites := 0
	var pagesWrites []map[string]any
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
		switch r.URL.Path {
		case "/repos/owner/current":
			if r.Method == "PATCH" {
				var p map[string]any
				json.NewDecoder(r.Body).Decode(&p)
				if !reflect.DeepEqual(p, map[string]any{"description": "updated"}) {
					t.Error(p)
				}
				description = "updated"
				generalWrites++
				fmt.Fprint(w, `{}`)
				return
			}
			if r.Method != "GET" {
				t.Error(r.Method)
				w.WriteHeader(400)
				return
			}
			var base map[string]any
			json.Unmarshal([]byte(planResponse), &base)
			base["id"] = 123
			base["full_name"] = "owner/current"
			base["archived"] = false
			base["description"] = description
			json.NewEncoder(w).Encode(base)
		case "/repos/owner/current/topics":
			fmt.Fprint(w, `{"names":[]}`)
		case "/repos/owner/current/labels":
			fmt.Fprint(w, `[]`)
		case "/repos/owner/current/pages":
			if r.Method == "GET" {
				json.NewEncoder(w).Encode(state)
				return
			}
			if r.Method != "PUT" {
				t.Error("site lifecycle/build mutation", r.Method)
				w.WriteHeader(400)
				return
			}
			var payload map[string]any
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Error(err)
			}
			valid := reflect.DeepEqual(payload, map[string]any{"build_type": "legacy"}) || reflect.DeepEqual(payload, map[string]any{"build_type": "workflow"}) || reflect.DeepEqual(payload, map[string]any{"https_enforced": false}) || reflect.DeepEqual(payload, map[string]any{"build_type": "legacy", "source": map[string]any{"branch": "main", "path": "/"}})
			if !valid {
				t.Error("unowned payload", payload)
			}
			pagesWrites = append(pagesWrites, payload)
			if !ignored {
				for key, value := range payload {
					state[key] = value
				}
			}
			w.WriteHeader(204)
		default:
			t.Error("unexpected request", r.Method, r.URL)
			w.WriteHeader(400)
		}
	}, false)
	if err != nil {
		t.Fatal(err)
	}
	defer fixture.close()
	env := append(exportFixtureEnv(fixture.cfg), "SETTINGS_TARGET=owner/current", "SETTINGS_ACTOR=user", "SETTINGS_TRIGGERING_ACTOR=user", "SETTINGS_EVENT=push", "SETTINGS_RUN_ID=123", "SETTINGS_RUN_ATTEMPT=1", "SETTINGS_WORKFLOW_REF=owner/current/.github/workflows/settings.yml@refs/heads/main", "SETTINGS_WORKFLOW_SHA="+strings.Repeat("a", 40), "SETTINGS_CONFIG_SHA="+strings.Repeat("b", 40), "SETTINGS_CLI_VERSION=dev", "SETTINGS_CLI_SHA256="+strings.Repeat("c", 64))
	if err := setupOrdinaryOrigin(fixture.work, env); err != nil {
		t.Fatal(err)
	}
	run := func(args ...string) (string, string, int) {
		t.Helper()
		r, err := executeCLI(binary, fixture.work, fixture.root, env, args, false)
		if err != nil {
			t.Fatal(err)
		}
		status := 0
		if r.runErr != nil {
			var failure *exec.ExitError
			if !errors.As(r.runErr, &failure) {
				t.Fatal(r.runErr)
			}
			status = failure.ExitCode()
		}
		return r.stdout.String(), r.stderr.String(), status
	}
	intent := func(pages, about string) string {
		t.Helper()
		input := repositoryInput(about + "settings: {pages: " + pages + "}\n")
		if err := writeSettingsFile(fixture.work, input); err != nil {
			t.Fatal(err)
		}
		return input
	}
	checkSource := func(input string) {
		t.Helper()
		if err := checkSettingsUnchanged(fixture.work, input); err != nil {
			t.Fatal(err)
		}
	}
	path := "repository.settings.pages.build_and_deployment.source"
	warning := "Warning: " + path + ": observed \"FUTURE_MODE\" is outside UI choices; using \"deploy_from_a_branch\" (projection only)."
	output, diagnostic, status := run("export", "--pages")
	if status != 0 || output != "" || diagnostic != warning+"\n" {
		t.Fatal("export", status, output, diagnostic)
	}
	data, err := os.ReadFile(filepath.Join(fixture.work, ".github/settings.yml"))
	if err != nil || !strings.Contains(string(data), "source: deploy_from_a_branch") || !strings.Contains(string(data), "branch: main") || strings.Contains(string(data), "FUTURE_MODE") {
		t.Fatal("effective tuple export", string(data), err)
	}
	for _, choice := range []string{"deploy_from_a_branch", "github_actions"} {
		input := intent("{build_and_deployment: {source: "+choice+"}}", "")
		output, diagnostic, status = run("plan")
		decision := path + ": \"deploy_from_a_branch\" (normalized observation) → \"" + choice + "\" (write required)"
		if status != 0 || diagnostic != "" || strings.Count(output, warning) != 1 || !strings.Contains(output, decision) || strings.Contains(output, "No differences") {
			t.Fatal("mode decision", status, output, diagnostic)
		}
		checkSource(input)
	}
	report, summary := filepath.Join(fixture.root, "mode.json"), filepath.Join(fixture.root, "mode.md")
	output, diagnostic, status = run("action", "plan", report)
	if status != 0 || output != "" || diagnostic != "" {
		t.Fatal(status, output, diagnostic)
	}
	data, err = os.ReadFile(report)
	var saved struct {
		Outcome string
		Managed int
		Pages   []struct {
			Field         string
			Before, After string
			Normalization struct {
				Path     string
				Raw      string `json:"raw_value"`
				Fallback string
			}
		}
	}
	if err != nil || json.Unmarshal(data, &saved) != nil || saved.Outcome != "changed" || saved.Managed != 1 || len(saved.Pages) != 1 || saved.Pages[0].Field != "pages.build_type" || saved.Pages[0].Before != "legacy" || saved.Pages[0].After != "workflow" || saved.Pages[0].Normalization.Path != path || saved.Pages[0].Normalization.Raw != "FUTURE_MODE" || saved.Pages[0].Normalization.Fallback != "deploy_from_a_branch" {
		t.Fatal("mode private evidence", string(data), err)
	}
	output, diagnostic, status = run("action", "summary", report, summary)
	data, err = os.ReadFile(summary)
	if err != nil || status != 0 || output != "" || diagnostic != "" || strings.Count(html.UnescapeString(string(data)), warning) != 1 || !strings.Contains(html.UnescapeString(string(data)), `"deploy_from_a_branch" (normalized observation) → "github_actions" (write required) (branch source becomes inactive)`) {
		t.Fatal("detailed summary", status, output, diagnostic, string(data), err)
	}
	private := filepath.Join(fixture.root, "mode-private.md")
	output, diagnostic, status = run("action", "summary", report, private, "status-only")
	data, err = os.ReadFile(private)
	if err != nil || status != 0 || output != "plan-outcome=changed\n" || diagnostic != "" || strings.Contains(string(data), "FUTURE_MODE") || strings.Contains(string(data), path) {
		t.Fatal("status privacy", status, output, diagnostic, string(data), err)
	}
	mu.Lock()
	readonlyWrites := len(pagesWrites) + generalWrites
	mu.Unlock()
	if readonlyWrites != 0 {
		t.Fatal("read-only writes", readonlyWrites)
	}
	input := intent("{build_and_deployment: {source: deploy_from_a_branch}}", "")
	output, diagnostic, status = run("apply")
	if status != 1 || output != "" || strings.Count(diagnostic, warning) != 3 || !strings.Contains(diagnostic, "write may have applied") {
		t.Fatal("retained unknown", status, output, diagnostic)
	}
	checkSource(input)
	mu.Lock()
	ignored = false
	mu.Unlock()
	output, diagnostic, status = run("apply")
	if status != 0 || strings.Count(diagnostic, warning) != 2 || !strings.Contains(output, "Applied repository settings") {
		t.Fatal("canonical legacy confirmation", status, output, diagnostic)
	}
	checkSource(input)
	mu.Lock()
	beforeRepeat := len(pagesWrites)
	mu.Unlock()
	output, diagnostic, status = run("apply")
	if status != 0 || diagnostic != "" {
		t.Fatal(status, output, diagnostic)
	}
	mu.Lock()
	afterRepeat := len(pagesWrites)
	state["build_type"] = "FUTURE_MODE"
	mu.Unlock()
	if beforeRepeat != afterRepeat {
		t.Fatal("canonical no-op wrote")
	}
	input = intent("{build_and_deployment: {source: github_actions}}", "")
	output, diagnostic, status = run("apply")
	if status != 0 || strings.Count(diagnostic, warning) != 2 {
		t.Fatal("different mode confirmation", status, output, diagnostic)
	}
	checkSource(input)
	mu.Lock()
	state["build_type"] = "FUTURE_MODE"
	beforePrerequisite := len(pagesWrites)
	mu.Unlock()
	input = intent("{build_and_deployment: {branch: main, select_folder: root}}", "")
	output, diagnostic, status = run("apply")
	if status != 1 || output != "" || strings.Contains(diagnostic, "FUTURE_MODE") || strings.Contains(diagnostic, "Warning:") || !strings.Contains(diagnostic, "explicit legacy mode") {
		t.Fatal("source applicability", status, output, diagnostic)
	}
	checkSource(input)
	mu.Lock()
	afterPrerequisite := len(pagesWrites)
	mu.Unlock()
	if beforePrerequisite != afterPrerequisite {
		t.Fatal("unknown mode allowed source write")
	}
	input = intent("{enforce_https: false}", "")
	output, diagnostic, status = run("apply")
	if status != 0 || diagnostic != "" || strings.Contains(output, "FUTURE_MODE") {
		t.Fatal("unowned mode", status, output, diagnostic)
	}
	checkSource(input)
	mu.Lock()
	if state["build_type"] != "FUTURE_MODE" {
		t.Fatal("unowned mode overwritten", state)
	}
	ignored = true
	mu.Unlock()
	input = intent("{build_and_deployment: {source: deploy_from_a_branch}}", "about: {description: updated}\n")
	output, diagnostic, status = run("apply")
	if status != 1 || output != "" || strings.Count(diagnostic, warning) != 3 || !strings.Contains(diagnostic, "earlier successful writes remain applied") {
		t.Fatal("partial progress", status, output, diagnostic)
	}
	checkSource(input)
	mu.Lock()
	if description != "updated" || generalWrites != 1 {
		t.Fatal(description, generalWrites)
	}
	state["source"] = map[string]any{"branch": "main", "path": "/future"}
	ignored = false
	mu.Unlock()
	input = intent("{build_and_deployment: {source: deploy_from_a_branch, branch: main, select_folder: root}}", "")
	output, diagnostic, status = run("action", "plan", report)
	if status != 0 || output != "" || diagnostic != "" {
		t.Fatal(status, output, diagnostic)
	}
	data, err = os.ReadFile(report)
	if err != nil || json.Unmarshal(data, &saved) != nil || saved.Managed != 3 || len(saved.Pages) != 2 || saved.Pages[0].Normalization.Path != path || saved.Pages[1].Normalization.Raw != "/future" || saved.Pages[1].Normalization.Fallback != "root" {
		t.Fatal("coupled source count", string(data), err)
	}
	output, diagnostic, status = run("apply")
	if status != 0 || strings.Count(diagnostic, warning) != 2 || strings.Count(diagnostic, "Warning:") != 4 {
		t.Fatal("coupled fresh warnings", status, output, diagnostic)
	}
	checkSource(input)
	mu.Lock()
	state["build_type"] = nil
	beforeInvalid := len(pagesWrites)
	mu.Unlock()
	output, diagnostic, status = run("export", "--pages")
	if status != 1 || output != "" || strings.Contains(diagnostic, "Warning:") {
		t.Fatal("null mode export", status, output, diagnostic)
	}
	checkSource(input)
	output, diagnostic, status = run("apply")
	if status != 1 || strings.Contains(diagnostic, "Warning:") {
		t.Fatal("null mode apply", status, output, diagnostic)
	}
	checkSource(input)
	mu.Lock()
	afterInvalid := len(pagesWrites)
	mu.Unlock()
	if beforeInvalid != afterInvalid {
		t.Fatal("missing mode wrote")
	}
}
