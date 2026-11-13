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
	"strings"
	"sync"
	"testing"
)

func TestBuiltCompositeDeployKeys(t *testing.T) {
	product, version := suppliedCompositeProduct(t)
	f := newCompositeFixture(t)
	f.base["GITHUB_ACTIONS"], f.base["GH_DEBUG"] = "true", "api"
	f.digest = fmt.Sprintf("%x", sha256.Sum256(product))
	f.write(f.base["MOCK_ASSET"], string(product))
	f.write(f.base["MOCK_CHECKSUMS"], f.digest+"  gh-settings-linux-amd64\n")
	f.write(filepath.Join(f.packagePath, "action/cli-version"), version+"\n")
	const publicKey = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIAEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEB"
	const input = "version: daiksud/gh-settings@v0\nrepository:\n  settings:\n    deploy_keys:\n      - title: deployment-title-canary\n        key: " + publicKey + " source-comment\n        allow_write_access: true\n"
	path := filepath.Join(f.caller, ".github/settings.yml")
	f.write(path, input)
	f.git("add", ".github/settings.yml")
	f.git("-c", "user.name=Fixture", "-c", "user.email=fixture@example.test", "-c", "commit.gpgsign=false", "commit", "--quiet", "-m", "test: deploy keys consumer")
	f.sha = f.git("rev-parse", "HEAD")
	dir, err := os.MkdirTemp("/tmp", "ghs-deploy-key-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	listener, err := net.Listen("unix", filepath.Join(dir, "api.sock"))
	if err != nil {
		t.Fatal(err)
	}
	var mu sync.Mutex
	exists, fail := false, false
	creates, deletes := 0, 0
	row := map[string]any{"id": 7, "title": "deployment-title-canary", "key": publicKey, "read_only": false, "verified": false, "enabled": true, "created_at": "2026-11-14T00:00:00Z", "url": "https://api.github.com/repos/owner/config/keys/7", "added_by": "PRIVATE_ATTRIBUTION_CANARY", "last_used": nil}
	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		if r.Host != "api.github.com" || r.Header.Get("Authorization") != "token deploy-key-fixture-token" {
			t.Error("credential/target boundary")
			w.WriteHeader(400)
			return
		}
		switch r.Method + " " + r.URL.Path {
		case "GET /repos/owner/config":
			fmt.Fprint(w, `{"id":1,"full_name":"owner/config","archived":false,"visibility":"public","owner":{"id":2,"login":"owner","type":"User"}}`)
		case "GET /repos/owner/config/keys":
			if fail && !exists {
				w.WriteHeader(403)
				fmt.Fprint(w, `{"message":"PRIVATE_RESPONSE_CANARY"}`)
				return
			}
			rows := []any{}
			if exists {
				rows = append(rows, row)
			}
			json.NewEncoder(w).Encode(rows)
		case "POST /repos/owner/config/keys":
			var payload map[string]any
			json.NewDecoder(r.Body).Decode(&payload)
			if len(payload) != 3 || payload["key"] != publicKey || payload["title"] != "deployment-title-canary" || payload["read_only"] != false {
				t.Error("payload")
			}
			creates++
			exists = true
			w.WriteHeader(201)
			json.NewEncoder(w).Encode(row)
		case "GET /repos/owner/config/keys/7":
			if !exists {
				w.WriteHeader(404)
				return
			}
			json.NewEncoder(w).Encode(row)
		case "DELETE /repos/owner/config/keys/7":
			deletes++
			exists = false
			w.WriteHeader(204)
		default:
			t.Error("unexpected endpoint", r.Method, r.URL)
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
	extra := map[string]string{"github.action_ref": version}
	planned := f.run(t, "plan", "deploy-key-fixture-token", extra)
	if planned.status != 0 || !strings.Contains(planned.summary, "SHA256:") || !strings.Contains(planned.summary, "allow_write_access=true") || strings.Contains(planned.summary, publicKey) || strings.Contains(planned.log, "PRIVATE") {
		t.Fatal(planned)
	}
	hidden := f.run(t, "plan", "deploy-key-fixture-token", map[string]string{"github.action_ref": version, "inputs.plan-reporting": "status-only"})
	if hidden.status != 0 || strings.Contains(hidden.summary+hidden.log, "deployment-title-canary") || strings.Contains(hidden.summary+hidden.log, "SHA256:") || len(hidden.remainingReports) != 0 {
		t.Fatal(hidden)
	}
	created := f.run(t, "apply", "deploy-key-fixture-token", extra)
	data, _ := os.ReadFile(path)
	if created.status != 0 || string(data) != input || created.outputs["config-sha"] != f.sha || created.outputs["target"] != "owner/config" || strings.Contains(created.log, "PRIVATE") || !strings.Contains(created.log, "SHA256:") {
		t.Fatal(created, string(data))
	}
	// Action does not bind. An IDless repeated apply must collide, not reinvite or adopt.
	collision := f.run(t, "apply", "deploy-key-fixture-token", extra)
	if collision.status == 0 || len(collision.outputs) != 0 {
		t.Fatal(collision)
	}
	bound := strings.Replace(input, "      - title:", "      - id: 7\n        title:", 1)
	f.write(path, bound)
	f.git("add", ".github/settings.yml")
	f.git("-c", "user.name=Fixture", "-c", "user.email=fixture@example.test", "-c", "commit.gpgsign=false", "commit", "--quiet", "-m", "test: bind verified deploy key")
	f.sha = f.git("rev-parse", "HEAD")
	for i := 0; i < 2; i++ {
		matching := f.run(t, "apply", "deploy-key-fixture-token", extra)
		if matching.status != 0 {
			t.Fatal(matching)
		}
	}
	empty := "version: daiksud/gh-settings@v0\nrepository: {settings: {deploy_keys: []}}\n"
	f.write(path, empty)
	f.git("add", ".github/settings.yml")
	f.git("-c", "user.name=Fixture", "-c", "user.email=fixture@example.test", "-c", "commit.gpgsign=false", "commit", "--quiet", "-m", "test: explicit deploy key deletion")
	f.sha = f.git("rev-parse", "HEAD")
	ordinary := f.run(t, "apply", "deploy-key-fixture-token", extra)
	if ordinary.status != 0 {
		t.Fatal(ordinary)
	}
	mu.Lock()
	if creates != 1 || deletes != 0 {
		t.Fatal(creates, deletes)
	}
	fail = true
	mu.Unlock()
	failed := f.run(t, "apply", "deploy-key-fixture-token", map[string]string{"github.action_ref": version, "inputs.prune": "deploy-keys"})
	data, _ = os.ReadFile(path)
	if failed.status == 0 || len(failed.outputs) != 0 || string(data) != empty || strings.Contains(failed.log, "PRIVATE") || !strings.Contains(failed.log, "unconfirmed") {
		t.Fatal(failed, string(data))
	}
	mu.Lock()
	defer mu.Unlock()
	if creates != 1 || deletes != 1 {
		t.Fatal(creates, deletes)
	}
}
