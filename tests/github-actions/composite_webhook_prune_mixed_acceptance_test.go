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

func TestBuiltCompositeWebhookPruneMixed(t *testing.T) {
	product, version := suppliedCompositeProduct(t)
	for _, scenario := range []string{"success", "delete-failure"} {
		t.Run(scenario, func(t *testing.T) {
			testCompositeWebhookPruneMixed(t, product, version, scenario)
		})
	}
}

func testCompositeWebhookPruneMixed(t *testing.T, product []byte, version, scenario string) {
	f := newCompositeFixture(t)
	f.base["GITHUB_ACTIONS"], f.base["GH_DEBUG"] = "true", "api"
	f.digest = fmt.Sprintf("%x", sha256.Sum256(product))
	f.write(f.base["MOCK_ASSET"], string(product))
	f.write(f.base["MOCK_CHECKSUMS"], f.digest+"  gh-settings-linux-amd64\n")
	f.write(filepath.Join(f.packagePath, "action/cli-version"), version+"\n")
	input := "version: daiksud/gh-settings@v0\nrepository: {settings: {webhooks: [{id: 7, content_type: application/x-www-form-urlencoded, ssl_verification: false, active: false, events: [push]}, {payload_url: https://created.example/hook, content_type: application/json, ssl_verification: false, active: false, events: [push]}]}}\n"
	path := filepath.Join(f.caller, ".github/settings.yml")
	f.write(path, input)
	dir, err := os.MkdirTemp("/tmp", "ghs-hook-mixed-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	listener, err := net.Listen("unix", filepath.Join(dir, "api.sock"))
	if err != nil {
		t.Fatal(err)
	}
	var mu sync.Mutex
	rows := map[int]map[string]any{}
	row := func(id int, url string) map[string]any {
		return map[string]any{"id": id, "name": "web", "type": "Repository", "url": fmt.Sprintf("https://api.github.com/repos/owner/config/hooks/%d", id), "created_at": "2026-11-14T00:00:00Z", "updated_at": "2026-11-14T01:00:00Z", "active": false, "events": []string{"push"}, "config": map[string]any{"url": url, "content_type": "json", "insecure_ssl": "0", "secret": "PRIVATE_MIXED_SECRET"}, "last_response": map[string]any{"message": "PRIVATE_MIXED_DELIVERY"}}
	}
	rows[7], rows[8] = row(7, "https://retained.example/old"), row(8, "https://removed.example/hook")
	writes := []string{}
	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		if r.Host != "api.github.com" || r.Header.Get("Authorization") != "token mixed-fixture" {
			t.Error("target/credential boundary")
			w.WriteHeader(400)
			return
		}
		switch r.Method + " " + r.URL.Path {
		case "GET /repos/owner/config":
			fmt.Fprint(w, `{"id":1,"full_name":"owner/config","visibility":"public","private":false,"archived":false,"owner":{"id":2,"login":"owner","type":"User"}}`)
		case "GET /repos/owner/config/hooks":
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
		case "PATCH /repos/owner/config/hooks/7/config":
			var body map[string]any
			json.NewDecoder(r.Body).Decode(&body)
			if !reflect.DeepEqual(body, map[string]any{"content_type": "form", "insecure_ssl": "1"}) {
				t.Error("unexpected config", body)
			}
			writes = append(writes, "update:7")
			rows[7]["config"].(map[string]any)["content_type"] = "form"
			rows[7]["config"].(map[string]any)["insecure_ssl"] = "1"
			rows[7]["updated_at"] = "2026-11-14T02:00:00Z"
			json.NewEncoder(w).Encode(rows[7]["config"])
		case "POST /repos/owner/config/hooks":
			var body map[string]any
			json.NewDecoder(r.Body).Decode(&body)
			want := map[string]any{"name": "web", "active": false, "events": []any{"push"}, "config": map[string]any{"url": "https://created.example/hook", "content_type": "json", "insecure_ssl": "1"}}
			if !reflect.DeepEqual(body, want) {
				t.Error("unexpected create payload", body)
			}
			writes = append(writes, "create:9")
			rows[9] = row(9, "https://created.example/hook")
			rows[9]["config"].(map[string]any)["insecure_ssl"] = "1"
			w.WriteHeader(201)
			json.NewEncoder(w).Encode(rows[9])
		case "DELETE /repos/owner/config/hooks/8":
			writes = append(writes, "delete:8")
			if scenario == "delete-failure" {
				w.WriteHeader(422)
				fmt.Fprint(w, `{"message":"PRIVATE_MIXED_DELETE_FAILURE"}`)
				return
			}
			delete(rows, 8)
			w.WriteHeader(204)
		default:
			if r.Method == "GET" && strings.HasPrefix(r.URL.Path, "/repos/owner/config/hooks/") {
				var id int
				fmt.Sscanf(strings.TrimPrefix(r.URL.Path, "/repos/owner/config/hooks/"), "%d", &id)
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
	})}
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(func() { server.Close() })
	f.write(filepath.Join(dir, "config.yml"), "http_unix_socket: "+filepath.Join(dir, "api.sock")+"\n")
	f.base["GH_CONFIG_DIR"], f.base["GH_PATH_DISABLED"] = dir, "1"
	for _, k := range []string{"HTTP_PROXY", "HTTPS_PROXY", "http_proxy", "https_proxy"} {
		f.base[k] = "http://127.0.0.1:1"
	}
	f.base["NO_PROXY"], f.base["no_proxy"] = "", ""
	opts := map[string]string{"github.action_ref": version, "inputs.prune": "webhooks"}
	observedText := func(result compositeResult) string {
		var text strings.Builder
		text.WriteString(result.log)
		text.WriteString(result.summary)
		for name, value := range result.outputs {
			text.WriteString(name)
			text.WriteString(value)
		}
		return text.String()
	}
	plan := f.run(t, "plan", "mixed-fixture", opts)
	if plan.status != 0 || !strings.Contains(plan.summary, "removed.example") || !strings.Contains(plan.summary, "automatic ping") || !strings.Contains(plan.summary, "opaque Secret") || !strings.Contains(plan.summary, "man-in-the-middle") || strings.Contains(observedText(plan), "PRIVATE_MIXED") {
		t.Fatal(plan)
	}
	private := f.run(t, "plan", "mixed-fixture", map[string]string{"github.action_ref": version, "inputs.prune": "all", "inputs.plan-reporting": "status-only"})
	if private.status != 0 || private.outputs["plan-outcome"] != "changed" || len(private.remainingReports) != 0 || strings.Contains(observedText(private), "removed.example") || strings.Contains(observedText(private), "created.example") || strings.Contains(observedText(private), "PRIVATE_MIXED") || strings.Contains(observedText(private), "ssl_verification") {
		t.Fatal(private)
	}
	mu.Lock()
	planWrites := len(writes)
	mu.Unlock()
	if planWrites != 0 {
		t.Fatal("plan mutated webhook state")
	}
	apply := f.run(t, "apply", "mixed-fixture", opts)
	failed := scenario == "delete-failure"
	if (apply.status != 0) != failed || (failed && (apply.status != 1 || len(apply.outputs) != 0)) || (!failed && len(apply.outputs) != 4) || strings.Contains(observedText(apply), "PRIVATE_MIXED") {
		t.Fatal(apply)
	}
	if failed && (!strings.Contains(apply.log, "id 9") || !strings.Contains(apply.log, "retain this ID") || !strings.Contains(apply.log, "effects unconfirmed")) {
		t.Fatal("failed prune lost recovery or uncertain-effect diagnostics", apply)
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != input || f.git("rev-parse", "HEAD") != f.sha {
		t.Fatal("action changed settings file or checkout HEAD", err)
	}
	mu.Lock()
	defer mu.Unlock()
	if !reflect.DeepEqual(writes, []string{"update:7", "create:9", "delete:8"}) || rows[9] == nil || (rows[8] != nil) != failed || rows[7]["config"].(map[string]any)["content_type"] != "form" || rows[7]["config"].(map[string]any)["insecure_ssl"] != "1" || rows[9]["config"].(map[string]any)["url"] != "https://created.example/hook" {
		t.Fatal("wrong order or retained state", writes)
	}
}
