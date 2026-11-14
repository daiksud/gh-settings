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

func TestBuiltCompositeWebhooks(t *testing.T) {
	product, version := suppliedCompositeProduct(t)
	for _, scenario := range []string{"content", "url", "matching", "readback-failure", "HTTP-error", "malformed", "export"} {
		t.Run(scenario, func(t *testing.T) {
			f := newCompositeFixture(t)
			f.base["GITHUB_ACTIONS"], f.base["GH_DEBUG"] = "true", "api"
			f.digest = fmt.Sprintf("%x", sha256.Sum256(product))
			f.write(f.base["MOCK_ASSET"], string(product))
			f.write(f.base["MOCK_CHECKSUMS"], f.digest+"  gh-settings-linux-amd64\n")
			f.write(filepath.Join(f.packagePath, "action/cli-version"), version+"\n")
			entry := "{id: 7, content_type: application/x-www-form-urlencoded}"
			if scenario == "url" {
				entry = "{id: 7, payload_url: 'https://selected.example/next%2f?q=%2F'}"
			}
			if scenario == "matching" {
				entry = "{id: 7, content_type: application/json}"
			}
			input := "version: daiksud/gh-settings@v0\nrepository: {settings: {webhooks: [" + entry + "]}}\n"
			path := filepath.Join(f.caller, ".github/settings.yml")
			f.write(path, input)
			dir, e := os.MkdirTemp("/tmp", "ghs-hooks-")
			if e != nil {
				t.Fatal(e)
			}
			t.Cleanup(func() { os.RemoveAll(dir) })
			listener, e := net.Listen("unix", filepath.Join(dir, "api.sock"))
			if e != nil {
				t.Fatal(e)
			}
			cfg := map[string]any{"url": "https://omitted.example/old", "content_type": "json", "insecure_ssl": "1", "secret": "PRIVATE_WEBHOOK_SECRET"}
			patches := 0
			var mu sync.Mutex
			snapshot := func() int { mu.Lock(); defer mu.Unlock(); return patches }
			row := func() map[string]any {
				return map[string]any{"id": 7, "type": "Repository", "name": "web", "url": "https://api.github.com/repos/owner/config/hooks/7", "created_at": "2026-11-14T00:00:00Z", "active": false, "events": []string{"push"}, "config": cfg, "last_response": map[string]any{"message": "PRIVATE_WEBHOOK_DELIVERY"}}
			}
			server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				w.Header().Set("Content-Type", "application/json")
				if r.Host != "api.github.com" || r.Header.Get("Authorization") != "token webhook-installation-fixture-token" {
					t.Error("credential/target boundary")
					w.WriteHeader(400)
					return
				}
				switch r.Method + " " + r.URL.Path {
				case "GET /repos/owner/config":
					fmt.Fprint(w, `{"id":1,"full_name":"owner/config","private":false,"visibility":"public","archived":false,"owner":{"id":2,"login":"owner","type":"User"}}`)
				case "GET /repos/owner/config/hooks":
					json.NewEncoder(w).Encode([]any{row()})
				case "GET /repos/owner/config/hooks/7":
					if scenario == "readback-failure" && patches > 0 {
						w.WriteHeader(403)
						fmt.Fprint(w, `{"message":"PRIVATE_WEBHOOK_READBACK"}`)
						return
					}
					json.NewEncoder(w).Encode(row())
				case "GET /repos/owner/config/hooks/7/config":
					json.NewEncoder(w).Encode(cfg)
				case "PATCH /repos/owner/config/hooks/7/config":
					patches++
					var p map[string]any
					json.NewDecoder(r.Body).Decode(&p)
					want := map[string]any{"content_type": "form"}
					if scenario == "url" {
						want = map[string]any{"url": "https://selected.example/next%2f?q=%2F"}
					}
					if !reflect.DeepEqual(p, want) {
						t.Error("nonselected or secret payload", p)
					}
					if scenario == "HTTP-error" {
						w.WriteHeader(422)
						fmt.Fprint(w, `{"message":"PRIVATE_WEBHOOK_UPSTREAM"}`)
						return
					}
					for k, v := range p {
						cfg[k] = v
					}
					if scenario == "malformed" {
						fmt.Fprint(w, `{"PRIVATE_WEBHOOK_UNKNOWN":true}`)
						return
					}
					cfg["secret"] = "PRIVATE_WEBHOOK_CHANGED_MASK"
					json.NewEncoder(w).Encode(cfg)
				default:
					if scenario != "export" {
						t.Error("unexpected or destination route", r.Method, r.URL)
					}
					w.WriteHeader(403)
					fmt.Fprint(w, `{"message":"unavailable"}`)
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
			if scenario == "export" {
				os.Remove(path)
				r := f.run(t, "export", "webhook-installation-fixture-token", options)
				if r.status != 0 || r.outputs["config-path"] != ".github/settings.yml" || strings.Contains(r.log+r.summary, "omitted.example") || strings.Contains(r.log+r.summary, "PRIVATE_WEBHOOK") {
					t.Fatal(r)
				}
				data, _ := os.ReadFile(path)
				if !strings.Contains(string(data), "payload_url: https://omitted.example/old") || !strings.Contains(string(data), "content_type: application/json") || strings.Contains(string(data), "secret:") || snapshot() != 0 {
					t.Fatal("export projection", string(data))
				}
				return
			}
			for _, reporting := range []string{"detailed", "status-only"} {
				options["inputs.plan-reporting"] = reporting
				r := f.run(t, "plan", "webhook-installation-fixture-token", options)
				if r.status != 0 || strings.Contains(r.log+r.summary, "PRIVATE_WEBHOOK") {
					t.Fatal(r)
				}
				if reporting == "status-only" && (strings.Contains(r.log+r.summary, "example") || strings.Contains(r.summary, "id 7")) {
					t.Fatal("status-only leaked", r)
				}
				if reporting == "detailed" && scenario == "url" && !strings.Contains(r.summary, "selected.example") {
					t.Fatal("redirect unreviewable", r.summary)
				}
				if scenario != "url" && strings.Contains(r.summary, "omitted.example") {
					t.Fatal("omitted URL exposed", r.summary)
				}
				if len(r.remainingReports) != 0 {
					t.Fatal("private plan retained")
				}
			}
			delete(options, "inputs.plan-reporting")
			if snapshot() != 0 {
				t.Fatal("plan mutated")
			}
			r := f.run(t, "apply", "webhook-installation-fixture-token", options)
			failed := scenario == "readback-failure" || scenario == "HTTP-error" || scenario == "malformed"
			if (r.status != 0) != failed || failed && len(r.outputs) != 0 || !failed && len(r.outputs) != 4 {
				t.Fatal(r)
			}
			if strings.Contains(r.log+r.summary, "PRIVATE_WEBHOOK") {
				t.Fatal("private API evidence leaked", r)
			}
			data, _ := os.ReadFile(path)
			if string(data) != input || f.git("rev-parse", "HEAD") != f.sha {
				t.Fatal("input/checkout changed")
			}
			want := 1
			if scenario == "matching" {
				want = 0
			}
			if snapshot() != want {
				t.Fatal("write count", snapshot())
			}
			if !failed {
				r = f.run(t, "apply", "webhook-installation-fixture-token", options)
				if r.status != 0 || snapshot() != want {
					t.Fatal("matching repeat wrote", r)
				}
			}
		})
	}
}
