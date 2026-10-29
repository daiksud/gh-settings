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
	"strconv"
	"strings"
	"sync"
	"testing"
)

func TestSecurityStatusNormalizationAcrossExecutableCommands(t *testing.T) {
	binary := acceptanceBinary(t)
	for _, control := range []struct{ ui, key, companion, status string }{
		{"secret_protection", "secret_scanning", "secret_scanning_push_protection", "disabled"},
		{"push_protection", "secret_scanning_push_protection", "secret_scanning", "enabled"},
	} {
		t.Run(control.ui, func(t *testing.T) {
			var mu sync.Mutex
			statuses := map[string]any{control.key: "FUTURE_STATUS", control.companion: control.status}
			keepUnknown := true
			description := "current"
			var securityWrites []map[string]any
			generalWrites := 0
			fixture, err := startHTTPFixture(func(w http.ResponseWriter, req *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				w.Header().Set("Content-Type", "application/json")
				if servePRAccessFixtureRead(w, req) || serveIssueCreationPolicyFixtureRead(w, req) || serveSponsorshipsFixtureRead(w, req) {
					return
				}
				if isMergeMethodsFixtureRead(req) {
					fmt.Fprint(w, mergeMethodsFixtureResponse)
					return
				}
				if req.URL.Path == "/repos/owner/current" {
					if req.Method == "GET" {
						var base map[string]any
						if err := json.Unmarshal([]byte(planResponse), &base); err != nil {
							t.Error(err)
						}
						base["description"] = description
						block := map[string]any{}
						for key, status := range statuses {
							block[key] = map[string]any{"status": status}
						}
						base["security_and_analysis"] = block
						json.NewEncoder(w).Encode(base)
						return
					}
					if req.Method != "PATCH" {
						t.Error(req.Method)
						w.WriteHeader(400)
						return
					}
					var payload map[string]any
					if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
						t.Error(err)
						w.WriteHeader(400)
						return
					}
					if block, ok := payload["security_and_analysis"].(map[string]any); ok {
						if len(payload) != 1 || len(block) != 1 {
							t.Error("wide security PATCH", payload)
						}
						securityWrites = append(securityWrites, payload)
						for key, value := range block {
							nested, ok := value.(map[string]any)
							if !ok || len(nested) != 1 || (nested["status"] != "enabled" && nested["status"] != "disabled") || (key != "secret_scanning" && key != "secret_scanning_push_protection") {
								t.Error("invalid narrow status", payload)
								w.WriteHeader(400)
								return
							}
							if !keepUnknown {
								statuses[key] = nested["status"]
							}
						}
					} else {
						if !reflect.DeepEqual(payload, map[string]any{"description": "earlier retained"}) {
							t.Error("unexpected ordinary write", payload)
							w.WriteHeader(400)
							return
						}
						generalWrites++
						description = "earlier retained"
					}
					fmt.Fprint(w, `{}`)
					return
				}
				if req.Method != "GET" {
					t.Error(req.Method, req.URL)
					w.WriteHeader(400)
					return
				}
				switch req.URL.Path {
				case "/repos/owner/current/topics":
					fmt.Fprint(w, `{"names":[]}`)
				case "/repos/owner/current/labels":
					fmt.Fprint(w, `[]`)
				default:
					t.Error("unexpected endpoint", req.URL)
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
				result, err := executeCLI(binary, fixture.work, fixture.root, env, args, false)
				if err != nil {
					t.Fatal(err)
				}
				status := 0
				if result.runErr != nil {
					var failure *exec.ExitError
					if !errors.As(result.runErr, &failure) {
						t.Fatal(result.runErr)
					}
					status = failure.ExitCode()
				}
				return result.stdout.String(), result.stderr.String(), status
			}
			intent := func(values, about string) string {
				t.Helper()
				input := repositoryInput(about + "settings: {security_and_quality: {advanced_security: " + values + "}}\n")
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
			path := "repository.settings.security_and_quality.advanced_security." + control.ui
			warning := "Warning: " + path + ": observed \"FUTURE_STATUS\" is outside UI choices; using false (projection only)."
			output, diagnostic, status := run("export", "--security", "secret-scanning")
			if status != 0 || output != "" || diagnostic != warning+"\n" {
				t.Fatal("export warning", status, output, diagnostic)
			}
			data, err := os.ReadFile(filepath.Join(fixture.work, ".github/settings.yml"))
			if err != nil || !strings.Contains(string(data), control.ui+": false") || strings.Contains(string(data), "FUTURE_STATUS") {
				t.Fatal("effective YAML", string(data), err)
			}
			for _, desired := range []bool{false, true} {
				input := intent("{"+control.ui+": "+strconv.FormatBool(desired)+"}", "")
				output, diagnostic, status = run("plan")
				decision := path + ": false (normalized observation) → " + strconv.FormatBool(desired) + " (write required)"
				if status != 0 || diagnostic != "" || strings.Count(output, warning) != 1 || !strings.Contains(output, decision) || strings.Contains(output, "No differences") {
					t.Fatal("plan decision", status, output, diagnostic)
				}
				checkSource(input)
			}
			report, summary := filepath.Join(fixture.root, "security.json"), filepath.Join(fixture.root, "security-summary.md")
			output, diagnostic, status = run("action", "plan", report)
			if status != 0 || output != "" || diagnostic != "" {
				t.Fatal("private plan", status, output, diagnostic)
			}
			data, err = os.ReadFile(report)
			var saved struct {
				Outcome  string
				Managed  int
				Security []struct {
					Field         string
					Before, After bool
					Normalization struct {
						Path     string
						Raw      string `json:"raw_value"`
						Fallback *bool
					} `json:"security_status_normalization"`
				}
			}
			if err != nil || json.Unmarshal(data, &saved) != nil {
				t.Fatal(string(data), err)
			}
			if saved.Outcome != "changed" || saved.Managed != 1 || len(saved.Security) != 1 || saved.Security[0].Field != "security."+control.key || saved.Security[0].Before || !saved.Security[0].After || saved.Security[0].Normalization.Path != path || saved.Security[0].Normalization.Raw != "FUTURE_STATUS" || saved.Security[0].Normalization.Fallback == nil || *saved.Security[0].Normalization.Fallback {
				t.Fatal("typed private evidence", string(data))
			}
			output, diagnostic, status = run("action", "summary", report, summary)
			data, err = os.ReadFile(summary)
			if err != nil || status != 0 || output != "" || diagnostic != "" || !strings.Contains(html.UnescapeString(string(data)), warning) || !strings.Contains(html.UnescapeString(string(data)), path+": false (normalized observation) → true (write required)") {
				t.Fatal("detailed Summary", status, output, diagnostic, string(data), err)
			}
			summary = filepath.Join(fixture.root, "security-status-only.md")
			output, diagnostic, status = run("action", "summary", report, summary, "status-only")
			data, err = os.ReadFile(summary)
			if err != nil || status != 0 || diagnostic != "" || output != "plan-outcome=changed\n" || strings.Contains(string(data), "FUTURE_STATUS") || strings.Contains(string(data), path) {
				t.Fatal("status privacy", status, output, diagnostic, string(data), err)
			}
			mu.Lock()
			readOnlyWrites := len(securityWrites) + generalWrites
			mu.Unlock()
			if readOnlyWrites != 0 {
				t.Fatal("read-only commands wrote", readOnlyWrites)
			}
			input := intent("{}", "")
			output, diagnostic, status = run("plan")
			if status != 0 || diagnostic != "" || strings.Contains(output, "FUTURE_STATUS") || strings.Contains(output, "Warning:") {
				t.Fatal("unmanaged disclosure", status, output, diagnostic)
			}
			checkSource(input)
			input = intent("{"+control.ui+": false}", "")
			output, diagnostic, status = run("apply")
			if status != 1 || output != "" || strings.Count(diagnostic, warning) != 3 || !strings.Contains(diagnostic, "write effects may already have applied") {
				t.Fatal("fallback false verification", status, output, diagnostic)
			}
			mu.Lock()
			last := securityWrites[len(securityWrites)-1]
			mu.Unlock()
			if !reflect.DeepEqual(last, map[string]any{"security_and_analysis": map[string]any{control.key: map[string]any{"status": "disabled"}}}) {
				t.Fatal(last)
			}
			checkSource(input)
			mu.Lock()
			keepUnknown = false
			mu.Unlock()
			input = intent("{"+control.ui+": true}", "")
			output, diagnostic, status = run("apply")
			if status != 0 || strings.Count(diagnostic, warning) != 2 || !strings.Contains(output, "Applied repository settings") {
				t.Fatal("different desired", status, output, diagnostic)
			}
			checkSource(input)
			mu.Lock()
			beforeRepeat := len(securityWrites)
			mu.Unlock()
			output, diagnostic, status = run("apply")
			if status != 0 || diagnostic != "" {
				t.Fatal("canonical repeat", status, output, diagnostic)
			}
			mu.Lock()
			afterRepeat := len(securityWrites)
			mu.Unlock()
			if beforeRepeat != afterRepeat {
				t.Fatal("canonical no-op wrote")
			}
			output, diagnostic, status = run("plan")
			if status != 0 || diagnostic != "" || output != "No differences for managed repository settings.\n" {
				t.Fatal(status, output, diagnostic)
			}

			mu.Lock()
			unsafeDesired := true
			if control.ui == "secret_protection" {
				statuses[control.key] = "enabled"
				unsafeDesired = false
			} else {
				statuses[control.key] = "disabled"
			}
			statuses[control.companion] = "UNMANAGED_SECRET_CANARY"
			beforeUnsafe := len(securityWrites)
			mu.Unlock()
			input = intent("{"+control.ui+": "+strconv.FormatBool(unsafeDesired)+"}", "")
			output, diagnostic, status = run("apply")
			if status != 1 || output != "" || strings.Contains(diagnostic, "Warning:") || strings.Contains(diagnostic, "UNMANAGED_SECRET_CANARY") || !strings.Contains(diagnostic, "preflight Secret scanning") || strings.Contains(diagnostic, "write effects may already have applied") {
				t.Fatal("unverified companion prerequisite", status, output, diagnostic)
			}
			mu.Lock()
			afterUnsafe := len(securityWrites)
			mu.Unlock()
			if afterUnsafe != beforeUnsafe {
				t.Fatal("unsafe prerequisite wrote")
			}
			checkSource(input)
			mu.Lock()
			statuses[control.key] = "FUTURE_STATUS"
			statuses[control.companion] = control.status
			keepUnknown = true
			beforeGeneral := generalWrites
			mu.Unlock()
			input = intent("{"+control.ui+": false}", "about: {description: earlier retained}\n")
			output, diagnostic, status = run("apply")
			if status != 1 || !strings.Contains(diagnostic, "earlier successful writes remain applied") || strings.Count(diagnostic, warning) != 3 {
				t.Fatal("partial effects", status, output, diagnostic)
			}
			mu.Lock()
			retained := description == "earlier retained" && generalWrites == beforeGeneral+1 && statuses[control.key] == "FUTURE_STATUS"
			mu.Unlock()
			if !retained {
				t.Fatal("earlier write not retained")
			}
			checkSource(input)
			mu.Lock()
			statuses["secret_scanning"] = "FUTURE_SCANNING"
			statuses["secret_scanning_push_protection"] = "FUTURE_PUSH"
			keepUnknown = false
			beforeCoupled := len(securityWrites)
			mu.Unlock()
			input = intent("{secret_protection: true, push_protection: true}", "")
			output, diagnostic, status = run("apply")
			if status != 0 || strings.Count(diagnostic, "Warning:") != 4 {
				t.Fatal("coupled normalization", status, output, diagnostic)
			}
			mu.Lock()
			coupled := append([]map[string]any{}, securityWrites[beforeCoupled:]...)
			bothEnabled := statuses["secret_scanning"] == "enabled" && statuses["secret_scanning_push_protection"] == "enabled"
			mu.Unlock()
			if !bothEnabled || !reflect.DeepEqual(coupled, []map[string]any{{"security_and_analysis": map[string]any{"secret_scanning": map[string]any{"status": "enabled"}}}, {"security_and_analysis": map[string]any{"secret_scanning_push_protection": map[string]any{"status": "enabled"}}}}) {
				t.Fatal("unsafe coupled order", coupled)
			}
			checkSource(input)
			mu.Lock()
			statuses["secret_scanning"] = "FUTURE_STATUS"
			statuses["secret_scanning_push_protection"] = "enabled"
			mu.Unlock()
			output, diagnostic, status = run("export", "--security", "secret-scanning")
			inconsistentWarning := "Warning: repository.settings.security_and_quality.advanced_security.secret_protection: observed \"FUTURE_STATUS\" is outside UI choices; using false (projection only)."
			if status != 1 || output != "" || strings.Count(diagnostic, inconsistentWarning) != 1 || !strings.Contains(diagnostic, "consistent declaration") {
				t.Fatal("inconsistent export", status, output, diagnostic)
			}
			checkSource(input)
			mu.Lock()
			statuses[control.key] = nil
			statuses[control.companion] = control.status
			beforeInvalid := len(securityWrites)
			mu.Unlock()
			input = intent("{"+control.ui+": false}", "")
			output, diagnostic, status = run("apply")
			if status != 1 || strings.Contains(diagnostic, "Warning:") || !strings.Contains(diagnostic, "missing/null") {
				t.Fatal("null became default", status, output, diagnostic)
			}
			mu.Lock()
			afterInvalid := len(securityWrites)
			mu.Unlock()
			if beforeInvalid != afterInvalid {
				t.Fatal("null acquisition wrote")
			}
			checkSource(input)
		})
	}
}
