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
	"strings"
	"sync"
	"testing"
)

func TestBuiltCompositeWebhookCreation(t *testing.T) {
	product, version := suppliedCompositeProduct(t)
	for _, scenario := range []string{"create", "empty-events", "collision", "guard", "HTTP-error", "bad-ack", "readback-failure", "later-failure"} {
		t.Run(scenario, func(t *testing.T) {
			f := newCompositeFixture(t)
			f.base["GITHUB_ACTIONS"], f.base["GH_DEBUG"] = "true", "api"
			f.digest = fmt.Sprintf("%x", sha256.Sum256(product))
			f.write(f.base["MOCK_ASSET"], string(product))
			f.write(f.base["MOCK_CHECKSUMS"], f.digest+"  gh-settings-linux-amd64\n")
			f.write(filepath.Join(f.packagePath, "action/cli-version"), version+"\n")
			entry := "{payload_url: 'https://selected.example/new%2f?q=%2F', content_type: application/json, active: false, events: [push, future_event]}"
			if scenario == "empty-events" {
				entry = strings.Replace(entry, "[push, future_event]", "[]", 1)
			}
			entries := entry
			if scenario == "guard" {
				entries += ", {id: 7, content_type: application/x-www-form-urlencoded, active: true}"
			}
			if scenario == "later-failure" {
				entries += ", " + strings.Replace(entry, "new%2f?q=%2F", "second", 1)
			}
			input := "version: daiksud/gh-settings@v0\nrepository: {settings: {webhooks: [" + entries + "]}}\n"
			path := filepath.Join(f.caller, ".github/settings.yml")
			f.write(path, input)
			dir, e := os.MkdirTemp("/tmp", "ghs-hook-create-")
			if e != nil {
				t.Fatal(e)
			}
			t.Cleanup(func() { os.RemoveAll(dir) })
			listener, e := net.Listen("unix", filepath.Join(dir, "api.sock"))
			if e != nil {
				t.Fatal(e)
			}
			var mu sync.Mutex
			posts := 0
			snapshot := func() int { mu.Lock(); defer mu.Unlock(); return posts }
			rows := map[int]map[string]any{}
			row := func(id int, cfg map[string]any, events []string) map[string]any {
				return map[string]any{"id": id, "type": "Repository", "name": "web", "url": fmt.Sprintf("https://api.github.com/repos/owner/config/hooks/%d", id), "created_at": "2026-11-14T00:00:00Z", "active": false, "events": events, "config": cfg, "last_response": map[string]any{"message": "PRIVATE_CREATE_DELIVERY"}}
			}
			oldURL := "https://omitted.example/old"
			if scenario == "collision" {
				oldURL = "https://selected.example/new%2f?q=%2F"
			}
			rows[7] = row(7, map[string]any{"url": oldURL, "content_type": "json", "insecure_ssl": "1", "secret": "PRIVATE_CREATE_OLD_SECRET"}, []string{"push"})
			server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				w.Header().Set("Content-Type", "application/json")
				if r.Host != "api.github.com" || r.Header.Get("Authorization") != "token webhook-creation-fixture-token" {
					t.Error("credential/target boundary")
					w.WriteHeader(400)
					return
				}
				switch r.Method + " " + r.URL.Path {
				case "GET /repos/owner/config":
					fmt.Fprint(w, `{"id":1,"full_name":"owner/config","private":false,"visibility":"public","archived":false,"owner":{"id":2,"login":"owner","type":"User"}}`)
				case "GET /repos/owner/config/hooks":
					values := []any{}
					for _, id := range []int{7, 8, 9} {
						if v, ok := rows[id]; ok {
							values = append(values, v)
						}
					}
					json.NewEncoder(w).Encode(values)
				case "POST /repos/owner/config/hooks":
					posts++
					var p map[string]any
					json.NewDecoder(r.Body).Decode(&p)
					events := []any{"future_event", "push"}
					names := []string{"future_event", "push"}
					if scenario == "empty-events" {
						events = []any{}
						names = []string{}
					}
					url := "https://selected.example/new%2f?q=%2F"
					if posts == 2 {
						url = "https://selected.example/second"
					}
					want := map[string]any{"name": "web", "active": false, "events": events, "config": map[string]any{"url": url, "content_type": "json", "insecure_ssl": "0"}}
					if !reflect.DeepEqual(p, want) {
						t.Error("unexpected/secret creation payload", p)
					}
					if scenario == "HTTP-error" || scenario == "later-failure" && posts == 2 {
						w.WriteHeader(422)
						fmt.Fprint(w, `{"id":99,"message":"PRIVATE_CREATE_HTTP"}`)
						return
					}
					id := 7 + posts
					rows[id] = row(id, map[string]any{"url": url, "content_type": "json", "insecure_ssl": "0", "secret": "PRIVATE_CREATE_NEW_SECRET"}, names)
					w.WriteHeader(201)
					if scenario == "bad-ack" {
						fmt.Fprint(w, `{"id":8,"PRIVATE_CREATE_UNKNOWN":true}`)
						return
					}
					json.NewEncoder(w).Encode(rows[id])
				default:
					if r.Method == "GET" && strings.HasPrefix(r.URL.Path, "/repos/owner/config/hooks/") {
						var id int
						fmt.Sscanf(strings.TrimPrefix(r.URL.Path, "/repos/owner/config/hooks/"), "%d", &id)
						if v, ok := rows[id]; ok {
							if id > 7 && scenario == "readback-failure" {
								w.WriteHeader(403)
								fmt.Fprint(w, `{"message":"PRIVATE_CREATE_READBACK"}`)
								return
							}
							if strings.HasSuffix(r.URL.Path, "/config") {
								json.NewEncoder(w).Encode(v["config"])
							} else {
								json.NewEncoder(w).Encode(v)
							}
							return
						}
					}
					t.Error("forbidden/destination route", r.Method, r.URL)
					w.WriteHeader(404)
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
			options := map[string]string{"github.action_ref": version}
			blocked := scenario == "guard" || scenario == "collision"
			for _, privacy := range []string{"detailed", "status-only"} {
				options["inputs.plan-reporting"] = privacy
				result := f.run(t, "plan", "webhook-creation-fixture-token", options)
				if (result.status != 0) != blocked || len(result.remainingReports) != 0 {
					t.Fatal(result)
				}
				if blocked {
					if len(result.outputs) != 0 {
						t.Fatal("failure published success outputs", result)
					}
				} else {
					want := map[string]string{"config-sha": f.sha, "target": "owner/config", "cli-version": version, "cli-sha256": f.digest}
					if privacy == "status-only" {
						want["plan-outcome"] = "changed"
					}
					if !reflect.DeepEqual(result.outputs, want) {
						t.Fatal("unexpected outputs", result.outputs)
					}
					if privacy == "detailed" && (!strings.Contains(result.summary, "selected.example") || !strings.Contains(result.summary, "automatic ping") || !strings.Contains(result.summary, "no Secret supplied") || !strings.Contains(result.summary, "active: false")) {
						t.Fatal("creation unreviewable", result.summary)
					}
				}
				if strings.Contains(result.log+result.summary, "PRIVATE_CREATE") || strings.Contains(result.log+result.summary, "omitted.example") {
					t.Fatal("private evidence leaked", result)
				}
				if privacy == "status-only" && (strings.Contains(result.log+result.summary, "selected.example") || strings.Contains(result.summary, "future_event")) {
					t.Fatal("status-only leaked", result)
				}
			}
			if snapshot() != 0 {
				t.Fatal("plan mutated")
			}
			delete(options, "inputs.plan-reporting")
			result := f.run(t, "apply", "webhook-creation-fixture-token", options)
			failed := blocked || scenario == "HTTP-error" || scenario == "bad-ack" || scenario == "readback-failure" || scenario == "later-failure"
			if (result.status != 0) != failed {
				t.Fatal(result)
			}
			if failed {
				if len(result.outputs) != 0 {
					t.Fatal("failed apply published success outputs", result)
				}
			} else {
				want := map[string]string{"config-sha": f.sha, "target": "owner/config", "cli-version": version, "cli-sha256": f.digest}
				if !reflect.DeepEqual(result.outputs, want) {
					t.Fatal("unexpected outputs", result.outputs)
				}
			}
			if strings.Contains(result.log+result.summary, "PRIVATE_CREATE") || strings.Contains(result.log+result.summary, "omitted.example") {
				t.Fatal("private evidence leaked", result)
			}
			data, _ := os.ReadFile(path)
			if string(data) != input || f.git("rev-parse", "HEAD") != f.sha {
				t.Fatal("Action changed source/checkout")
			}
			want := 1
			if blocked {
				want = 0
			}
			if scenario == "later-failure" {
				want = 2
			}
			if snapshot() != want {
				t.Fatal("write count", snapshot())
			}
			if scenario == "bad-ack" || scenario == "readback-failure" || scenario == "later-failure" {
				if !strings.Contains(result.log, "id 8") || !strings.Contains(result.log, "retain this ID") {
					t.Fatal("recovery lost", result)
				}
			}
			if scenario == "HTTP-error" && strings.Contains(result.log, "id 99") {
				t.Fatal("error response invented identity", result)
			}
			if !failed {
				again := f.run(t, "apply", "webhook-creation-fixture-token", options)
				if again.status == 0 || len(again.outputs) != 0 || snapshot() != want {
					t.Fatal("IDless repeat did not stop", again)
				}
				f.write(path, strings.Replace(input, "{payload_url:", "{id: 8, payload_url:", 1))
				bound := f.run(t, "apply", "webhook-creation-fixture-token", options)
				if bound.status != 0 || snapshot() != want {
					t.Fatal("manual binding did not noop", bound)
				}
			}
		})
	}
}
