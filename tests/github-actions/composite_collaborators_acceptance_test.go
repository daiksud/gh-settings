//go:build acceptance

package githubactions

import (
	"crypto/sha256"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// The supplied product runs through the actual installer and composite steps;
// the Unix-socket service is the only invitation recipient.
func TestBuiltCompositeCollaborators(t *testing.T) {
	product, version := suppliedCompositeProduct(t)
	f := newCompositeFixture(t)
	f.base["GITHUB_ACTIONS"] = "true"
	f.digest = fmt.Sprintf("%x", sha256.Sum256(product))
	f.write(f.base["MOCK_ASSET"], string(product))
	f.write(f.base["MOCK_CHECKSUMS"], f.digest+"  gh-settings-linux-amd64\n")
	f.write(filepath.Join(f.packagePath, "action/cli-version"), version+"\n")
	const input = "version: daiksud/gh-settings@v0\nrepository: {settings: {collaborators: [{username: recipient-canary}]}}\n"
	path := filepath.Join(f.caller, ".github/settings.yml")
	f.write(path, input)
	f.git("add", ".github/settings.yml")
	f.git("-c", "user.name=Fixture", "-c", "user.email=fixture@example.test", "-c", "commit.gpgsign=false", "commit", "--quiet", "-m", "test: collaborator consumer declaration")
	f.sha = f.git("rev-parse", "HEAD")
	state, err := os.MkdirTemp("/tmp", "ghs-collab-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(state) })
	listener, err := net.Listen("unix", filepath.Join(state, "api.sock"))
	if err != nil {
		t.Fatal(err)
	}
	var mu sync.Mutex
	pending, puts := false, 0
	const user = `{"id":3123,"login":"recipient-canary","type":"User"}`
	const invitation = `{"id":7123,"repository":{"id":1,"full_name":"owner/config"},"invitee":` + user + `,"permissions":"write","created_at":"2026-11-13T00:00:00Z"}`
	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		if r.Host != "api.github.com" || r.Header.Get("Authorization") != "token collaborator-fixture-token" {
			t.Error("token/host escaped")
			w.WriteHeader(400)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.Method + " " + r.URL.Path {
		case "GET /repos/owner/config":
			fmt.Fprint(w, `{"id":1,"full_name":"owner/config","private":false,"archived":false,"owner":{"id":2,"login":"owner","type":"User"}}`)
		case "GET /repos/owner/config/collaborators":
			fmt.Fprint(w, "[]")
		case "GET /repos/owner/config/invitations":
			if pending {
				fmt.Fprint(w, "["+invitation+"]")
			} else {
				fmt.Fprint(w, "[]")
			}
		case "GET /users/recipient-canary", "GET /user/3123":
			fmt.Fprint(w, user)
		case "PUT /repos/owner/config/collaborators/recipient-canary":
			body, _ := io.ReadAll(r.Body)
			if len(body) != 0 || r.ContentLength != 0 {
				t.Error("unexpected permission payload")
			}
			puts++
			pending = true
			w.WriteHeader(201)
			fmt.Fprint(w, invitation)
		default:
			t.Error("unexpected collaborator API", r.Method, r.URL)
			w.WriteHeader(400)
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
	extra := map[string]string{"github.action_ref": version}
	first := f.run(t, "plan", "collaborator-fixture-token", extra)
	if first.status != 0 || !strings.Contains(first.summary, "send initial invitation") {
		t.Fatal("initial plan", first)
	}
	for i := 0; i < 2; i++ {
		result := f.run(t, "apply", "collaborator-fixture-token", extra)
		data, _ := os.ReadFile(path)
		if result.status != 0 || string(data) != input || len(result.remainingReports) != 0 {
			t.Fatal("Action apply/file boundary", result, string(data))
		}
	}
	mu.Lock()
	if puts != 1 {
		t.Fatal("Action resent invitation", puts)
	}
	mu.Unlock()
	bound := "version: daiksud/gh-settings@v0\nrepository: {settings: {collaborators: [{username: recipient-canary, id: 3123}]}}\n"
	f.write(path, bound)
	matching := f.run(t, "plan", "collaborator-fixture-token", extra)
	if matching.status != 0 || !strings.Contains(matching.summary, "acceptance pending; no new invitation") {
		t.Fatal("pending hidden", matching)
	}
	private := f.run(t, "plan", "collaborator-fixture-token", map[string]string{"github.action_ref": version, "inputs.plan-reporting": "status-only"})
	public := private.log + private.summary + fmt.Sprint(private.outputs)
	if private.status != 0 || strings.Contains(public, "recipient-canary") || strings.Contains(public, "3123") || strings.Contains(public, "7123") || len(private.remainingReports) != 0 {
		t.Fatal("private plan disclosed actor", private)
	}
}
