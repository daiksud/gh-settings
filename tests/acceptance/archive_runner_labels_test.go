//go:build acceptance

package acceptance

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestArchivedRunnerLabelsCLI(t *testing.T) {
	for _, actions := range []bool{false, true} {
		for _, scenario := range []string{"matching", "empty", "fold order", "label mismatch", "id mismatch", "missing", "denied", "malformed", "duplicate member", "duplicate label", "inconsistent labels", "runner drift", "repo drift", "owner drift", "archive drift", "prune", "mixed"} {
			t.Run(fmt.Sprintf("actions=%t/%s", actions, scenario), func(t *testing.T) {
				var mu sync.Mutex
				counts := map[string]int{}
				writes := 0
				f, err := startHTTPFixture(func(w http.ResponseWriter, r *http.Request) {
					mu.Lock()
					defer mu.Unlock()
					w.Header().Set("Content-Type", "application/json")
					if r.Method != http.MethodGet {
						writes++
						w.WriteHeader(400)
						return
					}
					counts[r.URL.Path]++
					switch r.URL.Path {
					case "/repos/owner/current":
						id, owner, archived := 1, 2, true
						if counts[r.URL.Path] >= 3 {
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
					case "/repos/owner/current/actions/runners":
						labels := []map[string]any{{"id": 1, "name": "self-hosted-private", "type": "read-only"}, {"id": 2, "name": "gpu", "type": "custom"}, {"id": 3, "name": "accelerated", "type": "custom"}}
						row := map[string]any{"id": 23, "name": "private-runner", "os": "private-os", "status": "offline", "busy": false, "labels": labels}
						json.NewEncoder(w).Encode(map[string]any{"total_count": 1, "runners": []any{row}})
					case "/repos/owner/current/actions/runners/23", "/repos/owner/current/actions/runners/23/labels":
						if scenario == "missing" {
							w.WriteHeader(404)
							return
						}
						if scenario == "denied" {
							w.WriteHeader(403)
							fmt.Fprint(w, `{"message":"private-upstream-canary"}`)
							return
						}
						labelEndpoint := strings.HasSuffix(r.URL.Path, "/labels")
						labels := []map[string]any{{"id": 1, "name": "self-hosted-private", "type": "read-only"}}
						if scenario != "empty" {
							labels = append(labels, map[string]any{"id": 2, "name": "gpu", "type": "custom"}, map[string]any{"id": 3, "name": "accelerated", "type": "custom"})
						}
						if scenario == "label mismatch" {
							labels[1]["name"] = "other"
						}
						if scenario == "duplicate label" {
							labels = append(labels, map[string]any{"id": 4, "name": "GPU", "type": "custom"})
						}
						if scenario == "inconsistent labels" && labelEndpoint {
							labels[1]["name"] = "other"
						}
						if scenario == "malformed" && labelEndpoint {
							fmt.Fprint(w, `{"total_count":1}`)
							return
						}
						if scenario == "duplicate member" && !labelEndpoint {
							fmt.Fprint(w, `{"id":24,"id":23,"name":"private-runner","os":"private-os","labels":[]}`)
							return
						}
						if labelEndpoint {
							json.NewEncoder(w).Encode(map[string]any{"total_count": len(labels), "labels": labels})
							return
						}
						id, name := 23, "private-runner"
						if scenario == "id mismatch" {
							id = 24
						}
						if scenario == "runner drift" && counts[r.URL.Path] >= 2 {
							name = "private-runner-changed"
						}
						json.NewEncoder(w).Encode(map[string]any{"id": id, "name": name, "os": "private-os", "status": "offline", "busy": false, "labels": labels})
					case "/repos/owner/current/actions/variables/FLAG":
						fmt.Fprint(w, `{"name":"FLAG","value":"FIXTURE_VALUE"}`)
					default:
						t.Error("unexpected request", r.URL.Path)
						w.WriteHeader(400)
					}
				}, false, "/actions/runners")
				if err != nil {
					t.Fatal(err)
				}
				defer f.close()
				env := append(labelIDFixtureEnv(f.cfg, actions), "GH_DEBUG=api")
				if err := setupOrdinaryOrigin(f.work, env); err != nil {
					t.Fatal(err)
				}
				labels := "[gpu, accelerated]"
				if scenario == "empty" {
					labels = "[]"
				}
				if scenario == "fold order" {
					labels = "[ACCELERATED, GPU]"
				}
				selected := "actions: {runners: [{id: 23, custom_labels: " + labels + "}]}"
				if scenario == "mixed" {
					selected += ", secrets_and_variables: {actions: {variables: [{name: FLAG, value: FIXTURE_VALUE}]}}"
				}
				input := repositoryInput("settings: {general: {danger_zone: {archive_this_repository: true}}, " + selected + "}\n")
				if err := writeSettingsFile(f.work, input); err != nil {
					t.Fatal(err)
				}
				args := []string{"apply"}
				if scenario == "prune" {
					args = append(args, "--prune", "runners")
				}
				result, err := executeCLI(acceptanceBinary(t), f.work, f.root, env, args, false)
				if err != nil {
					t.Fatal(err)
				}
				success := scenario == "matching" || scenario == "empty" || scenario == "fold order" || scenario == "mixed"
				if (result.runErr == nil) != success {
					t.Error("unexpected command status", result.runErr)
				}
				if scenario == "prune" && !strings.Contains(result.stdout.String()+result.stderr.String(), "runner prune requires explicit unarchive") {
					t.Error("prune did not reach the archived guard")
				}
				if strings.Contains(result.stdout.String()+result.stderr.String(), "private-") {
					t.Error("private runner evidence leaked")
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
				if success && (counts["/repos/owner/current/actions/runners/23"] < 2 || counts["/repos/owner/current/actions/runners/23/labels"] < 2) {
					t.Error("missing independent runner and label observations", counts)
				}
				if scenario == "prune" && counts["/repos/owner/current/actions/runners"] == 0 {
					t.Error("prune inventory was not observed")
				}
				if scenario == "mixed" && counts["/repos/owner/current/actions/variables/FLAG"] == 0 {
					t.Error("mixed variable not observed")
				}
				if (scenario == "repo drift" || scenario == "owner drift" || scenario == "archive drift") && counts["/repos/owner/current"] < 3 {
					t.Error("target drift was not observed", counts)
				}
				if scenario == "runner drift" && counts["/repos/owner/current/actions/runners/23"] < 2 {
					t.Error("runner drift was not observed", counts)
				}
			})
		}
	}
}

func TestArchivedRunnerLabelsExportPlanApply(t *testing.T) {
	var mu sync.Mutex
	counts := map[string]int{}
	writes := 0
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
		counts[r.URL.Path]++
		labels := []map[string]any{{"id": 1, "name": "self-hosted-private", "type": "read-only"}, {"id": 2, "name": "gpu", "type": "custom"}}
		row := map[string]any{"id": 23, "name": "private-runner", "os": "private-os", "status": "offline", "busy": false, "labels": labels}
		switch r.URL.Path {
		case "/repos/owner/current":
			fmt.Fprint(w, `{"id":1,"full_name":"owner/current","private":false,"archived":true,"owner":{"id":2,"login":"owner","type":"User"}}`)
		case "/repos/owner/current/actions/runners":
			json.NewEncoder(w).Encode(map[string]any{"total_count": 1, "runners": []any{row}})
		case "/repos/owner/current/actions/runners/23":
			json.NewEncoder(w).Encode(row)
		case "/repos/owner/current/actions/runners/23/labels":
			json.NewEncoder(w).Encode(map[string]any{"total_count": len(labels), "labels": labels})
		default:
			w.WriteHeader(403)
		}
	}, false, "/actions/runners", "/actions/runners/23", "/actions/runners/23/labels")
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
	if !strings.Contains(string(exported), "id: 23") || !strings.Contains(string(exported), "gpu") || !strings.Contains(string(exported), "archive_this_repository: true") {
		t.Fatal("export omitted selected runner or archive state")
	}
	if strings.Contains(string(exported), "private-") {
		t.Fatal("export included private runner evidence")
	}
	for _, command := range []string{"plan", "apply"} {
		mu.Lock()
		beforeDetail, beforeLabels := counts["/repos/owner/current/actions/runners/23"], counts["/repos/owner/current/actions/runners/23/labels"]
		mu.Unlock()
		result, err = executeCLI(binary, f.work, f.root, env, []string{command}, false)
		if err != nil || result.runErr != nil {
			t.Errorf("%s failed: %v %v", command, err, result.runErr)
		}
		if strings.Contains(result.stdout.String()+result.stderr.String(), "private-") {
			t.Error("private runner evidence leaked")
		}
		mu.Lock()
		if command == "apply" && (counts["/repos/owner/current/actions/runners/23"]-beforeDetail < 2 || counts["/repos/owner/current/actions/runners/23/labels"]-beforeLabels < 2) {
			t.Error("apply did not independently reread runner and labels")
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
		t.Error("unexpected remote writes", writes)
	}
}
