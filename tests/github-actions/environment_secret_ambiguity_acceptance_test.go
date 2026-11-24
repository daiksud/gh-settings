//go:build acceptance

package githubactions

import (
	"crypto/sha256"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestBuiltCompositeEnvironmentSecretAmbiguityStopsPrune(t *testing.T) {
	product, version := suppliedCompositeProduct(t)
	f := newCompositeFixture(t)
	f.digest = fmt.Sprintf("%x", sha256.Sum256(product))
	f.write(f.base["MOCK_ASSET"], string(product))
	f.write(f.base["MOCK_CHECKSUMS"], f.digest+"  gh-settings-linux-amd64\n")
	const input = "version: daiksud/gh-settings@v0\nrepository: {settings: {environments: []}}\n"
	f.write(filepath.Join(f.caller, ".github/settings.yml"), input)
	state, err := os.MkdirTemp("/tmp", "ghs-secret-integrity-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(state) })
	listener, err := net.Listen("unix", filepath.Join(state, "api.sock"))
	if err != nil {
		t.Fatal(err)
	}
	var mu sync.Mutex
	secretReads, writes := 0, 0
	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		if r.Host != "api.github.com" || r.Header.Get("Authorization") != "token fixture-token" {
			t.Error("escaped token/host boundary")
			w.WriteHeader(400)
			return
		}
		if r.Method != http.MethodGet {
			writes++
			w.WriteHeader(http.StatusNoContent)
			return
		}
		switch r.URL.Path {
		case "/repos/owner/config":
			fmt.Fprint(w, `{"id":71,"full_name":"owner/config","archived":false}`)
		case "/repos/owner/config/environments":
			fmt.Fprint(w, `{"total_count":1,"environments":[{"id":9,"name":"old"}]}`)
		case "/repos/owner/config/environments/old":
			fmt.Fprint(w, `{"id":9,"name":"old","protection_rules":[],"can_admins_bypass":false,"deployment_branch_policy":null}`)
		case "/repos/owner/config/environments/old/deployment_protection_rules":
			fmt.Fprint(w, `{"total_count":0,"custom_deployment_protection_rules":[]}`)
		case "/repos/owner/config/environments/old/variables":
			fmt.Fprint(w, `{"total_count":0,"variables":[]}`)
		case "/repos/owner/config/environments/old/secrets":
			secretReads++
			if secretReads == 1 {
				fmt.Fprint(w, `{"total_count":1,"secrets":[{"name":"PRIVATE_SECRET_NAME_CANARY","created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"}]}`)
			} else {
				fmt.Fprint(w, `{"total_count":1,"secrets":[{"name":"PRIVATE_SECRET_NAME_CANARY","created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-02T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"}]}`)
			}
		default:
			t.Error("unexpected read", r.URL)
			w.WriteHeader(400)
		}
	})}
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(func() { _ = server.Close() })
	f.write(filepath.Join(state, "config.yml"), "http_unix_socket: "+filepath.Join(state, "api.sock")+"\n")
	f.base["GH_CONFIG_DIR"], f.base["GH_PATH_DISABLED"] = state, "1"
	for _, key := range []string{"HTTP_PROXY", "HTTPS_PROXY", "http_proxy", "https_proxy"} {
		f.base[key] = "http://127.0.0.1:1"
	}
	f.base["NO_PROXY"], f.base["no_proxy"] = "", ""
	result := f.run(t, "apply", "fixture-token", map[string]string{"github.action_ref": version, "inputs.prune": "environments", "GH_DEBUG": "api"})
	if result.status == 0 || len(result.outputs) != 0 || strings.Contains(result.log+result.summary, "PRIVATE_SECRET_NAME_CANARY") {
		t.Fatalf("ambiguous metadata accepted, outputs published, or private evidence leaked: status=%d", result.status)
	}
	mu.Lock()
	defer mu.Unlock()
	if writes != 0 || secretReads < 2 {
		t.Fatalf("writes=%d secret reads=%d", writes, secretReads)
	}
	data, err := os.ReadFile(filepath.Join(f.caller, ".github/settings.yml"))
	if err != nil || string(data) != input || f.git("rev-parse", "HEAD") != f.sha {
		t.Fatal("Action changed YAML or commit")
	}
}
