//go:build acceptance

package acceptance

import (
	"encoding/json"
	"errors"
	"fmt"
	"go.yaml.in/yaml/v4"
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

func acceptanceScanningAlertsIntent(mode string) map[string]any {
	tools := []any{map[string]any{"tool": "CodeQL", "alerts_threshold": "errors", "security_alerts_threshold": "high_or_higher"}}
	if mode != "tool removed" {
		name, alerts := "z tool", "errors"
		if mode == "different" {
			alerts = "all"
		}
		if mode == "replacement" {
			name, alerts = "zz new tool", "none"
		}
		tools = append(tools, map[string]any{"tool": name, "alerts_threshold": alerts, "security_alerts_threshold": "all"})
	}
	if mode == "empty tools" {
		tools = []any{}
	}
	rules := []any{map[string]any{"type": "code_scanning", "parameters": map[string]any{"code_scanning_tools": tools}}}
	if mode == "rule removed" {
		rules = []any{}
	}
	return map[string]any{"name": "basic", "target": "branch", "enforcement": "active", "conditions": map[string]any{"ref_name": map[string]any{"include": []any{"~DEFAULT_BRANCH"}, "exclude": []any{}}}, "bypass_actors": []any{}, "rules": rules}
}

func TestRulesetScanningAlertsNormalizationAcrossExecutableCommands(t *testing.T) {
	binary := acceptanceBinary(t)
	var mu sync.Mutex
	var state map[string]any
	json.Unmarshal([]byte(acceptanceRuleset), &state)
	reset := func(raw any) {
		tools := []any{map[string]any{"tool": "z tool", "alerts_threshold": raw, "security_alerts_threshold": "all"}, map[string]any{"tool": "CodeQL", "alerts_threshold": "", "security_alerts_threshold": "high_or_higher"}}
		state["rules"] = []any{map[string]any{"type": "code_scanning", "parameters": map[string]any{"code_scanning_tools": tools}}}
	}
	reset("future")
	desired := "equal"
	ignored, retained, includeOther := true, false, false
	reads, writes, requests := 0, 0, 0
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
		switch r.URL.Path {
		case "/repos/owner/current", "/repos/owner/current/topics", "/repos/owner/current/labels":
			if r.Method != "GET" {
				t.Error("unrelated write")
				w.WriteHeader(400)
				return
			}
			switch r.URL.Path {
			case "/repos/owner/current":
				fmt.Fprint(w, planResponse)
			case "/repos/owner/current/topics":
				fmt.Fprint(w, `{"names":[]}`)
			default:
				fmt.Fprint(w, `[]`)
			}
		case "/repos/owner/current/rulesets":
			if r.Method != "GET" {
				t.Error("lifecycle write")
				w.WriteHeader(400)
				return
			}
			items := []any{map[string]any{"id": 7, "source_type": "Repository", "source": "owner/current"}}
			if includeOther {
				items = append(items, map[string]any{"id": 9, "source_type": "Repository", "source": "owner/current"})
			}
			json.NewEncoder(w).Encode(items)
		case "/repos/owner/current/rulesets/9":
			if r.Method != "GET" {
				t.Error("unselected write")
				w.WriteHeader(400)
				return
			}
			other := acceptanceScanningAlertsIntent("equal")
			other["id"], other["name"], other["source_type"], other["source"] = 9, "UNMANAGED_SCANNING", "Repository", "owner/current"
			other["rules"].([]any)[0].(map[string]any)["parameters"].(map[string]any)["code_scanning_tools"].([]any)[0].(map[string]any)["alerts_threshold"] = "PRIVATE"
			json.NewEncoder(w).Encode(other)
		case "/repos/owner/current/rulesets/7":
			if r.Method == "GET" {
				reads++
				json.NewEncoder(w).Encode(state)
				return
			}
			if r.Method != "PUT" {
				t.Error("lifecycle write")
				w.WriteHeader(400)
				return
			}
			var payload map[string]any
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Error(err)
			}
			want := acceptanceScanningAlertsIntent(desired)
			if !reflect.DeepEqual(payload, want) {
				t.Error("literal whole-policy mismatch", payload, want)
			}
			writes++
			if !ignored {
				state["rules"] = acceptanceScanningAlertsIntent(desired)["rules"]
			}
			if retained {
				state["rules"] = acceptanceScanningAlertsIntent("equal")["rules"]
			}
			want["id"], want["source_type"], want["source"] = 7, "Repository", "owner/current"
			json.NewEncoder(w).Encode(want)
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
			var e *exec.ExitError
			if !errors.As(r.runErr, &e) {
				t.Fatal(r.runErr)
			}
			status = e.ExitCode()
		}
		return r.stdout.String(), r.stderr.String(), status
	}
	intent := func(mode string) string {
		t.Helper()
		mu.Lock()
		desired = mode
		mu.Unlock()
		tools := "[{tool_name: CodeQL, alerts: errors, security_alerts: high_or_higher}"
		if mode != "tool removed" {
			name, alerts := "z tool", "errors"
			if mode == "different" {
				alerts = "all"
			}
			if mode == "replacement" {
				name, alerts = "zz new tool", "none"
			}
			tools += fmt.Sprintf(", {tool_name: '%s', alerts: %s, security_alerts: all}", name, alerts)
		}
		tools += "]"
		if mode == "empty tools" {
			tools = "[]"
		}
		rules := "{require_code_scanning_results: {code_scanning_tools: " + tools + "}}"
		if mode == "rule removed" {
			rules = "{}"
		}
		input := repositoryInput("settings: {rulesets: [{id: 7, ruleset_name: basic, enforcement_status: active, target_branches: {include_default_branch: true}, rules: " + rules + "}]}\n")
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
	path := "repository.settings.rulesets.rules.require_code_scanning_results.code_scanning_tools.alerts"
	warningA := `Warning: ` + path + `: observed "" is outside UI choices; using "errors" (projection only). Ruleset 7 "basic", tool="CodeQL".`
	warningZ := `Warning: ` + path + `: observed "future" is outside UI choices; using "errors" (projection only). Ruleset 7 "basic", tool="z tool".`
	warnings := warningA + "\n" + warningZ + "\n"
	output, diagnostic, status := run("export", "--rulesets")
	if status != 0 || output != "" || diagnostic != warnings {
		t.Fatal("export", status, output, diagnostic)
	}
	data, err := os.ReadFile(filepath.Join(fixture.work, ".github/settings.yml"))
	if err != nil {
		t.Fatal(err)
	}
	var exported map[string]any
	if err := yaml.Unmarshal(data, &exported); err != nil {
		t.Fatal(err)
	}
	tools := exported["repository"].(map[string]any)["settings"].(map[string]any)["rulesets"].([]any)[0].(map[string]any)["rules"].(map[string]any)["require_code_scanning_results"].(map[string]any)["code_scanning_tools"]
	wantTools := []any{map[string]any{"tool_name": "CodeQL", "alerts": "errors", "security_alerts": "high_or_higher"}, map[string]any{"tool_name": "z tool", "alerts": "errors", "security_alerts": "all"}}
	if !reflect.DeepEqual(tools, wantTools) {
		t.Fatal("UI differs", tools, wantTools)
	}
	report, summary := filepath.Join(fixture.root, "scanning.json"), filepath.Join(fixture.root, "scanning.md")
	modes := []string{"equal", "different", "tool removed", "replacement", "empty tools", "rule removed"}
	for _, mode := range modes {
		input := intent(mode)
		output, diagnostic, status = run("plan")
		after := `"errors" (write required)`
		switch mode {
		case "different":
			after = `"all" (write required)`
		case "tool removed", "replacement", "empty tools":
			after = "inactive (tool removed; write required)"
		case "rule removed":
			after = "inactive (rule removed; write required)"
		}
		decision := path + ` (Ruleset 7 "basic", tool="z tool"): "errors" (normalized observation) → ` + after
		if status != 0 || diagnostic != "" || strings.Count(output, warningA) != 1 || strings.Count(output, warningZ) != 1 || !strings.Contains(output, decision) || strings.Count(output, "Effective policy:") != 1 {
			t.Fatal("plan", mode, status, output, diagnostic)
		}
		output, diagnostic, status = run("action", "plan", report)
		data, err = os.ReadFile(report)
		var saved struct {
			Outcome  string
			Managed  int
			Rulesets []struct {
				Records []struct {
					Tool, Path string
					Raw        *string `json:"raw_value"`
					Fallback   string
				} `json:"scanning_alerts_normalizations"`
			}
		}
		decodeErr := json.Unmarshal(data, &saved)
		if err != nil || decodeErr != nil || status != 0 || output != "" || diagnostic != "" || saved.Outcome != "changed" || saved.Managed != 1 || len(saved.Rulesets) != 1 || len(saved.Rulesets[0].Records) != 2 {
			t.Fatal("report", status, string(data), err, decodeErr)
		}
		for _, n := range saved.Rulesets[0].Records {
			if n.Raw == nil || n.Path != path || n.Fallback != "errors" || n.Tool == "CodeQL" && *n.Raw != "" || n.Tool == "z tool" && *n.Raw != "future" || n.Tool != "CodeQL" && n.Tool != "z tool" {
				t.Fatal(n)
			}
		}
		checkSource(input)
	}
	mu.Lock()
	beforeRequests := requests
	mu.Unlock()
	output, diagnostic, status = run("action", "summary", report, summary)
	data, err = os.ReadFile(summary)
	visible := html.UnescapeString(string(data))
	if err != nil || status != 0 || output != "" || diagnostic != "" || strings.Count(visible, warningA) != 1 || strings.Count(visible, warningZ) != 1 || !strings.Contains(visible, "inactive (rule removed; write required)") {
		t.Fatal("Summary", status, visible, diagnostic, err)
	}
	private := filepath.Join(fixture.root, "private.md")
	output, diagnostic, status = run("action", "summary", report, private, "status-only")
	data, err = os.ReadFile(private)
	if err != nil || status != 0 || output != "plan-outcome=changed\n" || diagnostic != "" || strings.Contains(string(data), "tool=") || strings.Contains(string(data), path) || strings.Contains(string(data), "Ruleset 7") {
		t.Fatal("privacy", status, string(data), diagnostic, err)
	}
	mu.Lock()
	if requests != beforeRequests || writes != 0 {
		t.Error("plan/Summary side effects")
	}
	beforeReads := reads
	mu.Unlock()
	input := intent("equal")
	output, diagnostic, status = run("apply")
	if status != 1 || output != "" || strings.Count(diagnostic, warningA) != 2 || strings.Count(diagnostic, warningZ) != 2 || !strings.Contains(diagnostic, "writes may already have applied") {
		t.Fatal("raw confirmed fallback", status, output, diagnostic)
	}
	checkSource(input)
	mu.Lock()
	if reads-beforeReads != 2 || writes != 1 {
		t.Error("single PUT/fresh GET lost", reads, writes)
	}
	ignored = false
	mu.Unlock()
	for _, mode := range modes {
		mu.Lock()
		reset("future")
		mu.Unlock()
		input = intent(mode)
		output, diagnostic, status = run("apply")
		if status != 0 || diagnostic != warnings {
			t.Fatal("confirmed policy", mode, status, output, diagnostic)
		}
		checkSource(input)
		mu.Lock()
		beforeWrites := writes
		includeOther = true
		mu.Unlock()
		output, diagnostic, status = run("plan")
		if status != 0 || diagnostic != "" || output != "No differences for managed repository settings.\n" {
			t.Fatal("canonical repetition", status, output, diagnostic)
		}
		output, diagnostic, status = run("apply")
		if status != 0 || diagnostic != "" || strings.Contains(output, "UNMANAGED") {
			t.Fatal("private unselected", status, output, diagnostic)
		}
		mu.Lock()
		if writes != beforeWrites {
			t.Error("canonical no-op wrote")
		}
		mu.Unlock()
	}
	for _, mode := range []string{"tool removed", "rule removed"} {
		mu.Lock()
		reset("future")
		ignored, retained = true, true
		mu.Unlock()
		input = intent(mode)
		output, diagnostic, status = run("apply")
		if status != 1 || output != "" || !strings.Contains(diagnostic, "writes may already have applied") {
			t.Fatal("retained canonical confirmed removal", status, output, diagnostic)
		}
		checkSource(input)
	}
	for _, bad := range []any{nil, 1, true} {
		mu.Lock()
		reset(bad)
		beforeWrites := writes
		mu.Unlock()
		input = intent("equal")
		for _, args := range [][]string{{"export", "--rulesets"}, {"apply"}} {
			output, diagnostic, status = run(args...)
			if status != 1 || output != "" || strings.Contains(diagnostic, "Warning:") {
				t.Fatal("malformed acquisition", args, status, output, diagnostic)
			}
			checkSource(input)
		}
		mu.Lock()
		if writes != beforeWrites {
			t.Error("malformed write")
		}
		mu.Unlock()
	}
}
