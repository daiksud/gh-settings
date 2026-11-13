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

func TestBuiltCompositeCollaboratorPrune(t *testing.T) {
	product, version := suppliedCompositeProduct(t)
	f := newCompositeFixture(t)
	f.base["GITHUB_ACTIONS"] = "true"
	f.base["GH_DEBUG"] = "api"
	f.digest = fmt.Sprintf("%x", sha256.Sum256(product))
	f.write(f.base["MOCK_ASSET"], string(product))
	f.write(f.base["MOCK_CHECKSUMS"], f.digest+"  gh-settings-linux-amd64\n")
	f.write(filepath.Join(f.packagePath, "action/cli-version"), version+"\n")
	const input = "version: daiksud/gh-settings@v0\nrepository: {settings: {collaborators: [{username: kept-canary}]}}\n"
	path := filepath.Join(f.caller, ".github/settings.yml")
	f.write(path, input)
	f.git("add", ".github/settings.yml")
	f.git("-c", "user.name=Fixture", "-c", "user.email=fixture@example.test", "-c", "commit.gpgsign=false", "commit", "--quiet", "-m", "test: collaborator prune consumer")
	f.sha = f.git("rev-parse", "HEAD")
	dir, err := os.MkdirTemp("/tmp", "ghs-collab-prune-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	listener, err := net.Listen("unix", filepath.Join(dir, "api.sock"))
	if err != nil {
		t.Fatal(err)
	}
	var mu sync.Mutex
	pending, active, fail := true, true, false
	events := []string{}
	const target = `{"id":1,"full_name":"owner/config","private":false,"archived":false,"owner":{"id":2,"login":"owner","type":"User"}}`
	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		if r.Host != "api.github.com" || r.Header.Get("Authorization") != "token installation-prune-fixture-token" {
			t.Error("credential or target escaped")
			w.WriteHeader(400)
			return
		}
		switch r.Method + " " + r.URL.Path {
		case "GET /repos/owner/config":
			fmt.Fprint(w, target)
		case "GET /user":
			w.WriteHeader(403)
			fmt.Fprint(w, `{"message":"not available to installation"}`)
		case "GET /installation/repositories":
			fmt.Fprint(w, `{"total_count":1,"repositories":[`+target+`]}`)
		case "GET /repos/owner/config/collaborators":
			rows := []any{map[string]any{"id": 5, "login": "kept-canary", "type": "User", "role_name": "write"}}
			if active {
				rows = append(rows, map[string]any{"id": 3, "login": "removed-canary", "type": "User", "role_name": "write"})
			}
			json.NewEncoder(w).Encode(rows)
		case "GET /repos/owner/config/invitations":
			if fail && !pending {
				w.WriteHeader(403)
				fmt.Fprint(w, `{"message":"PRIVATE_INVITER_EMAIL_CANARY"}`)
				return
			}
			if pending {
				fmt.Fprint(w, `[{"id":104,"repository":{"id":1,"full_name":"owner/config"},"invitee":{"id":4,"login":"cancelled-canary","type":"User"},"permissions":"write","created_at":"2026-11-14T00:00:00Z","inviter":{"id":99,"login":"PRIVATE-APP[bot]","type":"Bot"},"email":"PRIVATE_INVITEE_EMAIL_CANARY"}]`)
			} else {
				fmt.Fprint(w, "[]")
			}
		case "GET /users/kept-canary", "GET /user/5":
			fmt.Fprint(w, `{"id":5,"login":"kept-canary","type":"User"}`)
		case "GET /users/removed-canary", "GET /user/3":
			fmt.Fprint(w, `{"id":3,"login":"removed-canary","type":"User"}`)
		case "DELETE /repos/owner/config/invitations/104":
			if r.ContentLength != 0 {
				t.Error("DELETE body")
			}
			events = append(events, "cancel")
			pending = false
			w.WriteHeader(204)
		case "DELETE /repos/owner/config/collaborators/removed-canary":
			if r.ContentLength != 0 {
				t.Error("DELETE body")
			}
			events = append(events, "remove")
			active = false
			w.WriteHeader(204)
		default:
			t.Error("unexpected access endpoint", r.Method, r.URL)
			w.WriteHeader(400)
		}
	})}
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(func() { server.Close() })
	f.write(filepath.Join(dir, "config.yml"), "http_unix_socket: "+filepath.Join(dir, "api.sock")+"\n")
	f.base["GH_CONFIG_DIR"], f.base["GH_PATH_DISABLED"] = dir, "1"
	for _, key := range []string{"HTTP_PROXY", "HTTPS_PROXY", "http_proxy", "https_proxy"} {
		f.base[key] = "http://127.0.0.1:1"
	}
	f.base["NO_PROXY"], f.base["no_proxy"] = "", ""
	extra := map[string]string{"github.action_ref": version, "inputs.prune": "labels,collaborators"}
	plan := f.run(t, "plan", "installation-prune-fixture-token", extra)
	if plan.status != 0 || !strings.Contains(plan.summary, "cancel pending invitation id 104") || !strings.Contains(plan.summary, "remove collaborator grant") {
		t.Fatal(plan)
	}
	private := f.run(t, "plan", "installation-prune-fixture-token", map[string]string{"github.action_ref": version, "inputs.prune": "all", "inputs.plan-reporting": "status-only"})
	// Artifact hashes may contain identity digits; check the complete output contract separately.
	wantOutputs := map[string]string{"config-sha": f.sha, "target": "owner/config", "cli-version": version, "cli-sha256": f.digest, "plan-outcome": "changed"}
	public := private.log + private.summary
	if private.status != 0 || !reflect.DeepEqual(private.outputs, wantOutputs) || strings.Contains(public, "canary") || strings.Contains(public, "104") || strings.Contains(public, "PRIVATE") || len(private.remainingReports) != 0 {
		t.Fatal(private)
	}
	for i := 0; i < 2; i++ {
		result := f.run(t, "apply", "installation-prune-fixture-token", extra)
		data, _ := os.ReadFile(path)
		if result.status != 0 || string(data) != input || strings.Contains(result.log+result.summary, "PRIVATE") || result.outputs["config-sha"] != f.sha || result.outputs["target"] != "owner/config" {
			t.Fatal(result, string(data))
		}
	}
	mu.Lock()
	if fmt.Sprint(events) != "[cancel remove]" {
		t.Fatal(events)
	}
	pending, active, fail = true, true, true
	mu.Unlock()
	failure := f.run(t, "apply", "installation-prune-fixture-token", extra)
	data, _ := os.ReadFile(path)
	if failure.status == 0 || len(failure.outputs) != 0 || string(data) != input || strings.Contains(failure.log, "PRIVATE") || !strings.Contains(failure.log, "deletion acknowledged; resulting access unverified") {
		t.Fatal(failure, string(data))
	}
	mu.Lock()
	defer mu.Unlock()
	if fmt.Sprint(events) != "[cancel remove cancel]" {
		t.Fatal("later removal proceeded", events)
	}
}
