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

func TestBuiltCompositeEnvironmentPrune(t *testing.T) {
	product, version := suppliedCompositeProduct(t)
	f := newCompositeFixture(t)
	f.digest = fmt.Sprintf("%x", sha256.Sum256(product))
	f.write(f.base["MOCK_ASSET"], string(product))
	f.write(f.base["MOCK_CHECKSUMS"], f.digest+"  gh-settings-linux-amd64\n")
	f.write(filepath.Join(f.packagePath, "action/cli-version"), version+"\n")
	const input = "version: daiksud/gh-settings@v0\nrepository: {settings: {environments: []}}\n"
	f.write(filepath.Join(f.caller, ".github/settings.yml"), input)
	state, err := os.MkdirTemp("/tmp", "ghs-prune-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(state) })
	listener, err := net.Listen("unix", filepath.Join(state, "api.sock"))
	if err != nil {
		t.Fatal(err)
	}
	var mu sync.Mutex
	deleted := false
	writes := 0
	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		if r.Host != "api.github.com" || r.Header.Get("Authorization") != "token prune-explicit-token" {
			t.Error("escaped token/host boundary")
			w.WriteHeader(400)
			return
		}
		if r.Method == "DELETE" && r.URL.Path == "/repos/owner/config/environments/PRIVATE_NAME_CANARY" {
			writes++
			deleted = true
			w.WriteHeader(204)
			return
		}
		if r.Method != "GET" {
			t.Error("unexpected mutation", r.Method, r.URL)
			w.WriteHeader(400)
			return
		}
		switch r.URL.Path {
		case "/repos/owner/config":
			fmt.Fprint(w, `{"id":71,"full_name":"owner/config","archived":false}`)
		case "/repos/owner/config/environments":
			if deleted {
				fmt.Fprint(w, `{"total_count":0,"environments":[]}`)
			} else {
				fmt.Fprint(w, `{"total_count":1,"environments":[{"id":9,"name":"PRIVATE_NAME_CANARY"}]}`)
			}
		case "/repos/owner/config/environments/PRIVATE_NAME_CANARY":
			fmt.Fprint(w, `{"id":9,"name":"PRIVATE_NAME_CANARY","protection_rules":[],"can_admins_bypass":false,"deployment_branch_policy":null}`)
		case "/repos/owner/config/environments/PRIVATE_NAME_CANARY/deployment_protection_rules":
			fmt.Fprint(w, `{"total_count":0,"custom_deployment_protection_rules":[]}`)
		case "/repos/owner/config/environments/PRIVATE_NAME_CANARY/variables":
			fmt.Fprint(w, `{"total_count":1,"variables":[{"name":"CHILD","value":"CHILD_VALUE_CANARY"}]}`)
		case "/repos/owner/config/environments/PRIVATE_NAME_CANARY/secrets":
			fmt.Fprint(w, `{"total_count":1,"secrets":[{"name":"SECRET_METADATA_CANARY","created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"}]}`)
		default:
			t.Error("unrelated read", r.URL)
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
	for _, mode := range []string{"detailed", "status-only"} {
		extra := map[string]string{"github.action_ref": version, "inputs.prune": "environments", "inputs.plan-reporting": mode, "GH_DEBUG": "api"}
		result := f.run(t, "plan", "prune-explicit-token", extra)
		if result.status != 0 || (mode == "status-only" && result.outputs["plan-outcome"] != "changed") || len(result.remainingReports) != 0 {
			t.Fatal(result)
		}
		combined := result.log + result.summary
		if strings.Contains(combined, "CHILD_VALUE_CANARY") || strings.Contains(combined, "SECRET_METADATA_CANARY") {
			t.Fatal("snapshot evidence leaked")
		}
		if mode == "status-only" && strings.Contains(combined, "PRIVATE_NAME_CANARY") {
			t.Fatal("private Environment name leaked")
		}
		if mode == "detailed" && !strings.Contains(result.summary, "settings.environments: delete") {
			t.Fatal(result.summary)
		}
		mu.Lock()
		if writes != 0 {
			t.Error("plan deleted", writes)
		}
		mu.Unlock()
	}
	extra := map[string]string{"github.action_ref": version, "inputs.prune": "all"}
	result := f.run(t, "apply", "prune-explicit-token", extra)
	if result.status != 0 || result.summary != "" {
		t.Fatal(result)
	}
	again := f.run(t, "apply", "prune-explicit-token", extra)
	if again.status != 0 {
		t.Fatal(again)
	}
	mu.Lock()
	if writes != 1 {
		t.Error("repeat deletion", writes)
	}
	mu.Unlock()
	data, _ := os.ReadFile(filepath.Join(f.caller, ".github/settings.yml"))
	if string(data) != input || f.git("rev-parse", "HEAD") != f.sha {
		t.Fatal("Action changed YAML/commit")
	}
}
