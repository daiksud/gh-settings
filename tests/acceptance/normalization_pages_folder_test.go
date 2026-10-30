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

func TestPagesFolderNormalizationAcrossExecutableCommands(t *testing.T) {
	binary := acceptanceBinary(t)
	var mu sync.Mutex
	state := map[string]any{"build_type": "legacy", "source": map[string]any{"branch": "main", "path": "/future"}, "cname": nil, "https_enforced": true, "public": true}
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
				var payload map[string]any
				json.NewDecoder(r.Body).Decode(&payload)
				if !reflect.DeepEqual(payload, map[string]any{"description": "updated"}) {
					t.Error(payload)
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
			if !reflect.DeepEqual(payload, map[string]any{"source": map[string]any{"branch": "main", "path": "/"}}) && !reflect.DeepEqual(payload, map[string]any{"https_enforced": false}) {
				t.Error("unowned Pages payload", payload)
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
	path := "repository.settings.pages.build_and_deployment.select_folder"
	warning := "Warning: " + path + ": observed \"/future\" is outside UI choices; using \"root\" (projection only)."
	decision := path + ": \"root\" (normalized observation) → \"root\" (write required)"
	output, diagnostic, status := run("export", "--pages")
	if status != 0 || output != "" || diagnostic != warning+"\n" {
		t.Fatal("export", status, output, diagnostic)
	}
	data, err := os.ReadFile(filepath.Join(fixture.work, ".github/settings.yml"))
	if err != nil || !strings.Contains(string(data), "select_folder: root") || strings.Contains(string(data), "/future") {
		t.Fatal("effective export", string(data), err)
	}
	input := intent("{build_and_deployment: {branch: main, select_folder: root}}", "")
	output, diagnostic, status = run("plan")
	if status != 0 || diagnostic != "" || strings.Count(output, warning) != 1 || !strings.Contains(output, decision) || strings.Contains(output, "No differences") {
		t.Fatal("fallback equality plan", status, output, diagnostic)
	}
	checkSource(input)
	report, summary := filepath.Join(fixture.root, "pages-normalized.json"), filepath.Join(fixture.root, "pages-normalized.md")
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
	if err != nil || json.Unmarshal(data, &saved) != nil || saved.Outcome != "changed" || saved.Managed != 2 || len(saved.Pages) != 1 || saved.Pages[0].Field != "pages.source.path" || saved.Pages[0].Before != "/" || saved.Pages[0].After != "/" || saved.Pages[0].Normalization.Path != path || saved.Pages[0].Normalization.Raw != "/future" || saved.Pages[0].Normalization.Fallback != "root" {
		t.Fatal("private source evidence/count", string(data), err)
	}
	output, diagnostic, status = run("action", "summary", report, summary)
	data, err = os.ReadFile(summary)
	if err != nil || status != 0 || output != "" || diagnostic != "" || !strings.Contains(html.UnescapeString(string(data)), warning) || !strings.Contains(html.UnescapeString(string(data)), decision) {
		t.Fatal("detailed summary", string(data), status, output, diagnostic, err)
	}
	privateSummary := filepath.Join(fixture.root, "pages-private.md")
	output, diagnostic, status = run("action", "summary", report, privateSummary, "status-only")
	data, err = os.ReadFile(privateSummary)
	if err != nil || status != 0 || output != "plan-outcome=changed\n" || diagnostic != "" || strings.Contains(string(data), "/future") || strings.Contains(string(data), path) {
		t.Fatal("private summary", string(data), status, output, diagnostic, err)
	}
	mu.Lock()
	readOnlyWrites := len(pagesWrites) + generalWrites
	mu.Unlock()
	if readOnlyWrites != 0 {
		t.Fatal("read-only writes", readOnlyWrites)
	}
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
		t.Fatal("canonical confirmation", status, output, diagnostic)
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
	state["source"] = map[string]any{"branch": "main", "path": "/future"}
	mu.Unlock()
	if beforeRepeat != afterRepeat {
		t.Fatal("canonical no-op wrote")
	}
	input = intent("{enforce_https: false}", "")
	output, diagnostic, status = run("apply")
	if status != 0 || diagnostic != "" || strings.Contains(output, "/future") {
		t.Fatal("unowned source disclosure", status, output, diagnostic)
	}
	checkSource(input)
	mu.Lock()
	source := state["source"].(map[string]any)
	if source["path"] != "/future" {
		t.Fatal("unowned source overwritten", source)
	}
	ignored = true
	mu.Unlock()
	input = intent("{build_and_deployment: {branch: main, select_folder: root}}", "about: {description: updated}\n")
	output, diagnostic, status = run("apply")
	if status != 1 || output != "" || strings.Count(diagnostic, warning) != 3 || !strings.Contains(diagnostic, "earlier successful writes remain applied") {
		t.Fatal("partial failure", status, output, diagnostic)
	}
	checkSource(input)
	mu.Lock()
	if description != "updated" || generalWrites != 1 {
		t.Fatal(description, generalWrites)
	}
	state["source"].(map[string]any)["path"] = nil
	beforeInvalid := len(pagesWrites)
	mu.Unlock()
	output, diagnostic, status = run("export", "--pages")
	if status != 1 || output != "" || strings.Contains(diagnostic, "Warning:") {
		t.Fatal("null export", status, output, diagnostic)
	}
	checkSource(input)
	output, diagnostic, status = run("apply")
	if status != 1 || strings.Contains(diagnostic, "Warning:") {
		t.Fatal("null apply", status, output, diagnostic)
	}
	checkSource(input)
	mu.Lock()
	afterInvalid := len(pagesWrites)
	mu.Unlock()
	if beforeInvalid != afterInvalid {
		t.Fatal("null source mutated")
	}
}
