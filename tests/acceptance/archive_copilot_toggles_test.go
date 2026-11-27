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
)

func TestArchivedCopilotTogglesCLIReadOnly(t *testing.T) {
	binary := acceptanceBinary(t)
	for _, scenario := range []string{"matching both", "matching reverse", "instructions only", "mcp only", "instructions differ", "mcp differs", "fresh differs", "denied", "missing", "duplicate", "identity drift"} {
		t.Run(scenario, func(t *testing.T) {
			var mu sync.Mutex
			repoReads, toggleReads, writes := 0, 0, 0
			fixture, err := startHTTPFixture(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				w.Header().Set("Content-Type", "application/json")
				if r.Method != http.MethodGet {
					writes++
					w.WriteHeader(400)
					return
				}
				switch r.URL.Path {
				case "/repos/owner/current":
					repoReads++
					id := 42
					if scenario == "identity drift" && repoReads == 2 {
						id = 43
					}
					fmt.Fprintf(w, `{"id":%d,"full_name":"owner/current","archived":true}`, id)
				case "/repos/owner/current/copilot/code-review/settings":
					toggleReads++
					switch scenario {
					case "denied":
						w.WriteHeader(403)
						return
					case "missing":
						fmt.Fprint(w, `{"custom_instructions_enabled":true}`)
						return
					case "duplicate":
						fmt.Fprint(w, `{"custom_instructions_enabled":false,"custom_instructions_enabled":true,"mcp_enabled":false}`)
						return
					case "instructions only":
						fmt.Fprint(w, `{"custom_instructions_enabled":true}`)
						return
					case "mcp only":
						fmt.Fprint(w, `{"mcp_enabled":false}`)
						return
					}
					instructions := scenario != "instructions differ" && scenario != "matching reverse"
					mcp := scenario == "matching reverse" || scenario == "mcp differs" || scenario == "fresh differs" && toggleReads == 2
					fmt.Fprintf(w, `{"custom_instructions_enabled":%t,"mcp_enabled":%t}`, instructions, mcp)
				default:
					t.Errorf("unexpected route %s", r.URL.Path)
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
			controls := "use_custom_instructions: true, allow_copilot_to_use_mcp_tools_when_reviewing_pull_requests: false"
			if scenario == "matching reverse" {
				controls = "use_custom_instructions: false, allow_copilot_to_use_mcp_tools_when_reviewing_pull_requests: true"
			} else if scenario == "instructions only" {
				controls = "use_custom_instructions: true"
			} else if scenario == "mcp only" {
				controls = "allow_copilot_to_use_mcp_tools_when_reviewing_pull_requests: false"
			}
			input := repositoryInput("settings: {general: {danger_zone: {archive_this_repository: true}}, copilot: {code_review: {general: {" + controls + "}}}}\n")
			if err := writeSettingsFile(fixture.work, input); err != nil {
				t.Fatal(err)
			}
			result, err := executeCLI(binary, fixture.work, fixture.root, env, []string{"apply"}, false)
			matching := scenario == "matching both" || scenario == "matching reverse" || scenario == "instructions only" || scenario == "mcp only"
			if err != nil || (result.runErr == nil) != matching {
				t.Fatalf("unexpected CLI result: %v %v %s", err, result.runErr, result.stderr.String())
			}
			mu.Lock()
			defer mu.Unlock()
			if writes != 0 || toggleReads == 0 || matching && (repoReads != 2 || toggleReads != 2) {
				t.Fatalf("unexpected reads/writes: repo=%d toggles=%d writes=%d", repoReads, toggleReads, writes)
			}
			data, err := os.ReadFile(filepath.Join(fixture.work, ".github/settings.yml"))
			if err != nil || string(data) != input {
				t.Fatal("declaration changed", err)
			}
		})
	}
}

func TestArchivedCopilotTogglesExportPlanApply(t *testing.T) {
	var mu sync.Mutex
	writes, toggleReads := 0, 0
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
		if r.Method != http.MethodGet {
			writes++
			w.WriteHeader(400)
			return
		}
		switch r.URL.Path {
		case "/repos/owner/current":
			var base map[string]any
			_ = json.Unmarshal([]byte(planResponse), &base)
			base["id"], base["full_name"], base["archived"] = 42, "owner/current", true
			_ = json.NewEncoder(w).Encode(base)
		case "/repos/owner/current/topics":
			fmt.Fprint(w, `{"names":[]}`)
		case "/repos/owner/current/labels":
			fmt.Fprint(w, `[]`)
		case "/repos/owner/current/copilot/code-review/settings":
			toggleReads++
			fmt.Fprint(w, `{"custom_instructions_enabled":true,"mcp_enabled":false,"severity_filter":"no_filter","review_effort":{"configured":"balanced","effective":"balanced"},"available_review_efforts":["lite","balanced"]}`)
		default:
			t.Errorf("unexpected route %s", r.URL.Path)
			w.WriteHeader(400)
		}
	}, false, "/copilot/code-review/settings", "/labels", "/topics")
	if err != nil {
		t.Fatal(err)
	}
	defer fixture.close()
	env := exportFixtureEnv(fixture.cfg)
	if err := setupOrdinaryOrigin(fixture.work, env); err != nil {
		t.Fatal(err)
	}
	run := func(command string) string {
		t.Helper()
		result, err := executeCLI(acceptanceBinary(t), fixture.work, fixture.root, env, []string{command}, false)
		if err != nil || result.runErr != nil || hasNonInfoDiagnostic(result.stderr.String()) {
			t.Fatalf("%s: %v %v %s", command, err, result.runErr, result.stderr.String())
		}
		return result.stdout.String()
	}
	// There is no prior settings file: unavailable independent export families
	// cannot contribute retained intent to this successful path.
	run("export")
	path := filepath.Join(fixture.work, ".github/settings.yml")
	data, err := os.ReadFile(path)
	if err != nil || !strings.Contains(string(data), "archive_this_repository: true") || !strings.Contains(string(data), "use_custom_instructions: true") || !strings.Contains(string(data), "allow_copilot_to_use_mcp_tools_when_reviewing_pull_requests: false") {
		t.Fatalf("missing observed exported values: %v %s", err, data)
	}
	if output := run("plan"); output != "No differences for managed repository settings.\n" {
		t.Fatal(output)
	}
	run("apply")
	after, err := os.ReadFile(path)
	if err != nil || string(after) != string(data) {
		t.Fatal("apply changed exported declaration", err)
	}
	mu.Lock()
	defer mu.Unlock()
	if writes != 0 || toggleReads < 5 {
		t.Fatalf("unexpected writes or missing observations: writes=%d reads=%d", writes, toggleReads)
	}
}
