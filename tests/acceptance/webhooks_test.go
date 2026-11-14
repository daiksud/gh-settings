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

func TestWebhooksExecutableLifecycle(t *testing.T) {
	for _, mode := range []string{"content", "url", "both", "matching", "empty", "omitted", "own-unarchive", "readback-failure", "HTTP-error", "unknown-config", "invalid-url"} {
		t.Run(mode, func(t *testing.T) {
			var mu sync.Mutex
			patches := 0
			archived := mode == "own-unarchive"
			events := []string{}
			cfg := map[string]any{"url": "https://old.example/private-omitted", "content_type": "json", "insecure_ssl": "1", "secret": "PRIVATE_SECRET_WEBHOOK"}
			row := func() map[string]any {
				return map[string]any{"id": 7, "type": "Repository", "name": "web", "active": false, "events": []string{"push", "future_event"}, "created_at": "2026-11-14T00:00:00Z", "url": "https://api.github.com/repos/owner/current/hooks/7", "config": cfg, "last_response": map[string]any{"message": "PRIVATE_DELIVERY_WEBHOOK"}}
			}
			f, e := startHTTPFixture(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				w.Header().Set("Content-Type", "application/json")
				switch r.Method + " " + r.URL.Path {
				case "GET /repos/owner/current":
					json.NewEncoder(w).Encode(map[string]any{"id": 1, "full_name": "owner/current", "visibility": "public", "private": false, "archived": archived, "owner": map[string]any{"id": 2, "login": "owner", "type": "User"}})
				case "PATCH /repos/owner/current":
					var p map[string]any
					json.NewDecoder(r.Body).Decode(&p)
					if mode != "own-unarchive" || !reflect.DeepEqual(p, map[string]any{"archived": false}) {
						t.Error("unexpected repository mutation")
					}
					archived = false
					events = append(events, "unarchive")
					fmt.Fprint(w, `{"id":1,"full_name":"owner/current","archived":false}`)
				case "GET /repos/owner/current/hooks":
					json.NewEncoder(w).Encode([]any{row()})
				case "GET /repos/owner/current/hooks/7":
					if mode == "readback-failure" && patches > 0 {
						w.WriteHeader(403)
						fmt.Fprint(w, `{"message":"PRIVATE_UPSTREAM_WEBHOOK"}`)
						return
					}
					json.NewEncoder(w).Encode(row())
				case "GET /repos/owner/current/hooks/7/config":
					if mode == "unknown-config" {
						cfg["PRIVATE_UNKNOWN_WEBHOOK"] = true
					}
					json.NewEncoder(w).Encode(cfg)
				case "PATCH /repos/owner/current/hooks/7/config":
					patches++
					events = append(events, "config")
					if archived {
						t.Error("write before unarchive")
					}
					var p map[string]any
					json.NewDecoder(r.Body).Decode(&p)
					want := map[string]any{"content_type": "form"}
					if mode == "url" {
						want = map[string]any{"url": "https://NEW.example:8443/a%2Fb?q=%2f"}
					}
					if mode == "both" {
						want["url"] = "https://NEW.example:8443/a%2Fb?q=%2f"
					}
					if !reflect.DeepEqual(p, want) {
						t.Error("payload contains omitted/secret state", p)
					}
					if mode == "HTTP-error" {
						w.WriteHeader(422)
						fmt.Fprint(w, `{"message":"PRIVATE_ERROR_WEBHOOK"}`)
						return
					}
					for k, v := range p {
						cfg[k] = v
					}
					cfg["secret"] = "PRIVATE_CHANGED_SECRET"
					json.NewEncoder(w).Encode(cfg)
				default:
					t.Error("unexpected/destination route", r.Method, r.URL)
					w.WriteHeader(404)
				}
			}, false, "/hooks", "/hooks/")
			if e != nil {
				t.Fatal(e)
			}
			t.Cleanup(f.close)
			env := append(labelIDFixtureEnv(f.cfg, false), "GH_DEBUG=api")
			if e = setupOrdinaryOrigin(f.work, env); e != nil {
				t.Fatal(e)
			}
			entry := "{id: 7, content_type: application/x-www-form-urlencoded}"
			if mode == "url" {
				entry = "{id: 7, payload_url: 'https://NEW.example:8443/a%2Fb?q=%2f'}"
			}
			if mode == "both" {
				entry = "{id: 7, payload_url: 'https://NEW.example:8443/a%2Fb?q=%2f', content_type: application/x-www-form-urlencoded}"
			}
			if mode == "matching" {
				entry = "{id: 7, content_type: application/json}"
			}
			if mode == "invalid-url" {
				entry = "{id: 7, payload_url: 'https://user:password@example.com'}"
			}
			input := "settings: {webhooks: [" + entry + "]}"
			if mode == "empty" {
				input = "settings: {webhooks: []}"
			}
			if mode == "omitted" {
				input = "settings: {}"
			}
			if mode == "own-unarchive" {
				input = "settings: {webhooks: [" + entry + "], general: {danger_zone: {archive_this_repository: false}}}"
			}
			original := repositoryInput(input + "\n")
			path := filepath.Join(f.work, ".github/settings.yml")
			os.MkdirAll(filepath.Dir(path), 0700)
			os.WriteFile(path, []byte(original), 0600)
			plan, e := executeCLI(acceptanceBinary(t), f.work, f.root, env, []string{"plan"}, false)
			bad := mode == "unknown-config" || mode == "invalid-url"
			if e != nil || (plan.runErr != nil) != bad {
				t.Fatal(e, plan.runErr, plan.stderr.String())
			}
			if !bad && mode != "url" && mode != "both" && strings.Contains(plan.stdout.String(), "private-omitted") {
				t.Fatal("omitted URL leaked")
			}
			mu.Lock()
			before := patches
			mu.Unlock()
			if before != 0 {
				t.Fatal("plan mutated")
			}
			result, e := executeCLI(acceptanceBinary(t), f.work, f.root, env, []string{"apply"}, false)
			failed := bad || mode == "readback-failure" || mode == "HTTP-error"
			if e != nil || (result.runErr != nil) != failed {
				t.Fatal(e, result.runErr, result.stderr.String())
			}
			if strings.Contains(result.stdout.String()+result.stderr.String()+plan.stderr.String(), "PRIVATE_") {
				t.Fatal("private API evidence leaked", result.stderr.String())
			}
			data, _ := os.ReadFile(path)
			if string(data) != original {
				t.Fatal("input changed")
			}
			mu.Lock()
			count := patches
			gotEvents := append([]string{}, events...)
			mu.Unlock()
			wantCount := 1
			if bad || mode == "matching" || mode == "empty" || mode == "omitted" {
				wantCount = 0
			}
			if count != wantCount {
				t.Fatal("write count", count, wantCount)
			}
			if mode == "own-unarchive" && !reflect.DeepEqual(gotEvents, []string{"unarchive", "config"}) {
				t.Fatal(gotEvents)
			}
			if !failed {
				again, e := executeCLI(acceptanceBinary(t), f.work, f.root, env, []string{"apply"}, false)
				if e != nil || again.runErr != nil {
					t.Fatal(e, again.runErr, again.stderr.String())
				}
				mu.Lock()
				after := patches
				mu.Unlock()
				if after != count {
					t.Fatal("repeat mutated")
				}
			}
		})
	}
}
func TestWebhooksExecutableExportRetention(t *testing.T) {
	for _, mode := range []string{"complete", "empty", "initial", "retained", "replaced-empty-retained", "replaced-empty-initial", "short-next-retained", "short-next-initial"} {
		t.Run(mode, func(t *testing.T) {
			var mu sync.Mutex
			repositoryID := 1
			cfg := map[string]any{"url": "https://example.test/exact%2f?q=%2F", "content_type": "form", "insecure_ssl": 0, "secret": "PRIVATE_EXPORT_SECRET"}
			row := map[string]any{"id": 7, "type": "Repository", "name": "web", "active": true, "events": []string{"push"}, "created_at": "2026-11-14T00:00:00Z", "url": "https://api.github.com/repos/owner/current/hooks/7", "config": cfg}
			f, e := startHTTPFixture(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				w.Header().Set("Content-Type", "application/json")
				if r.Method != "GET" {
					if isMergeMethodsFixtureRead(r) || isPRAccessFixtureRead(r) || isIssueCreationPolicyFixtureRead(r) || isSponsorshipsFixtureRead(r) {
						w.WriteHeader(403)
						return
					}
					t.Error("export wrote")
					w.WriteHeader(400)
					return
				}
				switch r.URL.Path {
				case "/repos/owner/current":
					fmt.Fprintf(w, `{"id":%d,"full_name":"owner/current","visibility":"public","private":false,"archived":false,"owner":{"id":2,"login":"owner","type":"User"}}`, repositoryID)
				case "/repos/owner/current/hooks":
					if strings.HasPrefix(mode, "replaced-empty") {
						repositoryID = 9
					}
					if strings.HasPrefix(mode, "short-next") {
						w.Header().Set("Link", `<https://foreign.example/PRIVATE_PAGINATION>; rel=next`)
					}
					if mode == "empty" || strings.HasPrefix(mode, "replaced-empty") {
						fmt.Fprint(w, `[]`)
					} else {
						json.NewEncoder(w).Encode([]any{row})
					}
				case "/repos/owner/current/hooks/7":
					json.NewEncoder(w).Encode(row)
				case "/repos/owner/current/hooks/7/config":
					if mode == "initial" || mode == "retained" {
						w.WriteHeader(403)
						fmt.Fprint(w, `{"message":"PRIVATE_EXPORT_ERROR"}`)
					} else {
						json.NewEncoder(w).Encode(cfg)
					}
				case "/repos/owner/current/topics":
					fmt.Fprint(w, `{"names":["independent"]}`)
				default:
					w.WriteHeader(403)
					fmt.Fprint(w, `{"message":"unavailable"}`)
				}
			}, false, "/hooks", "/hooks/")
			if e != nil {
				t.Fatal(e)
			}
			t.Cleanup(f.close)
			env := append(labelIDFixtureEnv(f.cfg, false), "GH_DEBUG=api")
			if e = setupOrdinaryOrigin(f.work, env); e != nil {
				t.Fatal(e)
			}
			path := filepath.Join(f.work, ".github/settings.yml")
			if strings.HasSuffix(mode, "retained") {
				os.MkdirAll(filepath.Dir(path), 0700)
				os.WriteFile(path, []byte(repositoryInput("settings: {webhooks: [{id: 9, content_type: application/json}]}\n")), 0600)
			}
			r, e := executeCLI(acceptanceBinary(t), f.work, f.root, env, []string{"export"}, false)
			if e != nil || r.runErr != nil {
				t.Fatal(e, r.runErr, r.stderr.String())
			}
			data, _ := os.ReadFile(path)
			text := string(data)
			if !strings.Contains(text, "independent") || strings.Contains(text+r.stderr.String()+r.stdout.String(), "PRIVATE_") || strings.Contains(text, "secret:") {
				t.Fatal("export privacy/independence", text, r.stderr.String())
			}
			want := "payload_url: https://example.test/exact%2f?q=%2F"
			if mode == "empty" {
				want = "webhooks: []"
			}
			if strings.HasSuffix(mode, "retained") {
				want = "id: 9"
			}
			if strings.HasSuffix(mode, "initial") {
				if strings.Contains(text, "webhooks:") {
					t.Fatal("guessed unavailable")
				}
			} else if !strings.Contains(text, want) {
				t.Fatal(text, r.stderr.String())
			}
			v, e := executeCLI(acceptanceBinary(t), f.work, f.root, env, []string{"validate"}, true)
			if e != nil || v.runErr != nil {
				t.Fatal(e, v.runErr, v.stderr.String())
			}
		})
	}
}
