//go:build acceptance

package acceptance

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"testing"
)

func TestWebhooksPruneExecutable(t *testing.T) {
	for _, mode := range []string{"delete-all", "all", "retain", "ordinary-empty", "omitted", "preflight-unknown", "preflight-timestamp", "retained-drift", "HTTP-error", "absence-denied", "partial", "own-unarchive", "update-before-delete"} {
		t.Run(mode, func(t *testing.T) {
			var mu sync.Mutex
			rows := map[int64]map[string]any{}
			for _, id := range []int64{7, 8} {
				rows[id] = map[string]any{"id": id, "name": "web", "type": "Repository", "url": fmt.Sprintf("https://api.github.com/repos/owner/current/hooks/%d", id), "created_at": "2026-11-14T00:00:00Z", "updated_at": "2026-11-14T01:00:00Z", "active": false, "events": []string{"push", "opaque_event"}, "config": map[string]any{"url": fmt.Sprintf("https://hook.example/%d", id), "content_type": "json", "insecure_ssl": "1", "secret": "PRIVATE_PRUNE_SECRET"}, "last_response": map[string]any{"message": "PRIVATE_PRUNE_DELIVERY"}}
			}
			if mode == "preflight-unknown" {
				rows[8]["PRIVATE_FUTURE_CONFIG"] = true
			}
			if mode == "preflight-timestamp" {
				delete(rows[8], "updated_at")
			}
			archived := mode == "own-unarchive"
			phase, applyLists, requests := "plan", 0, 0
			mutations := []string{}
			f, err := startHTTPFixture(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				requests++
				w.Header().Set("Content-Type", "application/json")
				if r.URL.Path == "/repos/owner/current" {
					if r.Method == "PATCH" {
						var payload map[string]any
						json.NewDecoder(r.Body).Decode(&payload)
						if mode != "own-unarchive" || !reflect.DeepEqual(payload, map[string]any{"archived": false}) {
							t.Error("unexpected unrelated write", payload)
						}
						mutations = append(mutations, "unarchive")
						archived = false
					} else if r.Method != "GET" {
						t.Error("unexpected target method", r.Method)
					}
					json.NewEncoder(w).Encode(map[string]any{"id": 1, "full_name": "owner/current", "visibility": "public", "private": false, "archived": archived, "owner": map[string]any{"id": 2, "login": "owner", "type": "User"}})
					return
				}
				if r.URL.Path == "/repos/owner/current/hooks" && r.Method == "GET" {
					if r.URL.RawQuery != "per_page=100&page=1" {
						t.Error("wrong fixed inventory path", r.URL)
					}
					if phase == "apply" {
						applyLists++
						if mode == "retained-drift" && applyLists == 2 {
							rows[7]["updated_at"] = "2026-11-14T02:00:00Z"
						}
					}
					if mode == "absence-denied" && len(mutations) > 0 {
						w.WriteHeader(403)
						fmt.Fprint(w, `{"message":"PRIVATE_ABSENCE"}`)
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
				parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/repos/owner/current/hooks/"), "/")
				id, e := strconv.ParseInt(parts[0], 10, 64)
				if e != nil || rows[id] == nil || len(parts) > 2 || len(parts) == 2 && parts[1] != "config" {
					t.Error("unexpected/destination route", r.Method, r.URL)
					w.WriteHeader(400)
					return
				}
				row := rows[id]
				if r.Method == "DELETE" && len(parts) == 1 {
					if archived {
						t.Error("delete before unarchive")
					}
					mutations = append(mutations, fmt.Sprintf("delete:%d", id))
					if mode == "HTTP-error" || mode == "partial" && id == 8 {
						w.WriteHeader(422)
						fmt.Fprint(w, `{"message":"PRIVATE_DELETE_ERROR"}`)
						return
					}
					delete(rows, id)
					w.WriteHeader(204)
					return
				}
				if r.Method == "PATCH" && len(parts) == 2 {
					var payload map[string]any
					json.NewDecoder(r.Body).Decode(&payload)
					if mode != "update-before-delete" || id != 7 || !reflect.DeepEqual(payload, map[string]any{"content_type": "form"}) {
						t.Error("invalid config update", payload)
					}
					mutations = append(mutations, "config:7")
					row["config"].(map[string]any)["content_type"] = "form"
					row["updated_at"] = "2026-11-14T02:00:00Z"
				} else if r.Method != "GET" {
					t.Error("unexpected method", r.Method)
					w.WriteHeader(400)
					return
				}
				if len(parts) == 2 {
					json.NewEncoder(w).Encode(row["config"])
				} else {
					json.NewEncoder(w).Encode(row)
				}
			}, false, "/hooks", "/hooks/")
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(f.close)
			env := append(labelIDFixtureEnv(f.cfg, true), "GH_DEBUG=api")
			if err := setupOrdinaryOrigin(f.work, env); err != nil {
				t.Fatal(err)
			}
			settingsYAML := "webhooks: []\n"
			if mode == "retain" || mode == "retained-drift" {
				settingsYAML = "webhooks: [{id: 7, content_type: application/json}]\n"
			}
			if mode == "update-before-delete" {
				settingsYAML = "webhooks: [{id: 7, content_type: application/x-www-form-urlencoded}]\n"
			}
			if mode == "own-unarchive" {
				settingsYAML += "general: {danger_zone: {archive_this_repository: false}}\n"
			}
			if strings.HasPrefix(mode, "preflight-") {
				settingsYAML += "general: {danger_zone: {archive_this_repository: false}}\n"
			}
			input := repositoryInput("settings:\n" + indentInput(settingsYAML, 2))
			if mode == "omitted" {
				input = repositoryInput("{}\n")
			}
			if err := writeSettingsFile(f.work, input); err != nil {
				t.Fatal(err)
			}
			option := []string{"--prune", "webhooks"}
			if mode == "all" {
				option[1] = "all"
			}
			if mode == "ordinary-empty" {
				option = nil
			}
			plan, err := executeCLI(acceptanceBinary(t), f.work, f.root, env, append([]string{"plan"}, option...), false)
			if err != nil {
				t.Fatal(err)
			}
			preflight := strings.HasPrefix(mode, "preflight-")
			if (plan.runErr != nil) != preflight || strings.Contains(plan.stdout.String()+plan.stderr.String(), "PRIVATE_") {
				t.Fatal("plan", plan.runErr, plan.stdout.String(), plan.stderr.String())
			}
			if !preflight && mode != "ordinary-empty" && mode != "omitted" && (!strings.Contains(plan.stdout.String(), "delete") || !strings.Contains(plan.stdout.String(), "hook.example")) {
				t.Fatal("deletion plan incomplete", plan.stdout.String())
			}
			mu.Lock()
			if len(mutations) != 0 {
				t.Fatal("plan wrote")
			}
			phase = "apply"
			mu.Unlock()
			applied, err := executeCLI(acceptanceBinary(t), f.work, f.root, env, append([]string{"apply"}, option...), false)
			if err != nil {
				t.Fatal(err)
			}
			failed := preflight || mode == "retained-drift" || mode == "HTTP-error" || mode == "absence-denied" || mode == "partial"
			if (applied.runErr != nil) != failed || strings.Contains(applied.stdout.String()+applied.stderr.String(), "PRIVATE_") {
				t.Fatal("apply", applied.runErr, applied.stdout.String(), applied.stderr.String())
			}
			mu.Lock()
			got := append([]string{}, mutations...)
			count := requests
			mu.Unlock()
			want := []string{"delete:7", "delete:8"}
			switch mode {
			case "retain":
				want = []string{"delete:8"}
			case "update-before-delete":
				want = []string{"config:7", "delete:8"}
			case "own-unarchive":
				want = []string{"unarchive", "delete:7", "delete:8"}
			case "HTTP-error", "absence-denied":
				want = []string{"delete:7"}
			case "preflight-unknown", "preflight-timestamp", "retained-drift", "omitted", "ordinary-empty":
				want = []string{}
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatal("order/progress", got, want)
			}
			if (mode == "omitted" || mode == "ordinary-empty") && count != 0 {
				t.Fatal("unmanaged collection acquired", count)
			}
			if mode == "partial" && !strings.Contains(applied.stderr.String(), "earlier") {
				t.Fatal("earlier successful deletion lost", applied.stderr.String())
			}
			if err := checkSettingsUnchanged(f.work, input); err != nil {
				t.Fatal(err)
			}
			if !failed {
				repeat, e := executeCLI(acceptanceBinary(t), f.work, f.root, env, append([]string{"apply"}, option...), false)
				if e != nil || repeat.runErr != nil {
					t.Fatal(e, repeat.runErr, repeat.stderr.String())
				}
				mu.Lock()
				same := reflect.DeepEqual(mutations, want)
				mu.Unlock()
				if !same {
					t.Fatal("repeat wrote")
				}
			}
		})
	}
}
