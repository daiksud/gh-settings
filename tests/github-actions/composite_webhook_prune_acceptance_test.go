//go:build acceptance

package githubactions

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"testing"
)

func TestBuiltCompositeWebhookPrune(t *testing.T) {
	product, version := suppliedCompositeProduct(t)
	for _, scenario := range []string{"empty-retention", "update-before-delete", "preflight-unknown", "partial-delete", "absence-denied"} {
		t.Run(scenario, func(t *testing.T) {
			f := newCompositeFixture(t)
			f.base["GITHUB_ACTIONS"], f.base["GH_DEBUG"] = "true", "api"
			f.digest = fmt.Sprintf("%x", sha256.Sum256(product))
			f.write(f.base["MOCK_ASSET"], string(product))
			f.write(f.base["MOCK_CHECKSUMS"], f.digest+"  gh-settings-linux-amd64\n")
			f.write(filepath.Join(f.packagePath, "action/cli-version"), version+"\n")
			list := "[]"
			if scenario == "update-before-delete" {
				list = "[{id: 7, content_type: application/x-www-form-urlencoded}]"
			}
			input := "version: daiksud/gh-settings@v0\nrepository: {settings: {webhooks: " + list + "}}\n"
			path := filepath.Join(f.caller, ".github/settings.yml")
			f.write(path, input)
			dir, err := os.MkdirTemp("/tmp", "ghs-hook-prune-")
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { os.RemoveAll(dir) })
			listener, err := net.Listen("unix", filepath.Join(dir, "api.sock"))
			if err != nil {
				t.Fatal(err)
			}
			var mu sync.Mutex
			rows := map[int64]map[string]any{}
			for _, id := range []int64{7, 8} {
				rows[id] = map[string]any{"id": id, "type": "Repository", "name": "web", "url": fmt.Sprintf("https://api.github.com/repos/owner/config/hooks/%d", id), "created_at": "2026-11-14T00:00:00Z", "updated_at": "2026-11-14T01:00:00Z", "active": false, "events": []string{"push", "opaque_event"}, "config": map[string]any{"url": fmt.Sprintf("https://owned-deletion.example/%d", id), "content_type": "json", "insecure_ssl": "1", "secret": "PRIVATE_PRUNE_SECRET"}, "last_response": map[string]any{"message": "PRIVATE_PRUNE_DELIVERY"}}
			}
			if scenario == "preflight-unknown" {
				rows[8]["PRIVATE_PRUNE_UNKNOWN"] = true
			}
			mutations := []string{}
			snapshot := func() []string { mu.Lock(); defer mu.Unlock(); return append([]string{}, mutations...) }
			server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				w.Header().Set("Content-Type", "application/json")
				if r.Host != "api.github.com" || r.Header.Get("Authorization") != "token webhook-prune-installation-fixture-token" {
					t.Error("credential boundary")
					w.WriteHeader(400)
					return
				}
				if r.Method == "GET" && r.URL.Path == "/repos/owner/config" {
					fmt.Fprint(w, `{"id":1,"full_name":"owner/config","private":false,"visibility":"public","archived":false,"owner":{"id":2,"login":"owner","type":"User"}}`)
					return
				}
				if r.Method == "GET" && r.URL.Path == "/repos/owner/config/hooks" {
					if r.URL.RawQuery != "per_page=100&page=1" {
						t.Error("wrong fixed page", r.URL)
					}
					if scenario == "absence-denied" && len(mutations) > 0 {
						w.WriteHeader(403)
						fmt.Fprint(w, `{"message":"PRIVATE_PRUNE_ABSENCE"}`)
						return
					}
					list := []any{}
					for _, id := range []int64{7, 8} {
						if row, ok := rows[id]; ok {
							list = append(list, row)
						}
					}
					json.NewEncoder(w).Encode(list)
					return
				}
				parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/repos/owner/config/hooks/"), "/")
				id, err := strconv.ParseInt(parts[0], 10, 64)
				if err != nil || rows[id] == nil || len(parts) > 2 || len(parts) == 2 && parts[1] != "config" {
					t.Error("unexpected/destination route", r.Method, r.URL)
					w.WriteHeader(400)
					return
				}
				if r.Method == "DELETE" && len(parts) == 1 {
					mutations = append(mutations, fmt.Sprintf("delete:%d", id))
					if scenario == "partial-delete" && id == 8 {
						w.WriteHeader(422)
						fmt.Fprint(w, `{"message":"PRIVATE_PRUNE_UPSTREAM"}`)
						return
					}
					delete(rows, id)
					w.WriteHeader(204)
					return
				}
				row := rows[id]
				if r.Method == "PATCH" && len(parts) == 2 {
					var p map[string]any
					json.NewDecoder(r.Body).Decode(&p)
					if scenario != "update-before-delete" || id != 7 || !reflect.DeepEqual(p, map[string]any{"content_type": "form"}) {
						t.Error("wrong payload", p)
					}
					mutations = append(mutations, "config:7")
					row["config"].(map[string]any)["content_type"] = "form"
					row["updated_at"] = "2026-11-14T02:00:00Z"
					row["config"].(map[string]any)["secret"] = "PRIVATE_CHANGED_MASK"
				} else if r.Method != "GET" {
					t.Error("unexpected mutation", r.Method)
					w.WriteHeader(400)
					return
				}
				if len(parts) == 2 {
					json.NewEncoder(w).Encode(row["config"])
				} else {
					json.NewEncoder(w).Encode(row)
				}
			})}
			go func() { _ = server.Serve(listener) }()
			t.Cleanup(func() { _ = server.Close() })
			f.write(filepath.Join(dir, "config.yml"), "http_unix_socket: "+filepath.Join(dir, "api.sock")+"\n")
			f.base["GH_CONFIG_DIR"], f.base["GH_PATH_DISABLED"] = dir, "1"
			for _, k := range []string{"HTTP_PROXY", "HTTPS_PROXY", "http_proxy", "https_proxy"} {
				f.base[k] = "http://127.0.0.1:1"
			}
			f.base["NO_PROXY"], f.base["no_proxy"] = "", ""
			options := map[string]string{"github.action_ref": version, "inputs.prune": "webhooks"}
			for _, reporting := range []string{"detailed", "status-only"} {
				options["inputs.plan-reporting"] = reporting
				plan := f.run(t, "plan", "webhook-prune-installation-fixture-token", options)
				if (plan.status != 0) != (scenario == "preflight-unknown") || strings.Contains(plan.log+plan.summary, "PRIVATE_PRUNE") {
					t.Fatal(plan)
				}
				if scenario != "preflight-unknown" {
					if reporting == "status-only" && (strings.Contains(plan.log+plan.summary, "owned-deletion.example") || strings.Contains(plan.summary, "id 8")) {
						t.Fatal("status-only leak", plan)
					}
					if reporting == "detailed" && (!strings.Contains(plan.summary, "owned-deletion.example/8") || !strings.Contains(plan.summary, "delete")) {
						t.Fatal("whole deletion unreviewable", plan.summary)
					}
				}
				if len(plan.remainingReports) != 0 {
					t.Fatal("private report retained")
				}
			}
			if len(snapshot()) != 0 {
				t.Fatal("plan mutated")
			}
			delete(options, "inputs.plan-reporting")
			options["inputs.prune"] = "all"
			result := f.run(t, "apply", "webhook-prune-installation-fixture-token", options)
			failed := scenario == "preflight-unknown" || scenario == "partial-delete" || scenario == "absence-denied"
			if (result.status != 0) != failed || failed && len(result.outputs) != 0 || !failed && len(result.outputs) != 4 || strings.Contains(result.log+result.summary, "PRIVATE_PRUNE") {
				t.Fatal(result)
			}
			want := []string{"delete:7", "delete:8"}
			if scenario == "update-before-delete" {
				want = []string{"config:7", "delete:8"}
			}
			if scenario == "preflight-unknown" {
				want = []string{}
			}
			if scenario == "absence-denied" {
				want = []string{"delete:7"}
			}
			if !reflect.DeepEqual(snapshot(), want) {
				t.Fatal("write ordering", snapshot(), want)
			}
			data, _ := os.ReadFile(path)
			if string(data) != input || f.git("rev-parse", "HEAD") != f.sha {
				t.Fatal("Action changed caller")
			}
			if !failed {
				again := f.run(t, "apply", "webhook-prune-installation-fixture-token", options)
				if again.status != 0 || !reflect.DeepEqual(snapshot(), want) {
					t.Fatal("matching repeat wrote", again)
				}
			}
		})
	}
}
