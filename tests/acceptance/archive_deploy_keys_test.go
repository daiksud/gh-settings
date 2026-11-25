//go:build acceptance

package acceptance

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// All key material is deliberately synthetic and must not appear in diagnostics.
func TestArchivedDeployKeysCLI(t *testing.T) {
	const upstream = "UPSTREAM_DEPLOY_KEY_CANARY"
	for _, actions := range []bool{false, true} {
		for _, scenario := range []string{"matching", "title differs", "key differs", "permission differs", "missing", "denied", "duplicate member", "idless", "prune", "repo drift", "owner drift", "archive drift", "inventory drift", "mixed"} {
			t.Run(fmt.Sprintf("actions=%t/%s", actions, scenario), func(t *testing.T) {
				var mu sync.Mutex
				reads, details, repoReads, variableReads, writes := 0, 0, 0, 0, 0
				f, err := startHTTPFixture(func(w http.ResponseWriter, r *http.Request) {
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
						id, owner, archived := 1, 2, true
						if repoReads >= 3 {
							if scenario == "repo drift" {
								id = 3
							}
							if scenario == "owner drift" {
								owner = 4
							}
							if scenario == "archive drift" {
								archived = false
							}
						}
						fmt.Fprintf(w, `{"id":%d,"full_name":"owner/current","private":false,"visibility":"public","archived":%t,"owner":{"id":%d,"login":"owner","type":"User"}}`, id, archived, owner)
					case "/repos/owner/current/keys", "/repos/owner/current/keys/7":
						detail := strings.HasSuffix(r.URL.Path, "/7")
						if detail {
							details++
						} else {
							reads++
						}
						if scenario == "denied" {
							w.WriteHeader(403)
							fmt.Fprintf(w, `{"message":%q}`, upstream)
							return
						}
						if scenario == "missing" {
							fmt.Fprint(w, `[]`)
							return
						}
						if scenario == "duplicate member" {
							fmt.Fprintf(w, `[{"id":8,"id":7,"title":"build","key":%q,"read_only":true,"verified":false,"created_at":"2026-11-14T00:00:00Z","url":"https://api.github.com/repos/owner/current/keys/7"}]`, deployKeyOne)
							return
						}
						title, key, readOnly := "build", deployKeyOne, true
						if scenario == "title differs" || (scenario == "inventory drift" && !detail && reads >= 2) {
							title = "other"
						}
						if scenario == "key differs" {
							key = deployKeyTwo
						}
						if scenario == "permission differs" {
							readOnly = false
						}
						row := fmt.Sprintf(`{"id":7,"title":%q,"key":%q,"read_only":%t,"verified":false,"enabled":true,"created_at":"2026-11-14T00:00:00Z","url":"https://api.github.com/repos/owner/current/keys/7","added_by":"%s"}`, title, key, readOnly, upstream)
						if detail {
							fmt.Fprint(w, row)
						} else {
							fmt.Fprintf(w, "[%s]", row)
						}
					case "/repos/owner/current/actions/variables/FLAG":
						variableReads++
						fmt.Fprint(w, `{"name":"FLAG","value":"FIXTURE_VALUE"}`)
					default:
						t.Error("unexpected route", r.URL.Path)
						w.WriteHeader(400)
					}
				}, false, "/keys")
				if err != nil {
					t.Fatal(err)
				}
				defer f.close()
				env := append(labelIDFixtureEnv(f.cfg, actions), "GH_DEBUG=api")
				if err := setupOrdinaryOrigin(f.work, env); err != nil {
					t.Fatal(err)
				}
				id := "id: 7, "
				if scenario == "idless" {
					id = ""
				}
				selected := "deploy_keys: [{" + id + "title: build, key: '" + deployKeyOne + " source-comment', allow_write_access: false}]"
				if scenario == "mixed" {
					selected += ", secrets_and_variables: {actions: {variables: [{name: FLAG, value: FIXTURE_VALUE}]}}"
				}
				input := repositoryInput("settings: {general: {danger_zone: {archive_this_repository: true}}, " + selected + "}\n")
				if err := writeSettingsFile(f.work, input); err != nil {
					t.Fatal(err)
				}
				if scenario == "key differs" {
					reportPath := filepath.Join(f.root, "mismatch-plan.json")
					planned, planErr := executeCLI(acceptanceBinary(t), f.work, f.root, env, []string{"action", "plan", reportPath}, false)
					if planErr != nil {
						t.Fatal("plan setup failed", planErr)
					}
					report, readErr := os.ReadFile(reportPath)
					if readErr != nil {
						t.Error("mismatch report unavailable", readErr)
					}
					public := planned.stdout.String() + planned.stderr.String() + string(report)
					for _, sensitive := range []string{deployKeyOne, deployKeyTwo, upstream} {
						if strings.Contains(public, sensitive) {
							t.Error("mismatch report or output leaked key material")
						}
					}
				}
				args := []string{"apply"}
				if scenario == "prune" {
					args = append(args, "--prune", "deploy-keys")
				}
				result, err := executeCLI(acceptanceBinary(t), f.work, f.root, env, args, false)
				if err != nil {
					t.Fatal(err)
				}
				success := scenario == "matching" || scenario == "mixed"
				if (result.runErr == nil) != success {
					t.Error("unexpected command status", result.runErr)
				}
				message := result.stdout.String() + result.stderr.String()
				for _, sensitive := range []string{deployKeyOne, deployKeyTwo, upstream} {
					if strings.Contains(message, sensitive) {
						t.Error("key or upstream metadata leaked")
					}
				}
				data, err := os.ReadFile(filepath.Join(f.work, ".github/settings.yml"))
				if err != nil || string(data) != input {
					t.Error("declaration changed", err)
				}
				mu.Lock()
				defer mu.Unlock()
				if writes != 0 {
					t.Error("unexpected remote mutation", writes)
				}
				if success && reads < 2 {
					t.Error("missing independent complete inventory read", reads)
				}
				if scenario == "mixed" && variableReads == 0 {
					t.Error("mixed variable not observed")
				}
				if (scenario == "repo drift" || scenario == "owner drift" || scenario == "archive drift") && repoReads < 3 {
					t.Error("target drift was not observed", repoReads)
				}
			})
		}
	}
}

func TestArchivedDeployKeysExportPlanApply(t *testing.T) {
	var mu sync.Mutex
	writes, inventory, details := 0, 0, 0
	exporting := true
	f, err := startHTTPFixture(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		if exporting && r.Method == http.MethodPost && r.URL.Path == "/graphql" {
			body, _ := io.ReadAll(r.Body)
			if strings.Contains(strings.ToLower(string(body)), "mutation") {
				writes++
			}
			w.WriteHeader(403)
			return
		}
		if r.Method != http.MethodGet {
			writes++
			w.WriteHeader(400)
			return
		}
		switch r.URL.Path {
		case "/repos/owner/current":
			fmt.Fprint(w, `{"id":1,"full_name":"owner/current","private":false,"visibility":"public","archived":true,"owner":{"id":2,"login":"owner","type":"User"}}`)
		case "/repos/owner/current/keys", "/repos/owner/current/keys/7":
			row := fmt.Sprintf(`{"id":7,"title":"build","key":%q,"read_only":true,"verified":false,"enabled":true,"created_at":"2026-11-14T00:00:00Z","url":"https://api.github.com/repos/owner/current/keys/7"}`, deployKeyOne)
			if strings.HasSuffix(r.URL.Path, "/7") {
				details++
				fmt.Fprint(w, row)
			} else {
				inventory++
				fmt.Fprintf(w, "[%s]", row)
			}
		default:
			w.WriteHeader(403)
		}
	}, false, "/keys")
	if err != nil {
		t.Fatal(err)
	}
	defer f.close()
	env := append(labelIDFixtureEnv(f.cfg, false), "GH_DEBUG=api")
	if err := setupOrdinaryOrigin(f.work, env); err != nil {
		t.Fatal(err)
	}
	binary := acceptanceBinary(t)
	result, err := executeCLI(binary, f.work, f.root, env, []string{"export"}, false)
	if err != nil || result.runErr != nil {
		t.Fatal("export failed", err, result.runErr)
	}
	mu.Lock()
	exporting = false
	mu.Unlock()
	path := filepath.Join(f.work, ".github/settings.yml")
	exported, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(exported), "id: 7") || !strings.Contains(string(exported), deployKeyOne) || !strings.Contains(string(exported), "archive_this_repository: true") {
		t.Fatal("export omitted identity, key, or archive state")
	}
	for _, args := range [][]string{{"plan"}, {"action", "plan", filepath.Join(f.root, "plan.json")}, {"apply"}} {
		mu.Lock()
		beforeInventory := inventory
		mu.Unlock()
		result, err = executeCLI(binary, f.work, f.root, env, args, false)
		if err != nil || result.runErr != nil {
			t.Errorf("%s failed: %v %v", args[0], err, result.runErr)
		}
		mu.Lock()
		if args[0] == "apply" && inventory-beforeInventory < 2 {
			t.Error("apply did not independently reread the complete inventory", inventory-beforeInventory)
		}
		mu.Unlock()
		if strings.Contains(result.stdout.String()+result.stderr.String(), deployKeyOne) {
			t.Error("public key leaked in output")
		}
	}
	report, err := os.ReadFile(filepath.Join(f.root, "plan.json"))
	if err != nil || strings.Contains(string(report), deployKeyOne) {
		t.Error("report unavailable or exposed public key", err)
	}
	final, err := os.ReadFile(path)
	if err != nil || string(final) != string(exported) {
		t.Error("apply changed exported declaration", err)
	}
	mu.Lock()
	defer mu.Unlock()
	if writes != 0 {
		t.Error("unexpected writes", writes)
	}
}
