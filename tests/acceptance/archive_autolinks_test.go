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

func TestArchivedAutolinksCLI(t *testing.T) {
	const target = "https://fixture-private.example/<num>?q=A%2FB"
	const private = "UPSTREAM_PRIVATE_AUTOLINK_CANARY"
	const remoteOnly = "REMOTE_ONLY_AUTOLINK_CANARY"
	for _, actions := range []bool{false, true} {
		for _, scenario := range []string{"matching", "id differs", "prefix differs", "target differs", "format differs", "missing", "denied", "duplicate member", "idless", "prune", "repo drift", "owner drift", "archive drift", "mixed"} {
			t.Run(fmt.Sprintf("actions=%t/%s", actions, scenario), func(t *testing.T) {
				var mu sync.Mutex
				reads, repoReads, variableReads, writes := 0, 0, 0, 0
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
						fmt.Fprintf(w, `{"id":%d,"full_name":"owner/current","private":false,"archived":%t,"owner":{"id":%d,"login":"owner","type":"User"}}`, id, archived, owner)
					case "/repos/owner/current/autolinks":
						reads++
						if scenario == "denied" {
							w.WriteHeader(403)
							fmt.Fprintf(w, `{"message":%q}`, private)
							return
						}
						if scenario == "missing" {
							fmt.Fprint(w, `[]`)
							return
						}
						if scenario == "duplicate member" {
							fmt.Fprintf(w, `[{"id":8,"id":7,"key_prefix":"TICKET-","url_template":%q,"is_alphanumeric":false}]`, target)
							return
						}
						linkID := 7
						if scenario == "id differs" {
							linkID = 8
						}
						prefix, url, alpha := "TICKET-", target, false
						if scenario == "prefix differs" {
							prefix = "OTHER-"
						}
						if scenario == "target differs" {
							url = "https://other.example/<num>"
						}
						if scenario == "format differs" {
							alpha = true
						}
						fmt.Fprintf(w, `[{"id":%d,"key_prefix":%q,"url_template":%q,"is_alphanumeric":%t},{"id":9,"key_prefix":"UNSELECTED-","url_template":"https://fixture.example/%s/<num>","is_alphanumeric":false}]`, linkID, prefix, url, alpha, remoteOnly)
					case "/repos/owner/current/actions/variables/FLAG":
						variableReads++
						fmt.Fprint(w, `{"name":"FLAG","value":"FIXTURE_VALUE"}`)
					default:
						t.Error("unexpected route", r.URL.Path)
						w.WriteHeader(400)
					}
				}, false, "/autolinks")
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
				selected := "autolinks: [{" + id + "reference_prefix: TICKET-, target_url: '" + target + "', reference_format: numeric}]"
				if scenario == "mixed" {
					selected += ", actions: {variables: [{name: FLAG, value: FIXTURE_VALUE}]}"
				}
				input := repositoryInput("settings: {general: {danger_zone: {archive_this_repository: true}}, " + selected + "}\n")
				if err := writeSettingsFile(f.work, input); err != nil {
					t.Fatal(err)
				}
				args := []string{"apply"}
				if scenario == "prune" {
					args = append(args, "--prune", "autolinks")
				}
				result, err := executeCLI(acceptanceBinary(t), f.work, f.root, env, args, false)
				if err != nil {
					t.Fatal(err)
				}
				success := scenario == "matching" || scenario == "mixed"
				if (result.runErr == nil) != success {
					t.Error("unexpected command success", result.runErr)
				}
				message := result.stdout.String() + result.stderr.String()
				if scenario == "idless" && !strings.Contains(message, "existing IDs required") || scenario == "prune" && !strings.Contains(message, "autolink prune requires explicit unarchive") {
					t.Error("unexpected rejection", message)
				}
				if strings.Contains(message, private) || strings.Contains(message, remoteOnly) {
					t.Error("upstream response leaked")
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
				if scenario == "mixed" && variableReads == 0 {
					t.Error("mixed variable not observed")
				}
				if scenario != "idless" && scenario != "prune" && reads == 0 {
					t.Error("autolink inventory not observed")
				}
				if success && reads < 2 {
					t.Error("did not verify second inventory", reads)
				}
				if (scenario == "repo drift" || scenario == "owner drift" || scenario == "archive drift") && repoReads < 3 {
					t.Error("drift not observed", repoReads)
				}
			})
		}
	}
}

func TestArchivedAutolinksExportPlanApply(t *testing.T) {
	var mu sync.Mutex
	writes, reads, repoReads := 0, 0, 0
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
			repoReads++
			fmt.Fprint(w, `{"id":1,"full_name":"owner/current","private":false,"archived":true,"owner":{"id":2,"login":"owner","type":"User"}}`)
		case "/repos/owner/current/autolinks":
			reads++
			fmt.Fprint(w, `[{"id":7,"key_prefix":"TICKET-","url_template":"https://fixture.example/<num>","is_alphanumeric":false}]`)
		default:
			w.WriteHeader(403)
		}
	}, false, "/autolinks")
	if err != nil {
		t.Fatal(err)
	}
	defer f.close()
	env := labelIDFixtureEnv(f.cfg, false)
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
	if !strings.Contains(string(exported), "id: 7") || !strings.Contains(string(exported), "reference_prefix: TICKET-") || !strings.Contains(string(exported), "archive_this_repository: true") {
		t.Fatal("export omitted selected identity or archive state")
	}
	for _, command := range []string{"plan", "apply"} {
		mu.Lock()
		beforeReads, beforeRepos := reads, repoReads
		mu.Unlock()
		result, err = executeCLI(binary, f.work, f.root, env, []string{command}, false)
		if err != nil || result.runErr != nil {
			t.Errorf("%s failed: %v %v", command, err, result.runErr)
		}
		mu.Lock()
		if command == "apply" && (reads-beforeReads != 2 || repoReads-beforeRepos != 4) {
			t.Errorf("apply matching reads: inventory=%d repository=%d", reads-beforeReads, repoReads-beforeRepos)
		}
		mu.Unlock()
	}
	final, err := os.ReadFile(path)
	if err != nil || string(final) != string(exported) {
		t.Error("apply changed exported declaration", err)
	}
	mu.Lock()
	defer mu.Unlock()
	if writes != 0 {
		t.Error("remote mutation", writes)
	}
}
