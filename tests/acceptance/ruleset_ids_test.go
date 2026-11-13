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

const newRulesetInput = `# consumer configuration
version: daiksud/gh-settings@v0
repository:
  settings:
    rulesets:
      - ruleset_name: new # preserve intent
        enforcement_status: active
        target_branches: {include_all_branches: true}
`

func TestRulesetsCreatedIDWritebackModesAndRepeatApply(t *testing.T) {
	for _, mode := range []string{"writeback", "no-writeback", "actions"} {
		t.Run(mode, func(t *testing.T) {
			var mu sync.Mutex
			var created map[string]any
			posts, details := 0, 0
			fixture, err := startHTTPFixture(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				w.Header().Set("Content-Type", "application/json")
				switch {
				case r.Method == "GET" && r.URL.Path == "/repos/owner/current/rulesets":
					if created == nil {
						fmt.Fprint(w, `[]`)
					} else {
						fmt.Fprint(w, `[{"id":17,"name":"new","source_type":"Repository","source":"owner/current"}]`)
					}
				case r.Method == "GET" && r.URL.Path == "/repos/owner/current/rulesets/17" && created != nil:
					details++
					if r.URL.RawQuery != "includes_parents=false" {
						t.Error(r.URL)
					}
					json.NewEncoder(w).Encode(created)
				case r.Method == "POST" && r.URL.Path == "/repos/owner/current/rulesets":
					posts++
					if err := json.NewDecoder(r.Body).Decode(&created); err != nil {
						t.Error(err)
					}
					if len(created) != 6 || created["id"] != nil || created["name"] != "new" || created["target"] != "branch" || len(created["rules"].([]any)) != 0 || len(created["bypass_actors"].([]any)) != 0 {
						t.Error("unexpected complete create payload", created)
					}
					created["id"], created["source_type"], created["source"] = 17, "Repository", "owner/current"
					w.WriteHeader(201)
					json.NewEncoder(w).Encode(created)
				default:
					t.Error("unexpected adoption/update/delete", r.Method, r.URL)
					w.WriteHeader(400)
				}
			}, false, "/rulesets", "/rulesets/17")
			if err != nil {
				t.Fatal(err)
			}
			defer fixture.close()
			env := labelIDFixtureEnv(fixture.cfg, mode == "actions")
			if err := setupOrdinaryOrigin(fixture.work, env); err != nil {
				t.Fatal(err)
			}
			if err := writeSettingsFile(fixture.work, newRulesetInput); err != nil {
				t.Fatal(err)
			}
			args := []string{"apply"}
			if mode == "no-writeback" {
				args = append(args, "--no-writeback")
			}
			first, err := executeCLI(acceptanceBinary(t), fixture.work, fixture.root, env, args, false)
			if err != nil || first.runErr != nil || hasNonInfoDiagnostic(first.stderr.String()) || !strings.Contains(first.stdout.String(), `Created repository.settings.rulesets "new": id 17`) {
				t.Fatal(first, err)
			}
			mu.Lock()
			firstDetails := details
			mu.Unlock()
			if firstDetails != 1 {
				t.Fatalf("created ID was not independently verified: detail reads=%d", firstDetails)
			}
			data, err := os.ReadFile(filepath.Join(fixture.work, ".github/settings.yml"))
			if err != nil {
				t.Fatal(err)
			}
			if mode == "writeback" {
				for _, want := range []string{"id: 17", "# consumer configuration", "# preserve intent"} {
					if !strings.Contains(string(data), want) {
						t.Fatal("writeback lost value/comment", string(data))
					}
				}
			} else if string(data) != newRulesetInput {
				t.Fatal("disabled writeback modified file", string(data))
			}
			second, err := executeCLI(acceptanceBinary(t), fixture.work, fixture.root, env, args, false)
			if err != nil {
				t.Fatal(err)
			}
			if mode == "writeback" {
				if second.runErr != nil || hasNonInfoDiagnostic(second.stderr.String()) {
					t.Fatal(second)
				}
			} else if second.runErr == nil || !strings.Contains(withoutExportInfo(second.stderr.String()), "specify id: 17") {
				t.Fatal("name collision adopted", second)
			}
			mu.Lock()
			defer mu.Unlock()
			if posts != 1 {
				t.Fatal("repeat create", posts)
			}
		})
	}
}

func TestRulesetsPartialCreationFailureRetainsRecoverableID(t *testing.T) {
	for _, mode := range []string{"local conflict", "later remote failure", "unverified response", "denied readback", "mismatching readback", "later readback failure"} {
		t.Run(mode, func(t *testing.T) {
			var mu sync.Mutex
			var path, input string
			posts := 0
			created := map[string]map[string]any{}
			fixture, err := startHTTPFixture(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				w.Header().Set("Content-Type", "application/json")
				if r.Method == "GET" && r.URL.Path == "/repos/owner/current/rulesets" {
					fmt.Fprint(w, `[]`)
					return
				}
				if r.Method == "GET" && strings.HasPrefix(r.URL.Path, "/repos/owner/current/rulesets/") {
					id := strings.TrimPrefix(r.URL.Path, "/repos/owner/current/rulesets/")
					state, ok := created[id]
					if !ok || r.URL.RawQuery != "includes_parents=false" {
						t.Error("unexpected detail read", r.URL)
						w.WriteHeader(404)
						return
					}
					if mode == "denied readback" {
						w.WriteHeader(403)
						fmt.Fprint(w, `{"message":"fixture denied"}`)
						return
					}
					observed := map[string]any{}
					for key, value := range state {
						observed[key] = value
					}
					if mode == "mismatching readback" || mode == "later readback failure" && id == "28" {
						observed["enforcement"] = "disabled"
					}
					json.NewEncoder(w).Encode(observed)
					return
				}
				if r.Method != "POST" || r.URL.Path != "/repos/owner/current/rulesets" {
					t.Error(r.Method, r.URL)
					w.WriteHeader(400)
					return
				}
				posts++
				var payload map[string]any
				if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
					t.Error(err)
				}
				if payload["name"] == "b" && mode != "later readback failure" {
					w.WriteHeader(403)
					fmt.Fprint(w, `{"message":"fixture denied"}`)
					return
				}
				if mode == "local conflict" {
					if err := os.WriteFile(path, []byte(input+"# concurrent edit\n"), 0600); err != nil {
						t.Error(err)
					}
				}
				id := 27
				if payload["name"] == "b" {
					id = 28
				}
				payload["id"], payload["source_type"], payload["source"] = id, "Repository", "owner/current"
				created[fmt.Sprint(id)] = payload
				if mode == "unverified response" {
					payload["enforcement"] = "disabled"
				}
				w.WriteHeader(201)
				json.NewEncoder(w).Encode(payload)
			}, false, "/rulesets", "/rulesets/17")
			if err != nil {
				t.Fatal(err)
			}
			defer fixture.close()
			env := labelIDFixtureEnv(fixture.cfg, false)
			if err := setupOrdinaryOrigin(fixture.work, env); err != nil {
				t.Fatal(err)
			}
			input = repositoryInput("settings:\n  rulesets:\n    - {ruleset_name: a, enforcement_status: active, target_branches: {include_all_branches: true}}\n    - {ruleset_name: b, enforcement_status: active, target_branches: {include_all_branches: true}}\n")
			if mode == "later readback failure" {
				input += "      - {ruleset_name: c, enforcement_status: active, target_branches: {include_all_branches: true}}\n"
			}
			path = filepath.Join(fixture.work, ".github/settings.yml")
			if err := writeSettingsFile(fixture.work, input); err != nil {
				t.Fatal(err)
			}
			result, err := executeCLI(acceptanceBinary(t), fixture.work, fixture.root, env, []string{"apply"}, false)
			if err != nil || result.runErr == nil || !strings.Contains(withoutExportInfo(result.stderr.String()), "id 27") {
				t.Fatal(result, err)
			}
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			switch mode {
			case "local conflict":
				if string(data) != input+"# concurrent edit\n" || !strings.Contains(withoutExportInfo(result.stderr.String()), "local ID writeback failed") {
					t.Fatal(string(data), result)
				}
			case "later readback failure":
				if !strings.Contains(string(data), "id: 27") || strings.Contains(string(data), "id: 28") || !strings.Contains(result.stderr.String(), "id 28") || result.stdout.Len() != 0 {
					t.Fatal("failed readback lost recovery or recorded unverified ID", string(data), result)
				}
			case "later remote failure":
				if !strings.Contains(string(data), "id: 27") || !strings.Contains(withoutExportInfo(result.stderr.String()), "403") {
					t.Fatal(string(data), result)
				}
			case "unverified response", "denied readback", "mismatching readback":
				if string(data) != input || !strings.Contains(withoutExportInfo(result.stderr.String()), "writes may already have applied") {
					t.Fatal("unverified response recorded as success", string(data), result)
				}
			}
			mu.Lock()
			defer mu.Unlock()
			want := 1
			if mode == "later remote failure" || mode == "later readback failure" {
				want = 2
			}
			if posts != want {
				t.Fatal("later mutation/retry after failure", posts, want)
			}
		})
	}
}
