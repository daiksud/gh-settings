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
	"sync"
	"testing"
)

func TestWebhookPruneMixedExecutable(t *testing.T) {
	for _, mode := range []string{"hosted", "record", "record-failure"} {
		t.Run(mode, func(t *testing.T) {
			var mu sync.Mutex
			var path string
			rows := map[int]map[string]any{}
			row := func(id int, url string) map[string]any {
				return map[string]any{"id": id, "name": "web", "type": "Repository", "url": fmt.Sprintf("https://api.github.com/repos/owner/current/hooks/%d", id), "created_at": "2026-11-14T00:00:00Z", "updated_at": "2026-11-14T01:00:00Z", "active": false, "events": []string{"push"}, "config": map[string]any{"url": url, "content_type": "json", "insecure_ssl": "0", "secret": "PRIVATE_MIXED_SECRET"}, "last_response": map[string]any{"message": "PRIVATE_MIXED_DELIVERY"}}
			}
			rows[7], rows[8] = row(7, "https://retained.example/old"), row(8, "https://removed.example/hook")
			writes := []string{}
			f, err := startHTTPFixture(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				w.Header().Set("Content-Type", "application/json")
				if r.Host != "api.github.com" {
					t.Error("destination request")
					w.WriteHeader(400)
					return
				}
				switch r.Method + " " + r.URL.Path {
				case "GET /repos/owner/current":
					fmt.Fprint(w, `{"id":1,"full_name":"owner/current","visibility":"public","private":false,"archived":false,"owner":{"id":2,"login":"owner","type":"User"}}`)
				case "GET /repos/owner/current/hooks":
					if r.URL.RawQuery != "per_page=100&page=1" {
						t.Error("unexpected page", r.URL)
					}
					list := []any{}
					for _, id := range []int{7, 8, 9} {
						if v := rows[id]; v != nil {
							list = append(list, v)
						}
					}
					json.NewEncoder(w).Encode(list)
				case "PATCH /repos/owner/current/hooks/7/config":
					var body map[string]any
					json.NewDecoder(r.Body).Decode(&body)
					if !reflect.DeepEqual(body, map[string]any{"content_type": "form"}) {
						t.Error("unexpected config", body)
					}
					writes = append(writes, "update:7")
					rows[7]["config"].(map[string]any)["content_type"] = "form"
					rows[7]["updated_at"] = "2026-11-14T02:00:00Z"
					json.NewEncoder(w).Encode(rows[7]["config"])
				case "POST /repos/owner/current/hooks":
					var body map[string]any
					json.NewDecoder(r.Body).Decode(&body)
					expected := map[string]any{"name": "web", "active": false, "events": []any{"push"}, "config": map[string]any{"url": "https://created.example/hook", "content_type": "json", "insecure_ssl": "0"}}
					if !reflect.DeepEqual(body, expected) {
						t.Error("unexpected create payload", body)
					}
					writes = append(writes, "create:9")
					rows[9] = row(9, "https://created.example/hook")
					if mode == "record-failure" {
						if err := os.WriteFile(path, []byte("# concurrent edit\n"), 0600); err != nil {
							t.Error(err)
						}
					}
					w.WriteHeader(201)
					json.NewEncoder(w).Encode(rows[9])
				case "DELETE /repos/owner/current/hooks/8":
					writes = append(writes, "delete:8")
					delete(rows, 8)
					w.WriteHeader(204)
				default:
					if r.Method == "GET" && strings.HasPrefix(r.URL.Path, "/repos/owner/current/hooks/") {
						var id int
						fmt.Sscanf(strings.TrimPrefix(r.URL.Path, "/repos/owner/current/hooks/"), "%d", &id)
						if v := rows[id]; v != nil {
							if strings.HasSuffix(r.URL.Path, "/config") {
								json.NewEncoder(w).Encode(v["config"])
							} else {
								json.NewEncoder(w).Encode(v)
							}
							return
						}
					}
					t.Error("unexpected route", r.Method, r.URL)
					w.WriteHeader(400)
				}
			}, false, "/hooks", "/hooks/")
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(f.close)
			env := append(labelIDFixtureEnv(f.cfg, mode == "hosted"), "GH_DEBUG=api")
			if err = setupOrdinaryOrigin(f.work, env); err != nil {
				t.Fatal(err)
			}
			input := repositoryInput("settings: {webhooks: [{id: 7, content_type: application/x-www-form-urlencoded, active: false, events: [push]}, {payload_url: https://created.example/hook, content_type: application/json, active: false, events: [push]}]}\n")
			path = filepath.Join(f.work, ".github/settings.yml")
			if err = writeSettingsFile(f.work, input); err != nil {
				t.Fatal(err)
			}
			plan, e := executeCLI(acceptanceBinary(t), f.work, f.root, env, []string{"plan", "--prune", "webhooks"}, false)
			if e != nil || plan.runErr != nil || !strings.Contains(plan.stdout.String(), "removed.example") || !strings.Contains(plan.stdout.String(), "automatic ping") || !strings.Contains(plan.stdout.String(), "opaque Secret") {
				t.Fatal(e, plan.runErr, plan.stdout.String(), plan.stderr.String())
			}
			result, e := executeCLI(acceptanceBinary(t), f.work, f.root, env, []string{"apply", "--prune", "webhooks"}, false)
			if e != nil || (result.runErr != nil) != (mode == "record-failure") || strings.Contains(result.stdout.String()+result.stderr.String(), "PRIVATE_MIXED") {
				t.Fatal(e, result.runErr, result.stdout.String(), result.stderr.String())
			}
			mu.Lock()
			got := append([]string{}, writes...)
			_, retained := rows[9]
			mu.Unlock()
			want := []string{"update:7", "create:9", "delete:8"}
			if mode == "record-failure" {
				want = want[:2]
			}
			if !reflect.DeepEqual(got, want) || !retained {
				t.Fatal("unexpected sequencing", got, retained)
			}
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if mode == "hosted" && string(data) != input || mode == "record" && !strings.Contains(string(data), "id: 9") || mode == "record-failure" && string(data) != "# concurrent edit\n" {
				t.Fatal("source result", string(data))
			}
		})
	}
}
