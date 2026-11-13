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

func testCompositeOIDCImmutable(t *testing.T, product []byte, version string) {
	for _, mode := range []string{"immutable-matching", "immutable-change", "immutable-unknown", "immutable-drift"} {
		t.Run(mode, func(t *testing.T) {
			f := newCompositeFixture(t)
			f.digest = fmt.Sprintf("%x", sha256.Sum256(product))
			f.write(f.base["MOCK_ASSET"], string(product))
			f.write(f.base["MOCK_CHECKSUMS"], f.digest+"  gh-settings-linux-amd64\n")
			f.write(filepath.Join(f.packagePath, "action/cli-version"), version+"\n")
			input := "version: daiksud/gh-settings@v0\nrepository: {settings: {actions: {oidc: {subject_claim: {use_immutable_subject_claim: false}}}}}\n"
			path := filepath.Join(f.caller, ".github/settings.yml")
			f.write(path, input)
			dir, err := os.MkdirTemp("/tmp", "ghs-oidc-immutable-")
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { os.RemoveAll(dir) })
			listener, err := net.Listen("unix", filepath.Join(dir, "api.sock"))
			if err != nil {
				t.Fatal(err)
			}
			var mu sync.Mutex
			writes := 0
			value := map[string]any{"use_default": false, "include_claim_keys": []any{"context", "repo"}, "use_immutable_subject": mode != "immutable-matching", "sub_claim_prefix": "PRIVATE_PREFIX_CANARY"}
			if mode == "immutable-unknown" {
				delete(value, "use_immutable_subject")
			}
			server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				w.Header().Set("Content-Type", "application/json")
				if r.Host != "api.github.com" || r.Header.Get("Authorization") != "token oidc-immutable-fixture" {
					t.Error("target/token boundary")
					w.WriteHeader(400)
					return
				}
				if r.Method == "GET" && r.URL.Path == "/repos/owner/config" {
					fmt.Fprint(w, `{"id":1,"full_name":"owner/config","visibility":"public","archived":false,"owner":{"id":2,"type":"User"}}`)
					return
				}
				if r.URL.Path != "/repos/owner/config/actions/oidc/customization/sub" {
					w.WriteHeader(403)
					fmt.Fprint(w, `{}`)
					return
				}
				if r.Method == "GET" {
					json.NewEncoder(w).Encode(value)
					return
				}
				if r.Method != "PUT" {
					t.Error(r.Method)
					w.WriteHeader(400)
					return
				}
				var payload map[string]any
				if json.NewDecoder(r.Body).Decode(&payload) != nil || !reflect.DeepEqual(payload, map[string]any{"use_default": false, "include_claim_keys": []any{"context", "repo"}, "use_immutable_subject": false}) {
					t.Error("active companion replay", payload)
				}
				writes++
				value = payload
				value["sub_claim_prefix"] = "CHANGED_PRIVATE_PREFIX_CANARY"
				if mode == "immutable-drift" {
					value["include_claim_keys"] = []any{"repo", "context"}
				}
				w.WriteHeader(201)
				fmt.Fprint(w, `{}`)
			})}
			go func() { _ = server.Serve(listener) }()
			t.Cleanup(func() { server.Close() })
			f.write(filepath.Join(dir, "config.yml"), "http_unix_socket: "+filepath.Join(dir, "api.sock")+"\n")
			f.base["GH_CONFIG_DIR"], f.base["GH_PATH_DISABLED"], f.base["GH_DEBUG"] = dir, "1", "api"
			for _, key := range []string{"HTTP_PROXY", "HTTPS_PROXY", "http_proxy", "https_proxy"} {
				f.base[key] = "http://127.0.0.1:1"
			}
			f.base["NO_PROXY"], f.base["no_proxy"] = "", ""
			opts := map[string]string{"github.action_ref": version}
			exported := f.run(t, "export", "oidc-immutable-fixture", opts)
			if exported.status != 0 || len(exported.outputs) != 5 || strings.Contains(exported.log, "PRIVATE_PREFIX_CANARY") {
				t.Fatal(exported)
			}
			data, _ := os.ReadFile(path)
			if !strings.Contains(string(data), "subject_claim_template:") || !strings.Contains(string(data), "use_immutable_subject_claim:") {
				t.Fatal("active export or prior leaf retention lost", string(data))
			}
			f.write(path, input)
			for _, privacy := range []string{"detailed", "status-only"} {
				result := f.run(t, "plan", "oidc-immutable-fixture", map[string]string{"github.action_ref": version, "inputs.plan-reporting": privacy})
				if (result.status != 0) != (mode == "immutable-unknown") || len(result.remainingReports) != 0 {
					t.Fatal(result)
				}
				if privacy == "status-only" && strings.Contains(result.summary, "use_immutable_subject_claim") {
					t.Fatal("private summary", result)
				}
				if strings.Contains(result.summary, "use_default_template") || strings.Contains(result.log, "PRIVATE_PREFIX_CANARY") {
					t.Fatal("unselected/raw evidence", result)
				}
			}
			result := f.run(t, "apply", "oidc-immutable-fixture", opts)
			success := mode == "immutable-matching" || mode == "immutable-change"
			if (result.status == 0) != success || success && len(result.outputs) != 4 || !success && len(result.outputs) != 0 {
				t.Fatal(result)
			}
			if mode == "immutable-drift" && !strings.Contains(result.log, "configuration may have changed") {
				t.Fatal(result)
			}
			if success {
				if repeat := f.run(t, "apply", "oidc-immutable-fixture", opts); repeat.status != 0 {
					t.Fatal(repeat)
				}
			}
			data, _ = os.ReadFile(path)
			if string(data) != input || f.git("rev-parse", "HEAD") != f.sha {
				t.Fatal("apply changed checkout")
			}
			mu.Lock()
			defer mu.Unlock()
			want := 1
			if mode == "immutable-matching" || mode == "immutable-unknown" {
				want = 0
			}
			if writes != want {
				t.Fatal("unexpected/repeated mutation", writes)
			}
		})
	}
}
