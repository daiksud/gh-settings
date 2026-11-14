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

func TestWebhookCreationExecutableLifecycle(t *testing.T) {
	for _, mode := range []string{"record", "empty-events", "no-writeback", "hosted", "own-unarchive", "mixed", "guard", "collision", "unavailable", "HTTP-error", "bad-ack", "readback-failure", "file-drift", "later-failure"} {
		t.Run(mode, func(t *testing.T) {
			var mu sync.Mutex
			posts, patches, lists := 0, 0, 0
			archived := mode == "own-unarchive"
			order := []string{}
			oldCfg := map[string]any{"url": "https://existing.example/omitted", "content_type": "json", "insecure_ssl": "1", "secret": "PRIVATE_CREATION_OLD_SECRET"}
			rows := map[int]map[string]any{}
			makeRow := func(id int, cfg map[string]any, active bool, events []string) map[string]any {
				return map[string]any{"id": id, "type": "Repository", "name": "web", "active": active, "events": events, "created_at": "2026-11-14T00:00:00Z", "url": fmt.Sprintf("https://api.github.com/repos/owner/current/hooks/%d", id), "config": cfg, "last_response": map[string]any{"message": "PRIVATE_CREATION_DELIVERY"}}
			}
			rows[7] = makeRow(7, oldCfg, false, []string{"push"})
			if mode == "collision" {
				oldCfg["url"] = "https://selected.example/new%2f?q=%2F"
			}
			var path string
			f, e := startHTTPFixture(func(w http.ResponseWriter, r *http.Request) {
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
					json.NewEncoder(w).Encode(map[string]any{"id": 1, "full_name": "owner/current", "visibility": "public", "private": false, "archived": archived, "owner": map[string]any{"id": 2, "login": "owner", "type": "User"}})
				case "PATCH /repos/owner/current":
					var p map[string]any
					json.NewDecoder(r.Body).Decode(&p)
					if mode != "own-unarchive" || !reflect.DeepEqual(p, map[string]any{"archived": false}) {
						t.Error("unexpected repository mutation", p)
					}
					archived = false
					order = append(order, "unarchive")
					fmt.Fprint(w, `{"id":1,"full_name":"owner/current","archived":false}`)
				case "GET /repos/owner/current/hooks":
					lists++
					if mode == "unavailable" {
						w.WriteHeader(403)
						fmt.Fprint(w, `{"message":"PRIVATE_CREATION_UNAVAILABLE"}`)
						return
					}
					values := []any{}
					for _, id := range []int{7, 8, 9} {
						if row, ok := rows[id]; ok {
							values = append(values, row)
						}
					}
					json.NewEncoder(w).Encode(values)
				case "PATCH /repos/owner/current/hooks/7/config":
					patches++
					order = append(order, "config")
					var p map[string]any
					json.NewDecoder(r.Body).Decode(&p)
					if mode != "mixed" || !reflect.DeepEqual(p, map[string]any{"content_type": "form"}) {
						t.Error("unexpected config mutation", p)
					}
					oldCfg["content_type"] = "form"
					json.NewEncoder(w).Encode(oldCfg)
				case "POST /repos/owner/current/hooks":
					posts++
					order = append(order, "create")
					var p map[string]any
					json.NewDecoder(r.Body).Decode(&p)
					events := []any{"future_event", "push"}
					if mode == "empty-events" {
						events = []any{}
					}
					url := "https://selected.example/new%2f?q=%2F"
					if posts == 2 {
						url = "https://selected.example/second"
					}
					want := map[string]any{"name": "web", "active": false, "events": events, "config": map[string]any{"url": url, "content_type": "json", "insecure_ssl": "0"}}
					if archived || !reflect.DeepEqual(p, want) {
						t.Error("creation payload/ordering", p)
					}
					if mode == "HTTP-error" || mode == "later-failure" && posts == 2 {
						w.WriteHeader(422)
						fmt.Fprint(w, `{"id":99,"message":"PRIVATE_CREATION_ERROR"}`)
						return
					}
					selectedEvents := []string{"future_event", "push"}
					if mode == "empty-events" {
						selectedEvents = []string{}
					}
					id := 7 + posts
					cfg := map[string]any{"url": url, "content_type": "json", "insecure_ssl": "0", "secret": "PRIVATE_CREATION_ACK_SECRET"}
					rows[id] = makeRow(id, cfg, false, selectedEvents)
					if mode == "file-drift" {
						if err := os.WriteFile(path, []byte("# concurrent source edit\n"), 0600); err != nil {
							t.Error(err)
						}
					}
					w.WriteHeader(201)
					if mode == "bad-ack" {
						fmt.Fprint(w, `{"id":8,"config":{"PRIVATE_CREATION_UNKNOWN":true}}`)
						return
					}
					json.NewEncoder(w).Encode(rows[id])
				default:
					if r.Method == "GET" && strings.HasPrefix(r.URL.Path, "/repos/owner/current/hooks/") {
						var id int
						fmt.Sscanf(strings.TrimPrefix(r.URL.Path, "/repos/owner/current/hooks/"), "%d", &id)
						if row, ok := rows[id]; ok {
							if id > 7 && mode == "readback-failure" {
								w.WriteHeader(403)
								fmt.Fprint(w, `{"message":"PRIVATE_CREATION_READBACK"}`)
								return
							}
							if strings.HasSuffix(r.URL.Path, "/config") {
								json.NewEncoder(w).Encode(row["config"])
							} else {
								json.NewEncoder(w).Encode(row)
							}
							return
						}
					}
					t.Error("unexpected/destination/lifecycle route", r.Method, r.URL)
					w.WriteHeader(404)
				}
			}, false, "/hooks", "/hooks/")
			if e != nil {
				t.Fatal(e)
			}
			t.Cleanup(f.close)
			env := append(labelIDFixtureEnv(f.cfg, mode == "hosted"), "GH_DEBUG=api")
			if e = setupOrdinaryOrigin(f.work, env); e != nil {
				t.Fatal(e)
			}
			entry := "{payload_url: 'https://selected.example/new%2f?q=%2F', content_type: application/json, active: false, events: [push, future_event]}"
			if mode == "empty-events" {
				entry = strings.Replace(entry, "[push, future_event]", "[]", 1)
			}
			entries := entry
			if mode == "mixed" {
				entries = "{id: 7, content_type: application/x-www-form-urlencoded, active: false, events: [push]}, " + entry
			}
			if mode == "guard" {
				entries = entry + ", {id: 7, content_type: application/x-www-form-urlencoded, active: true}"
			}
			if mode == "later-failure" {
				entries += ", " + strings.Replace(entry, "new%2f?q=%2F", "second", 1)
			}
			body := "settings: {webhooks: [" + entries + "]}"
			if mode == "own-unarchive" {
				body = "settings: {general: {danger_zone: {archive_this_repository: false}}, webhooks: [" + entries + "]}"
			}
			original := repositoryInput(body + " # keep source comment\n")
			path = filepath.Join(f.work, ".github/settings.yml")
			os.MkdirAll(filepath.Dir(path), 0700)
			os.WriteFile(path, []byte(original), 0600)
			snapshot := func() (int, int, int, []string) {
				mu.Lock()
				defer mu.Unlock()
				return posts, patches, lists, append([]string{}, order...)
			}
			plan, e := executeCLI(acceptanceBinary(t), f.work, f.root, env, []string{"plan"}, false)
			blocked := mode == "guard" || mode == "collision" || mode == "unavailable"
			if e != nil || (plan.runErr != nil) != blocked {
				t.Fatal(e, plan.runErr, plan.stderr.String())
			}
			if !blocked && (!strings.Contains(plan.stdout.String(), "automatic ping") || !strings.Contains(plan.stdout.String(), "no Secret supplied") || !strings.Contains(plan.stdout.String(), "selected.example")) {
				t.Fatal("unreviewable creation", plan.stdout.String())
			}
			if p, c, _, _ := snapshot(); p != 0 || c != 0 {
				t.Fatal("plan mutated")
			}
			args := []string{"apply"}
			if mode == "no-writeback" {
				args = append(args, "--no-writeback")
			}
			result, e := executeCLI(acceptanceBinary(t), f.work, f.root, env, args, false)
			failed := blocked || mode == "HTTP-error" || mode == "bad-ack" || mode == "readback-failure" || mode == "file-drift" || mode == "later-failure"
			if e != nil || (result.runErr != nil) != failed {
				t.Fatal(e, result.runErr, result.stdout.String(), result.stderr.String())
			}
			all := plan.stdout.String() + plan.stderr.String() + result.stdout.String() + result.stderr.String()
			if strings.Contains(all, "PRIVATE_CREATION") || strings.Contains(all, "existing.example") {
				t.Fatal("private/omitted evidence leaked", all)
			}
			data, _ := os.ReadFile(path)
			text := string(data)
			confirmed := !failed || mode == "file-drift" || mode == "later-failure"
			recorded := confirmed && mode != "file-drift" && mode != "no-writeback" && mode != "hosted"
			if recorded {
				if !strings.Contains(text, "id: 8") || !strings.Contains(text, "keep source comment") || !strings.Contains(text, "active: false") {
					t.Fatal("missing guarded recording", text)
				}
			} else if mode == "file-drift" {
				if text != "# concurrent source edit\n" {
					t.Fatal("overwrote concurrent source")
				}
			} else if text != original {
				t.Fatal("unconfirmed/action input changed", text)
			}
			p, c, l, sequence := snapshot()
			want := 1
			if blocked {
				want = 0
			}
			if mode == "later-failure" {
				want = 2
			}
			if p != want || c != map[bool]int{true: 1, false: 0}[mode == "mixed"] {
				t.Fatal("mutation counts", p, c)
			}
			if !blocked && l < 2 {
				t.Fatal("missing fresh inventory")
			}
			if mode == "own-unarchive" && !reflect.DeepEqual(sequence, []string{"unarchive", "create"}) {
				t.Fatal(sequence)
			}
			if mode == "mixed" && !reflect.DeepEqual(sequence, []string{"config", "create"}) {
				t.Fatal(sequence)
			}
			if mode == "bad-ack" || mode == "readback-failure" || mode == "file-drift" || mode == "later-failure" {
				if !strings.Contains(result.stderr.String(), "id 8") || !strings.Contains(result.stderr.String(), "retain this ID") {
					t.Fatal("recovery lost", result.stderr.String())
				}
			}
			if mode == "HTTP-error" && strings.Contains(result.stderr.String(), "id 99") {
				t.Fatal("error body invented recovery")
			}
			if !failed {
				again, e := executeCLI(acceptanceBinary(t), f.work, f.root, env, args, false)
				collision := mode == "no-writeback" || mode == "hosted"
				if e != nil || (again.runErr != nil) != collision {
					t.Fatal(e, again.runErr, again.stderr.String())
				}
				if now, _, _, _ := snapshot(); now != p {
					t.Fatal("repeat created hook")
				}
				if collision {
					bound := strings.Replace(original, "{payload_url:", "{id: 8, payload_url:", 1)
					os.WriteFile(path, []byte(bound), 0600)
					recovered, e := executeCLI(acceptanceBinary(t), f.work, f.root, env, args, false)
					if e != nil || recovered.runErr != nil {
						t.Fatal("manual binding recovery", e, recovered.stderr.String())
					}
				}
			}
		})
	}
}
