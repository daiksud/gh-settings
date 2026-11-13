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

func TestBuiltCompositeAutolinks(t *testing.T) {
	product, version := suppliedCompositeProduct(t)
	for _, scenario := range []string{"matching", "create-prune", "empty-all", "readback-failure", "export"} {
		t.Run(scenario, func(t *testing.T) {
			f := newCompositeFixture(t)
			f.base["GITHUB_ACTIONS"] = "true"
			f.base["GH_DEBUG"] = "api"
			f.digest = fmt.Sprintf("%x", sha256.Sum256(product))
			f.write(f.base["MOCK_ASSET"], string(product))
			f.write(f.base["MOCK_CHECKSUMS"], f.digest+"  gh-settings-linux-amd64\n")
			f.write(filepath.Join(f.packagePath, "action/cli-version"), version+"\n")
			entry := "{reference_prefix: NEW_PRIVATE-, target_url: 'https://private.example/<num>?Case=A%2FB', reference_format: numeric}"
			if scenario == "matching" {
				entry = "{id: 7, reference_prefix: OLD_PRIVATE-, target_url: 'https://private.example/<num>?Case=A%2FB', reference_format: numeric}"
			}
			input := "version: daiksud/gh-settings@v0\nrepository: {settings: {autolinks: [" + entry + "]}}\n"
			if scenario == "empty-all" {
				input = "version: daiksud/gh-settings@v0\nrepository: {settings: {autolinks: []}}\n"
			}
			path := filepath.Join(f.caller, ".github/settings.yml")
			f.write(path, input)
			directory, err := os.MkdirTemp("/tmp", "ghs-autolink-")
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { os.RemoveAll(directory) })
			listener, err := net.Listen("unix", filepath.Join(directory, "api.sock"))
			if err != nil {
				t.Fatal(err)
			}
			row := func(id int, prefix string) map[string]any {
				return map[string]any{"id": id, "key_prefix": prefix, "url_template": "https://private.example/<num>?Case=A%2FB", "is_alphanumeric": false}
			}
			rows := map[int]map[string]any{7: row(7, "OLD_PRIVATE-")}
			events := []string{}
			var mu sync.Mutex
			server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				w.Header().Set("Content-Type", "application/json")
				if r.Host != "api.github.com" || r.Header.Get("Authorization") != "token autolink-installation-fixture-token" {
					t.Error("credential or target escaped")
					w.WriteHeader(400)
					return
				}
				switch r.Method + " " + r.URL.Path {
				case "GET /repos/owner/config":
					fmt.Fprint(w, `{"id":1,"full_name":"owner/config","private":false,"archived":false,"owner":{"id":2,"login":"owner","type":"User"}}`)
				case "GET /repos/owner/config/autolinks":
					values := []any{}
					for _, v := range rows {
						values = append(values, v)
					}
					json.NewEncoder(w).Encode(values)
				case "GET /repos/owner/config/autolinks/7":
					json.NewEncoder(w).Encode(rows[7])
				case "GET /repos/owner/config/autolinks/8":
					if scenario == "readback-failure" {
						w.WriteHeader(403)
						fmt.Fprint(w, `{"message":"UPSTREAM_PRIVATE_AUTOLINK"}`)
					} else {
						json.NewEncoder(w).Encode(rows[8])
					}
				case "POST /repos/owner/config/autolinks":
					var data map[string]any
					json.NewDecoder(r.Body).Decode(&data)
					if !reflect.DeepEqual(data, map[string]any{"key_prefix": "NEW_PRIVATE-", "url_template": "https://private.example/<num>?Case=A%2FB", "is_alphanumeric": false}) {
						t.Error(data)
					}
					events = append(events, "create")
					rows[8] = row(8, "NEW_PRIVATE-")
					w.WriteHeader(201)
					json.NewEncoder(w).Encode(rows[8])
				case "DELETE /repos/owner/config/autolinks/7":
					events = append(events, "delete")
					delete(rows, 7)
					w.WriteHeader(204)
				default:
					if scenario != "export" {
						t.Error("unexpected API call", r.Method, r.URL)
					}
					w.WriteHeader(403)
					fmt.Fprint(w, `{"message":"unavailable"}`)
				}
			})}
			go func() { _ = server.Serve(listener) }()
			t.Cleanup(func() { _ = server.Close() })
			f.write(filepath.Join(directory, "config.yml"), "http_unix_socket: "+filepath.Join(directory, "api.sock")+"\n")
			f.base["GH_CONFIG_DIR"], f.base["GH_PATH_DISABLED"] = directory, "1"
			for _, key := range []string{"HTTP_PROXY", "HTTPS_PROXY", "http_proxy", "https_proxy"} {
				f.base[key] = "http://127.0.0.1:1"
			}
			f.base["NO_PROXY"], f.base["no_proxy"] = "", ""
			options := map[string]string{"github.action_ref": version, "inputs.prune": "autolinks"}
			if scenario == "empty-all" {
				options["inputs.prune"] = "all"
			}
			if scenario == "export" {
				os.Remove(path)
				exported := f.run(t, "export", "autolink-installation-fixture-token", map[string]string{"github.action_ref": version})
				if exported.status != 0 || exported.outputs["config-path"] != ".github/settings.yml" || strings.Contains(exported.log+exported.summary, "private.example") {
					t.Fatal(exported)
				}
				data, _ := os.ReadFile(path)
				if !strings.Contains(string(data), "reference_prefix: OLD_PRIVATE-") || !strings.Contains(string(data), "reference_format: numeric") {
					t.Fatal(string(data))
				}
				if len(events) != 0 {
					t.Fatal("export mutated", events)
				}
				return
			}
			for _, reporting := range []string{"detailed", "status-only"} {
				options["inputs.plan-reporting"] = reporting
				result := f.run(t, "plan", "autolink-installation-fixture-token", options)
				if result.status != 0 {
					t.Fatal(result)
				}
				if reporting == "status-only" && (strings.Contains(result.log+result.summary, "PRIVATE-") || strings.Contains(result.log+result.summary, "private.example")) {
					t.Fatal("private plan leaked", result)
				}
				if reporting == "detailed" && scenario != "matching" && !strings.Contains(result.summary, "private.example") {
					t.Fatal(result.summary)
				}
			}
			delete(options, "inputs.plan-reporting")
			if len(events) != 0 {
				t.Fatal("plan mutated", events)
			}
			result := f.run(t, "apply", "autolink-installation-fixture-token", options)
			failed := scenario == "readback-failure"
			if (result.status != 0) != failed || (!failed && len(result.outputs) != 4) || (failed && len(result.outputs) != 0) {
				t.Fatal(result)
			}
			if strings.Contains(result.log+result.summary, "UPSTREAM_PRIVATE_AUTOLINK") {
				t.Fatal("upstream detail leaked", result)
			}
			data, _ := os.ReadFile(path)
			if string(data) != input || f.git("rev-parse", "HEAD") != f.sha {
				t.Fatal("Action changed declaration")
			}
			mu.Lock()
			observedEvents := append([]string{}, events...)
			mu.Unlock()
			want := []string{}
			switch scenario {
			case "create-prune":
				want = []string{"create", "delete"}
			case "empty-all":
				want = []string{"delete"}
			case "readback-failure":
				want = []string{"create"}
			}
			if !reflect.DeepEqual(observedEvents, want) {
				t.Fatal(observedEvents, want)
			}
			if scenario == "matching" || scenario == "empty-all" {
				again := f.run(t, "apply", "autolink-installation-fixture-token", options)
				if again.status != 0 {
					t.Fatal(again)
				}
				mu.Lock()
				count := len(events)
				mu.Unlock()
				if count != len(want) {
					t.Fatal("repeat wrote")
				}
			}
			if scenario == "create-prune" {
				again := f.run(t, "apply", "autolink-installation-fixture-token", options)
				if again.status == 0 || len(again.outputs) != 0 {
					t.Fatal("IDless Action silently adopted creation", again)
				}
			}
		})
	}
}
