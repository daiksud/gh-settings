//go:build acceptance

package githubactions

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// The supplied product runs the real composite with a local API; this is not
// evidence that GitHub permits these reads on an archived repository.
func TestBuiltCompositeArchivedRunners(t *testing.T) {
	product := suppliedCompositeProduct(t)
	for _, scenario := range []string{"matching", "empty", "fold and order", "different", "missing", "denied", "incomplete", "duplicate", "duplicate JSON", "wrong id", "label count", "detail disagreement", "repo drift", "owner drift", "archive drift", "prune", "mixed", "export flow"} {
		t.Run(scenario, func(t *testing.T) {
			f := product.newFixture(t)
			f.base["GITHUB_ACTIONS"], f.base["GH_DEBUG"] = "true", "api"

			desired := `["gpu", "linux-build"]`
			if scenario == "empty" {
				desired = `[]`
			}
			selected := "actions: {runners: [{id: 23, custom_labels: " + desired + "}]}"
			if scenario == "mixed" {
				selected += ", secrets_and_variables: {actions: {variables: [{name: FLAG, value: FIXTURE_VALUE}]}}"
			}
			input := "version: daiksud/gh-settings@v0\nrepository: {settings: {general: {danger_zone: {archive_this_repository: true}}, " + selected + "}}\n"
			path := filepath.Join(f.caller, ".github/settings.yml")
			f.write(path, input)
			head, index := f.git("rev-parse", "HEAD"), f.git("write-tree")
			dir, err := os.MkdirTemp("/tmp", "ghs-archived-runner-")
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = os.RemoveAll(dir) })
			listener, err := net.Listen("unix", filepath.Join(dir, "api.sock"))
			if err != nil {
				t.Fatal(err)
			}
			var mu sync.Mutex
			repoReads, detailReads, labelReads, variableReads, writes := 0, 0, 0, 0, 0
			exporting := scenario == "export flow"
			labels := []any{map[string]any{"id": 1, "name": "self-hosted", "type": "read-only"}, map[string]any{"name": "gpu", "type": "custom"}, map[string]any{"name": "linux-build", "type": "custom"}}
			switch scenario {
			case "empty":
				labels = labels[:1]
			case "fold and order":
				labels = []any{labels[0], map[string]any{"name": "LINUX-BUILD", "type": "custom"}, map[string]any{"name": "GPU", "type": "custom"}}
			case "different":
				labels = []any{labels[0], map[string]any{"name": "other", "type": "custom"}}
			case "duplicate":
				labels = []any{labels[0], labels[1], map[string]any{"name": "GPU", "type": "custom"}}
			}
			server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				w.Header().Set("Content-Type", "application/json")
				if r.Host != "api.github.com" || r.Header.Get("Authorization") != "token archived-runner-fixture" {
					t.Error("unexpected target or credential")
					w.WriteHeader(400)
					return
				}
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
				case "/repos/owner/config":
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
					fmt.Fprintf(w, `{"id":%d,"full_name":"owner/config","private":false,"archived":%t,"owner":{"id":%d,"login":"owner","type":"User"}}`, id, archived, owner)
				case "/repos/owner/config/actions/runners":
					row := map[string]any{"id": 23, "name": "private-runner-canary", "os": "private-os-canary", "status": "online", "busy": false, "labels": labels}
					_ = json.NewEncoder(w).Encode(map[string]any{"total_count": 1, "runners": []any{row}})
				case "/repos/owner/config/actions/runners/23":
					detailReads++
					if scenario == "missing" {
						w.WriteHeader(404)
						return
					}
					if scenario == "denied" {
						w.WriteHeader(403)
						fmt.Fprint(w, `{"message":"private-upstream-canary"}`)
						return
					}
					if scenario == "duplicate JSON" {
						fmt.Fprint(w, `{"id":24,"id":23,"name":"private-runner-canary","os":"private-os-canary","status":"online","busy":false,"labels":[]}`)
						return
					}
					row := map[string]any{"id": 23, "name": "private-runner-canary", "os": "private-os-canary", "status": "online", "busy": false, "labels": labels}
					if scenario == "wrong id" {
						row["id"] = 24
					}
					if scenario == "incomplete" {
						delete(row, "busy")
					}
					_ = json.NewEncoder(w).Encode(row)
				case "/repos/owner/config/actions/runners/23/labels":
					labelReads++
					observed := labels
					if scenario == "detail disagreement" {
						observed = []any{labels[0], map[string]any{"name": "different", "type": "custom"}}
					}
					total := len(observed)
					if scenario == "label count" {
						total++
					}
					_ = json.NewEncoder(w).Encode(map[string]any{"total_count": total, "labels": observed})
				case "/repos/owner/config/actions/variables/FLAG":
					variableReads++
					fmt.Fprint(w, `{"name":"FLAG","value":"FIXTURE_VALUE"}`)
				default:
					if !exporting {
						t.Error("unexpected route", r.URL.Path)
					}
					w.WriteHeader(403)
				}
			})}
			go func() { _ = server.Serve(listener) }()
			t.Cleanup(func() { _ = server.Close() })
			f.write(filepath.Join(dir, "config.yml"), "http_unix_socket: "+filepath.Join(dir, "api.sock")+"\n")
			f.base["GH_CONFIG_DIR"], f.base["GH_PATH_DISABLED"] = dir, "1"
			for _, name := range []string{"HTTP_PROXY", "HTTPS_PROXY", "http_proxy", "https_proxy"} {
				f.base[name] = "http://127.0.0.1:1"
			}
			f.base["NO_PROXY"], f.base["no_proxy"] = "", ""
			opts := map[string]string{"GH_DEBUG": "api"}
			if scenario == "prune" {
				opts["inputs.prune"] = "runners"
			}
			privacy := func(result compositeResult) {
				t.Helper()
				out := result.log + result.summary + result.api + result.calls + fmt.Sprint(result.outputs)
				for _, canary := range []string{"private-runner-canary", "private-os-canary", "private-upstream-canary"} {
					if strings.Contains(out, canary) {
						t.Error("runner metadata or upstream body leaked")
					}
				}
			}
			if scenario == "export flow" {
				if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
				exported := f.run(t, "export", "archived-runner-fixture", opts)
				privacy(exported)
				if exported.status != 0 {
					t.Fatal("export failed", exported.status)
				}
				mu.Lock()
				exporting = false
				mu.Unlock()
				data, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				input = string(data)
				if !strings.Contains(input, "id: 23") || !strings.Contains(input, "custom_labels:") || !strings.Contains(input, "archive_this_repository: true") {
					t.Error("export omitted runner identity, labels or archive state")
				}
				plan := f.run(t, "plan", "archived-runner-fixture", opts)
				privacy(plan)
				if plan.status != 0 || len(plan.remainingReports) != 0 {
					t.Error("plan failed or retained report", plan.status)
				}
			}
			mu.Lock()
			beforeDetail, beforeLabels := detailReads, labelReads
			mu.Unlock()
			result := f.run(t, "apply", "archived-runner-fixture", opts)
			privacy(result)
			success := scenario == "matching" || scenario == "empty" || scenario == "fold and order" || scenario == "mixed" || scenario == "export flow"
			if (result.status == 0) != success {
				t.Error("unexpected Action status", result.status)
			}
			if !success && len(result.outputs) != 0 {
				t.Error("failed apply published outputs")
			}
			data, err := os.ReadFile(path)
			if err != nil || string(data) != input || f.git("rev-parse", "HEAD") != head || f.git("write-tree") != index {
				t.Error("declaration, HEAD or index changed", err)
			}
			mu.Lock()
			defer mu.Unlock()
			if writes != 0 {
				t.Error("unexpected remote mutation", writes)
			}
			if success && (detailReads-beforeDetail < 2 || labelReads-beforeLabels < 2) {
				t.Error("selected runner detail and labels were not independently reread", detailReads-beforeDetail, labelReads-beforeLabels)
			}
			if scenario == "mixed" && variableReads == 0 {
				t.Error("mixed variable was not observed")
			}
			if (scenario == "repo drift" || scenario == "owner drift" || scenario == "archive drift") && repoReads < 3 {
				t.Error("identity drift was not observed", repoReads)
			}
		})
	}
}
