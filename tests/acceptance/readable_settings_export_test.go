//go:build acceptance

package acceptance

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"go.yaml.in/yaml/v4"
)

// A late inventory page must not become [] or a partial declared collection.
// Later independent families must still be exported, without upstream secrets.
func TestReadableSettingsExportLateVariablePageRetainsIntent(t *testing.T) {
	const sensitive = "upstream-credential-must-not-be-logged"
	pages, agents := 0, 0
	fixture, err := startHTTPFixture(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method != "GET" && !isMergeMethodsFixtureRead(r) && !isPRAccessFixtureRead(r) && !isIssueCreationPolicyFixtureRead(r) && !isSponsorshipsFixtureRead(r) {
			t.Error("export attempted mutation", r.Method, r.URL)
			w.WriteHeader(500)
			return
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
			fmt.Fprint(w, planResponse)
		case "/repos/owner/current/topics":
			fmt.Fprint(w, `{"names":[]}`)
		case "/repos/owner/current/labels":
			fmt.Fprint(w, `[]`)
		case "/repos/owner/current/environments":
			fmt.Fprint(w, `{"total_count":0,"environments":[]}`)
		case "/repos/owner/current/agents/variables":
			agents++
			fmt.Fprint(w, `{"total_count":0,"variables":[]}`)
		case "/repos/owner/current/actions/variables":
			pages++
			if r.URL.Query().Get("per_page") != "30" {
				t.Error("wrong page size")
			}
			if r.URL.Query().Get("page") == "2" {
				w.WriteHeader(403)
				json.NewEncoder(w).Encode(map[string]string{"message": sensitive})
				return
			}
			entries := []map[string]string{}
			for i := range 30 {
				entries = append(entries, map[string]string{"name": fmt.Sprintf("ITEM_%02d", i), "value": "acquired-but-incomplete"})
			}
			json.NewEncoder(w).Encode(map[string]any{"total_count": 31, "variables": entries})
		default:
			t.Error("unexpected owned API", r.URL)
			w.WriteHeader(500)
		}
	}, false, "/environments", "/actions/variables", "/agents/variables")
	if err != nil {
		t.Fatal(err)
	}
	defer fixture.close()
	env := exportFixtureEnv(fixture.cfg)
	if err := setupOrdinaryOrigin(fixture.work, env); err != nil {
		t.Fatal(err)
	}
	binary := acceptanceBinary(t)
	path := filepath.Join(fixture.work, ".github/settings.yml")
	for _, existing := range []bool{false, true} {
		if existing {
			if err := writeSettingsFile(fixture.work, "version: daiksud/gh-settings@v0\nrepository: {settings: {actions: {variables: [{name: KEEP, value: old-intent}]}}}\n"); err != nil {
				t.Fatal(err)
			}
		}
		result, err := executeCLI(binary, fixture.work, fixture.root, env, []string{"export"}, false)
		if err != nil || result.runErr != nil || result.stdout.Len() != 0 || !strings.Contains(result.stderr.String(), "info: export repository.settings.actions.variables:") || strings.Contains(result.stderr.String(), sensitive) || strings.Contains(result.stderr.String(), "acquired-but-incomplete") {
			t.Fatal("inventory failure was not bounded/continued", err, result.stderr.String())
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var output struct {
			Repository struct {
				Settings struct {
					Actions struct {
						Variables *[]map[string]string `yaml:"variables"`
					} `yaml:"actions"`
					Agents struct {
						Variables *[]map[string]string `yaml:"variables"`
					} `yaml:"agents"`
				} `yaml:"settings"`
			} `yaml:"repository"`
		}
		if err := yaml.Unmarshal(raw, &output); err != nil {
			t.Fatal(err)
		}
		actions := output.Repository.Settings.Actions.Variables
		if existing {
			if actions == nil || len(*actions) != 1 || (*actions)[0]["name"] != "KEEP" || (*actions)[0]["value"] != "old-intent" {
				t.Fatal("prior collection erased")
			}
		} else if actions != nil {
			t.Fatal("partial inventory became declaration")
		}
		if output.Repository.Settings.Agents.Variables == nil || len(*output.Repository.Settings.Agents.Variables) != 0 {
			t.Fatal("later complete empty inventory lost")
		}
		validated, err := executeCLI(binary, fixture.work, fixture.root, env, []string{"validate"}, false)
		if err != nil || validated.runErr != nil {
			t.Fatal("output invalid", err, validated.stderr.String())
		}
	}
	if pages != 4 || agents != 2 {
		t.Fatal("pagination/continuation", pages, agents)
	}
}

// Valid partial declarations must not reject unrelated acquired families or
// become fabricated complete observations when their own read is unavailable.
func TestReadableSettingsExportRetainsPartialActions(t *testing.T) {
	for _, general := range []string{
		`{workflow_permissions: {allow_github_actions_to_create_and_approve_pull_requests: false}}`,
		`{workflow_permissions: {default_permissions: read_repository_contents_and_packages_permissions}}`,
		`{actions_permissions: {require_actions_to_be_pinned_to_a_full_length_commit_sha: false}}`,
		`{actions_permissions: {allow_actions_created_by_github: false}}`,
		`{actions_permissions: {policy: disable_actions}}`,
	} {
		t.Run(general, func(t *testing.T) {
			fixture, err := startHTTPFixture(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if servePRAccessFixtureRead(w, r) || serveIssueCreationPolicyFixtureRead(w, r) || serveSponsorshipsFixtureRead(w, r) {
					return
				}
				if isMergeMethodsFixtureRead(r) {
					fmt.Fprint(w, mergeMethodsFixtureResponse)
					return
				}
				if r.Method != "GET" {
					t.Error("unexpected mutation", r.Method, r.URL)
					w.WriteHeader(500)
					return
				}
				switch r.URL.Path {
				case "/repos/owner/current":
					fmt.Fprint(w, planResponse)
				case "/repos/owner/current/topics":
					fmt.Fprint(w, `{"names":["fresh"]}`)
				case "/repos/owner/current/labels":
					fmt.Fprint(w, `[]`)
				case "/repos/owner/current/rulesets":
					fmt.Fprint(w, `[]`)
				case "/repos/owner/current/environments":
					fmt.Fprint(w, `{"total_count":0,"environments":[]}`)
				case "/repos/owner/current/agents/variables":
					fmt.Fprint(w, `{"total_count":0,"variables":[]}`)
				default:
					t.Error("unexpected owned API", r.URL)
					w.WriteHeader(500)
				}
			}, false, "/rulesets", "/environments", "/agents/variables")
			if err != nil {
				t.Fatal(err)
			}
			defer fixture.close()
			env := exportFixtureEnv(fixture.cfg)
			if err := setupOrdinaryOrigin(fixture.work, env); err != nil {
				t.Fatal(err)
			}
			input := "version: daiksud/gh-settings@v0\nrepository: {settings: {actions: {general: " + general + "}}}\n"
			if err := writeSettingsFile(fixture.work, input); err != nil {
				t.Fatal(err)
			}
			var prior map[string]any
			if err := yaml.Unmarshal([]byte(input), &prior); err != nil {
				t.Fatal(err)
			}
			want := prior["repository"].(map[string]any)["settings"].(map[string]any)["actions"].(map[string]any)["general"]
			binary := acceptanceBinary(t)
			for range 2 {
				result, err := executeCLI(binary, fixture.work, fixture.root, env, []string{"export"}, false)
				if err != nil || result.runErr != nil {
					t.Fatal("partial intent blocked export", err, result.stderr.String())
				}
				if strings.Contains(result.stderr.String(), "info: export repository.about.topics:") || strings.Contains(result.stderr.String(), "info: export repository.settings.rulesets:") {
					t.Fatal("retained intent rejected an unrelated successful read", result.stderr.String())
				}
				data, err := os.ReadFile(filepath.Join(fixture.work, ".github/settings.yml"))
				if err != nil {
					t.Fatal(err)
				}
				var output map[string]any
				if err := yaml.Unmarshal(data, &output); err != nil {
					t.Fatal(err)
				}
				repo := output["repository"].(map[string]any)
				settings := repo["settings"].(map[string]any)
				got := settings["actions"].(map[string]any)["general"]
				if !reflect.DeepEqual(got, want) {
					t.Fatalf("partial controls changed: %#v; want %#v", got, want)
				}
				if !reflect.DeepEqual(repo["about"].(map[string]any)["topics"], []any{"fresh"}) || !reflect.DeepEqual(repo["labels"], []any{}) || !reflect.DeepEqual(settings["rulesets"], []any{}) || !reflect.DeepEqual(settings["environments"], []any{}) || !reflect.DeepEqual(settings["agents"].(map[string]any)["variables"], []any{}) {
					t.Fatal("successful independent acquisition was discarded")
				}
				validated, err := executeCLI(binary, fixture.work, fixture.root, env, []string{"validate"}, false)
				if err != nil || validated.runErr != nil {
					t.Fatal("invalid output", err, validated.stderr.String())
				}
			}
		})
	}
}
