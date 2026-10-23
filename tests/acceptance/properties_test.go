//go:build acceptance

package acceptance

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
)

func TestCustomPropertyExportPlanApplyUnsetAndPrivateSummary(t *testing.T) {
	binary := acceptanceBinary(t)
	var mu sync.Mutex
	mode := ""
	propertyReads, redirected := 0, 0
	writes := []map[string]any{}
	events := []string{}
	state := map[string]any{"text": "CANARY-value", "tags": []string{"a"}, "flag": "true", "keep": "retained"}
	fixture, err := startHTTPFixture(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/repos/other/redirected" {
			redirected++
			w.WriteHeader(400)
			return
		}

		if r.URL.Path == "/graphql" {
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Error(err)
				w.WriteHeader(400)
				return
			}
			r.Body = io.NopCloser(bytes.NewReader(body))
			var q struct {
				Query     string
				Variables map[string]string
			}
			json.Unmarshal(body, &q)
			if strings.Contains(q.Query, "repositoryCustomProperty(") {
				propertyReads++
				if r.Method != "POST" || strings.Contains(q.Query, "mutation") || q.Variables["owner"] != "owner" {
					t.Error("unexpected property schema query")
					w.WriteHeader(400)
					return
				}
				property := map[string]any{"propertyName": q.Variables["property"], "valueType": "STRING", "regex": nil}
				if mode == "regex" {
					property["regex"] = ".*"
				}
				if mode == "missing-regex" {
					delete(property, "regex")
				}
				response := map[string]any{"data": map[string]any{"organization": map[string]any{"databaseId": 12, "login": "owner", "repositoryCustomProperty": property}}}
				if mode == "regex-error" {
					response["errors"] = []map[string]any{{"message": "CANARY-schema"}}
				}
				json.NewEncoder(w).Encode(response)
				return
			}
		}
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
				events = append(events, "repository")
				fmt.Fprint(w, `{}`)
				return
			}
			if r.Method != "GET" {
				t.Error("unexpected repository method")
				w.WriteHeader(400)
				return
			}
			var base map[string]any
			json.Unmarshal([]byte(planResponse), &base)
			base["id"] = 123
			base["full_name"] = "owner/current"
			base["archived"] = false
			base["owner"] = map[string]any{"id": 12, "login": "owner", "type": "Organization"}
			json.NewEncoder(w).Encode(base)
		case "/repos/owner/current/topics":
			fmt.Fprint(w, `{"names":[]}`)
		case "/repos/owner/current/labels":
			fmt.Fprint(w, `[]`)
		case "/orgs/owner/properties/schema":
			if r.Method != "GET" {
				t.Error("schema mutation")
				w.WriteHeader(400)
				return
			}
			propertyReads++
			if mode == "schema-denied" {
				w.WriteHeader(403)
				fmt.Fprint(w, `{"message":"CANARY-schema"}`)
				return
			}
			rows := []map[string]any{}
			for name, kind := range map[string]string{"text": "string", "tags": "multi_select", "flag": "true_false", "keep": "string"} {
				s := map[string]any{"property_name": name, "value_type": kind, "source_type": "organization", "required": false, "require_explicit_values": false, "default_value": nil, "values_editable_by": "org_and_repo_actors", "allowed_values": []string{"a", "b"}}
				if mode == "missing-schema" {
					delete(s, "required")
				}
				if mode == "default" {
					s["default_value"] = "fallback"
				}
				rows = append(rows, s)
			}
			json.NewEncoder(w).Encode(rows)
		case "/repos/owner/current/properties/values":
			if r.Method == "GET" {
				propertyReads++
				if mode == "404" {
					w.WriteHeader(404)
					fmt.Fprint(w, `{"message":"CANARY-value"}`)
					return
				}
				if mode == "get-redirect" {
					w.Header().Set("Location", "/repos/other/redirected")
					w.WriteHeader(301)
					return
				}
				rows := []map[string]any{}
				for n, v := range state {
					rows = append(rows, map[string]any{"property_name": n, "value": v})
				}
				json.NewEncoder(w).Encode(rows)
				return
			}
			if r.Method != "PATCH" {
				t.Error("unrelated mutation", r.Method)
				w.WriteHeader(400)
				return
			}
			var raw struct {
				Properties []struct {
					Name  string `json:"property_name"`
					Value any    `json:"value"`
				} `json:"properties"`
			}
			if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
				t.Error(err)
			}
			payload := map[string]any{}
			for _, p := range raw.Properties {
				if p.Name == "keep" {
					t.Error("unowned property sent")
				}
				payload[p.Name] = p.Value
			}
			writes = append(writes, payload)
			events = append(events, "properties")
			if mode == "write422" {
				w.WriteHeader(422)
				return
			}
			if mode == "patch-redirect" {
				w.Header().Set("Location", "/repos/other/redirected")
				w.WriteHeader(307)
				return
			}
			if mode != "ignored" {
				for n, v := range payload {
					if v == nil {
						delete(state, n)
					} else {
						state[n] = v
					}
				}
			}
			if mode == "unowned" {
				state["keep"] = "changed"
			}
			w.WriteHeader(204)
		default:
			t.Error("unrelated request", r.Method, r.URL)
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
	write := func(properties string) {
		t.Helper()
		if err := writeSettingsFile(fixture.work, "schemaVersion: 1\nsettings:\n  custom_properties: "+properties+"\n"); err != nil {
			t.Fatal(err)
		}
	}
	path := filepath.Join(fixture.work, ".github/settings.yml")
	run("export")
	mu.Lock()
	if propertyReads != 0 {
		t.Fatal("ordinary export reads properties")
	}
	mu.Unlock()
	run("export", "--custom-properties")
	run("validate")
	if got := run("plan"); !strings.Contains(got, "No differences") {
		t.Fatal(got)
	}
	write("{text: null, tags: [], flag: 'false'}")
	plan := run("plan")
	if !strings.Contains(plan, `"CANARY-value" → null`) || !strings.Contains(plan, `["a"] → []`) || !strings.Contains(plan, `"true" → "false"`) {
		t.Fatal(plan)
	}
	report := filepath.Join(fixture.root, "properties-report.json")
	run("action", "plan", report, "status-only")
	data, _ := os.ReadFile(report)
	var typed map[string]any
	json.Unmarshal(data, &typed)
	if typed["managed"] != float64(1) || len(typed["custom_properties"].([]any)) != 3 {
		t.Fatal(string(data))
	}
	summary := filepath.Join(fixture.root, "properties-summary.md")
	summaryEnv := append(fixtureEnv(fixture.cfg), "SETTINGS_TARGET=owner/current", "GH_TOKEN=", "GITHUB_TOKEN=")
	reads := fixture.reads.Load()
	r, err := executeCLI(binary, fixture.root, fixture.root, summaryEnv, []string{"action", "summary", report, summary, "status-only"}, false)
	data, _ = os.ReadFile(summary)
	if err != nil || r.runErr != nil || fixture.reads.Load() != reads || strings.Contains(string(data), "CANARY") || r.stdout.String() != "plan-outcome=changed\n" {
		t.Fatal(err, r.stderr.String(), string(data))
	}
	run("apply")
	run("apply")
	mu.Lock()
	if len(writes) != 1 || !reflect.DeepEqual(writes[0], map[string]any{"text": nil, "tags": []any{}, "flag": "false"}) {
		t.Fatal(writes)
	}
	mu.Unlock()
	run("export", "--custom-properties")
	data, _ = os.ReadFile(path)
	if !strings.Contains(string(data), "text: null") || !strings.Contains(run("plan"), "No differences") {
		t.Fatal(string(data))
	}
	for _, failure := range []string{"regex", "missing-regex", "regex-error", "404", "schema-denied", "missing-schema", "default", "get-redirect"} {
		mu.Lock()
		mode = failure
		before := len(writes)
		mu.Unlock()
		original, _ := os.ReadFile(path)
		r, err = executeCLI(binary, fixture.work, fixture.root, env, []string{"export", "--custom-properties"}, false)
		after, _ := os.ReadFile(path)
		if err != nil || r.runErr == nil || string(original) != string(after) {
			t.Fatal(failure, err, r.stderr.String())
		}
		r, err = executeCLI(binary, fixture.work, fixture.root, env, []string{"action", "plan", report, "status-only"}, false)
		if err != nil || r.runErr == nil || strings.Contains(r.stderr.String(), "CANARY") {
			t.Fatal("private error", failure, err, r.stderr.String())
		}
		mu.Lock()
		if len(writes) != before || redirected != 0 {
			t.Fatal("read failure mutated/redirected")
		}
		mu.Unlock()
	}
	for _, failure := range []string{"write422", "patch-redirect", "ignored", "unowned"} {
		mu.Lock()
		mode = failure
		state["text"] = "before"
		state["keep"] = "retained"
		before := len(writes)
		mu.Unlock()
		write("{text: null}")
		r, err = executeCLI(binary, fixture.work, fixture.root, env, []string{"apply"}, false)
		if err != nil || r.runErr == nil {
			t.Fatal(failure, err, r.stderr.String())
		}
		mu.Lock()
		if len(writes) != before+1 || redirected != 0 {
			t.Fatal("retry/fallback/redirect")
		}
		mu.Unlock()
	}
	mu.Lock()
	mode = "write422"
	state["text"] = "before"
	events = nil
	mu.Unlock()
	if err := writeSettingsFile(fixture.work, "schemaVersion: 1\nsettings:\n  repository: {description: changed}\n  custom_properties: {text: null}\n"); err != nil {
		t.Fatal(err)
	}
	r, err = executeCLI(binary, fixture.work, fixture.root, env, []string{"apply"}, false)
	if err != nil || r.runErr == nil || !strings.Contains(r.stderr.String(), "earlier successful writes remain applied") {
		t.Fatal(err, r.stderr.String())
	}
	mu.Lock()
	defer mu.Unlock()
	if !reflect.DeepEqual(events, []string{"repository", "properties"}) {
		t.Fatal(events)
	}
}
