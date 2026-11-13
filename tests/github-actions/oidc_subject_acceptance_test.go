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

func TestBuiltCompositeOIDCSubject(t *testing.T) {
	product, version := suppliedCompositeProduct(t)
	for _, scenario := range []string{"matching", "changed", "companion drift", "denied readback"} {
		t.Run(scenario, func(t *testing.T) {
			f := newCompositeFixture(t)
			f.digest = fmt.Sprintf("%x", sha256.Sum256(product))
			f.write(f.base["MOCK_ASSET"], string(product))
			f.write(f.base["MOCK_CHECKSUMS"], f.digest+"  gh-settings-linux-amd64\n")
			f.write(filepath.Join(f.packagePath, "action/cli-version"), version+"\n")
			const input = "version: daiksud/gh-settings@v0\nrepository: {settings: {actions: {oidc: {subject_claim: {use_default_template: false, subject_claim_template: [repo, context]}}}}}\n"
			path := filepath.Join(f.caller, ".github/settings.yml")
			f.write(path, input)
			state, err := os.MkdirTemp("/tmp", "ghs-oidc-")
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { os.RemoveAll(state) })
			listener, err := net.Listen("unix", filepath.Join(state, "api.sock"))
			if err != nil {
				t.Fatal(err)
			}
			var mu sync.Mutex
			enabled, patches := scenario == "matching", 0
			server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				if r.Host != "api.github.com" || r.Header.Get("Authorization") != "token oidc-fixture" {
					t.Error("target/token boundary")
					w.WriteHeader(400)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				if r.URL.Path == "/repos/owner/config" && r.Method == "GET" {
					fmt.Fprint(w, `{"id":1,"full_name":"owner/config","visibility":"public","archived":false,"owner":{"id":2,"type":"User"}}`)
					return
				}
				if r.URL.Path != "/repos/owner/config/actions/oidc/customization/sub" {
					t.Error("unexpected endpoint", r.URL)
					w.WriteHeader(400)
					return
				}
				if r.Method == "PUT" {
					var payload map[string]any
					if json.NewDecoder(r.Body).Decode(&payload) != nil || len(payload) != 3 || payload["use_default"] != false || payload["use_immutable_subject"] != true || !reflect.DeepEqual(payload["include_claim_keys"], []any{"repo", "context"}) {
						t.Error("unexpected payload", payload)
					}
					patches++
					enabled = true
					w.WriteHeader(201)
					fmt.Fprint(w, `{}`)
					return
				} else if r.Method != "GET" {
					t.Error("unexpected method", r.Method)
					w.WriteHeader(400)
					return
				}
				if r.Method == "GET" && patches > 0 && scenario == "denied readback" {
					w.WriteHeader(403)
					fmt.Fprint(w, `{"message":"private-canary"}`)
					return
				}
				immutable := true
				if patches > 0 && scenario == "companion drift" {
					immutable = false
				}
				if enabled {
					fmt.Fprintf(w, `{"use_default":false,"include_claim_keys":["repo","context"],"use_immutable_subject":%t}`, immutable)
				} else {
					fmt.Fprint(w, `{"use_default":true,"use_immutable_subject":true}`)
				}

			})}
			go func() { _ = server.Serve(listener) }()
			t.Cleanup(func() { server.Close() })
			f.write(filepath.Join(state, "config.yml"), "http_unix_socket: "+filepath.Join(state, "api.sock")+"\n")
			f.base["GH_CONFIG_DIR"], f.base["GH_PATH_DISABLED"] = state, "1"
			for _, key := range []string{"HTTP_PROXY", "HTTPS_PROXY", "http_proxy", "https_proxy"} {
				f.base[key] = "http://127.0.0.1:1"
			}
			f.base["NO_PROXY"], f.base["no_proxy"] = "", ""
			opts := map[string]string{"github.action_ref": version}
			detailed := f.run(t, "plan", "oidc-fixture", opts)
			if detailed.status != 0 || scenario != "matching" && !strings.Contains(detailed.summary, "subject_claim") {
				t.Fatal("detailed plan", detailed)
			}
			private := f.run(t, "plan", "oidc-fixture", map[string]string{"github.action_ref": version, "inputs.plan-reporting": "status-only"})
			if private.status != 0 || strings.Contains(private.log+private.summary, "subject_claim") || len(private.remainingReports) != 0 {
				t.Fatal("private report", private)
			}
			success := scenario == "matching" || scenario == "changed"
			attempts := 1
			if success {
				attempts = 2
			}
			for i := 0; i < attempts; i++ {
				result := f.run(t, "apply", "oidc-fixture", opts)
				if (result.status == 0) != success || success && len(result.outputs) != 4 || !success && len(result.outputs) != 0 {
					t.Fatal("publication", result)
				}
				if !success && (!strings.Contains(result.log, "configuration may have changed") || strings.Contains(result.log, "private-canary") || strings.Contains(result.log, "earlier successful writes remain applied")) {
					t.Fatal("uncertain failure", result.log)
				}
			}
			mu.Lock()
			want := 1
			if scenario == "matching" {
				want = 0
			}
			if patches != want {
				t.Fatal("repeated mutation", patches)
			}
			mu.Unlock()
			data, _ := os.ReadFile(path)
			if string(data) != input || f.git("rev-parse", "HEAD") != f.sha {
				t.Fatal("source checkout modified")
			}
		})
	}
}
