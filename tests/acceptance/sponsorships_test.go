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

func TestSponsorshipsExportPlanApplyAndFundingIsolation(t *testing.T) {
	binary := acceptanceBinary(t)
	var mu sync.Mutex
	enabled := false
	var writes []map[string]any
	fixture, err := startHTTPFixture(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		if isSponsorshipsFixtureRead(r) {
			fmt.Fprintf(w, `{"data":{"repository":{"hasSponsorshipsEnabled":%t}}}`, enabled)
			return
		}
		if servePRAccessFixtureRead(w, r) || serveIssueCreationPolicyFixtureRead(w, r) {
			return
		}
		if isMergeMethodsFixtureRead(r) {
			fmt.Fprint(w, mergeMethodsFixtureResponse)
			return
		}
		if r.Method == http.MethodGet {
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
			return
		}
		if r.Method != http.MethodPost || r.URL.Path != "/graphql" {
			t.Error("unexpected write", r.Method, r.URL)
			w.WriteHeader(400)
			return
		}
		var request struct {
			Query     string
			Variables map[string]any
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
			w.WriteHeader(400)
			return
		}
		if strings.HasPrefix(request.Query, "query") {
			if strings.Join(strings.Fields(request.Query), " ") != `query($owner: String!, $name: String!) { repository(owner: $owner, name: $name, followRenames: false) { id nameWithOwner } }` || !reflect.DeepEqual(request.Variables, map[string]any{"owner": "owner", "name": "current"}) {
				t.Error("unexpected lookup", request)
			}
			fmt.Fprint(w, prIdentityResponse)
			return
		}
		input, ok := request.Variables["input"].(map[string]any)
		value, boolean := input["hasSponsorshipsEnabled"].(bool)
		if !ok || !boolean || len(input) != 2 || input["repositoryId"] != "opaque-id" || strings.Contains(request.Query, "funding") {
			t.Error("over-broad mutation", request)
			w.WriteHeader(400)
			return
		}
		enabled = value
		writes = append(writes, input)
		fmt.Fprintf(w, `{"data":{"updateRepository":{"repository":{"id":"opaque-id","nameWithOwner":"owner/current","hasSponsorshipsEnabled":%t}}}}`, enabled)
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
		result, err := executeCLI(binary, fixture.work, fixture.root, env, args, false)
		if err != nil || result.runErr != nil || result.stderr.Len() != 0 {
			t.Fatal(args, err, result.stderr.String())
		}
		return result.stdout.String()
	}
	funding := filepath.Join(fixture.work, ".github/FUNDING.yml")
	run("export")
	run("validate")
	if got := run("plan"); got != "No differences for managed repository settings.\n" {
		t.Fatal(got)
	}
	exported, err := os.ReadFile(filepath.Join(fixture.work, ".github/settings.yml"))
	if err != nil || !strings.Contains(string(exported), "sponsorships: false") {
		t.Fatal(string(exported), err)
	}
	config := generalInput("features: {sponsorships: true}\n")
	if err := writeSettingsFile(fixture.work, config); err != nil {
		t.Fatal(err)
	}
	want := "repository.settings.general.features.sponsorships: false → true\n"
	if got := run("plan"); got != want {
		t.Fatal(got)
	}
	report := filepath.Join(fixture.root, "sponsorship-plan.json")
	run("action", "plan", report)
	data, err := os.ReadFile(report)
	if err != nil {
		t.Fatal(err)
	}
	var actual map[string]any
	if err := json.Unmarshal(data, &actual); err != nil {
		t.Fatal(err)
	}
	expected := map[string]any{"outcome": "changed", "target": "owner/current", "managed": float64(1), "changes": []any{map[string]any{"field": "repository.has_sponsorships", "before": false, "after": true}}}
	if !reflect.DeepEqual(actual, expected) {
		t.Fatal(string(data))
	}
	summary := filepath.Join(fixture.root, "sponsorship-summary.md")
	summaryEnv := append(fixtureEnv(fixture.cfg), "SETTINGS_ACTOR=user", "SETTINGS_TRIGGERING_ACTOR=user", "SETTINGS_EVENT=push", "SETTINGS_RUN_ID=123", "SETTINGS_RUN_ATTEMPT=1", "SETTINGS_WORKFLOW_REF=owner/current/.github/workflows/settings.yml@refs/heads/main", "SETTINGS_WORKFLOW_SHA="+strings.Repeat("a", 40), "SETTINGS_CONFIG_SHA="+strings.Repeat("b", 40), "SETTINGS_TARGET=owner/current", "SETTINGS_CLI_VERSION=dev", "SETTINGS_CLI_SHA256="+strings.Repeat("c", 64))
	reads := fixture.reads.Load()
	result, err := executeCLI(binary, fixture.root, fixture.root, summaryEnv, []string{"action", "summary", report, summary}, false)
	if err != nil || result.runErr != nil || reads != fixture.reads.Load() {
		t.Fatal(err, result.stderr.String())
	}
	data, err = os.ReadFile(summary)
	if err != nil || !strings.Contains(html.UnescapeString(string(data)), strings.TrimSpace(want)) {
		t.Fatal(string(data), err)
	}
	run("apply")
	if _, err := os.Lstat(funding); !os.IsNotExist(err) {
		t.Fatal("enabling synthesized funding destinations", err)
	}
	if got := run("plan"); got != "No differences for managed repository settings.\n" {
		t.Fatal(got)
	}
	// An existing funding file is outside the declaration, even while disabling.
	fundingBytes := []byte("custom: https://example.invalid/maintainer\n")
	if err := os.WriteFile(funding, fundingBytes, 0600); err != nil {
		t.Fatal(err)
	}
	run("apply")
	config = generalInput("features: {sponsorships: false}\n")
	if err := writeSettingsFile(fixture.work, config); err != nil {
		t.Fatal(err)
	}
	run("apply")
	if err := checkSettingsUnchanged(fixture.work, config); err != nil {
		t.Fatal(err)
	}
	data, err = os.ReadFile(funding)
	if err != nil || string(data) != string(fundingBytes) {
		t.Fatal("funding file changed", err)
	}
	mu.Lock()
	defer mu.Unlock()
	if enabled || !reflect.DeepEqual(writes, []map[string]any{
		{"repositoryId": "opaque-id", "hasSponsorshipsEnabled": true},
		{"repositoryId": "opaque-id", "hasSponsorshipsEnabled": true},
		{"repositoryId": "opaque-id", "hasSponsorshipsEnabled": false},
	}) {
		t.Fatal(enabled, writes)
	}
}

func TestSponsorshipsFailuresPreserveFilesAndStopLaterWrites(t *testing.T) {
	binary := acceptanceBinary(t)
	for _, test := range []struct {
		name, command, fields, want string
		requests                    int
	}{
		{"missing read", "plan", "features: {sponsorships: false}\n", "hasSponsorshipsEnabled", 1},
		{"partial read", "plan", "features: {sponsorships: true}\n", "sponsorship read denied", 1},
		{"incomplete export", "export", "", "hasSponsorshipsEnabled", 5},
		{"wrong target", "apply", "features: {sponsorships: true}\n", "target does not match", 1},
		{"ignored mutation", "apply", "features: {sponsorships: true}\n", "did not retain requested hasSponsorshipsEnabled", 2},
		{"partial mutation", "apply", "features: {sponsorships: true}\n", "sponsorship write denied", 2},
		{"redirect", "apply", "features: {sponsorships: true}\n", "307", 2},
		{"earlier REST write", "apply", "features: {issues: false, sponsorships: true}\n", "earlier repository writes remain applied", 3},
		{"invalid local", "apply", "features: {sponsorships: 'false'}\n", "must be a boolean", 0},
	} {
		t.Run(test.name, func(t *testing.T) {
			var mu sync.Mutex
			var requests []string
			restWritten := false
			fixture, err := startHTTPFixture(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				requests = append(requests, r.Method+" "+r.URL.Path)
				w.Header().Set("Content-Type", "application/json")
				if isSponsorshipsFixtureRead(r) {
					if test.name == "partial read" {
						fmt.Fprint(w, `{"data":{"repository":{"hasSponsorshipsEnabled":false}},"errors":[{"message":"sponsorship read denied"}]}`)
					} else {
						fmt.Fprint(w, `{"data":{"repository":{}}}`)
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
				if r.Method == http.MethodGet && r.URL.Path == "/repos/owner/current" {
					fmt.Fprint(w, planResponse)
					return
				}
				if r.Method == http.MethodPatch && r.URL.Path == "/repos/owner/current" && test.name == "earlier REST write" {
					var input map[string]any
					if err := json.NewDecoder(r.Body).Decode(&input); err != nil || !reflect.DeepEqual(input, map[string]any{"has_issues": false}) {
						t.Error("unexpected REST payload", input, err)
					}
					restWritten = true
					fmt.Fprint(w, `{}`)
					return
				}
				if r.Method != http.MethodPost || r.URL.Path != "/graphql" {
					t.Error("unexpected later operation", r.Method, r.URL)
					w.WriteHeader(400)
					return
				}
				var request struct{ Query string }
				if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
					t.Error(err)
				}
				if strings.HasPrefix(request.Query, "query") {
					if test.name == "wrong target" {
						fmt.Fprint(w, `{"data":{"repository":{"id":"opaque-id","nameWithOwner":"owner/other"}}}`)
					} else {
						fmt.Fprint(w, prIdentityResponse)
					}
					return
				}
				switch test.name {
				case "ignored mutation":
					fmt.Fprint(w, `{"data":{"updateRepository":{"repository":{"id":"opaque-id","nameWithOwner":"owner/current","hasSponsorshipsEnabled":false}}}}`)
				case "redirect":
					w.Header().Set("Location", "https://api.github.com/repos/owner/moved")
					w.WriteHeader(http.StatusTemporaryRedirect)
				default:
					fmt.Fprint(w, `{"data":{"updateRepository":{"repository":{"id":"opaque-id","nameWithOwner":"owner/current","hasSponsorshipsEnabled":true}}},"errors":[{"message":"sponsorship write denied"}]}`)
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
			config := repositoryInput("about: {topics: [must-not-be-written]}\nsettings:\n  general:\n" + indentInput(test.fields, 4))
			if err := writeSettingsFile(fixture.work, config); err != nil {
				t.Fatal(err)
			}
			result, err := executeCLI(binary, fixture.work, fixture.root, env, []string{test.command}, false)
			if err != nil || result.runErr == nil || result.stdout.Len() != 0 || !strings.Contains(result.stderr.String(), test.want) {
				t.Fatalf("error=%v run=%v stdout=%q stderr=%q", err, result.runErr, result.stdout.String(), result.stderr.String())
			}
			if err := checkSettingsUnchanged(fixture.work, config); err != nil {
				t.Fatal(err)
			}
			mu.Lock()
			defer mu.Unlock()
			if len(requests) != test.requests || restWritten != (test.name == "earlier REST write") {
				t.Fatal(requests, restWritten)
			}
		})
	}
}
