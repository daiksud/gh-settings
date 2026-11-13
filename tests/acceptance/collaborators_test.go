//go:build acceptance

package acceptance

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

const collaboratorProcessUser = `{"id":3123,"login":"recipient-canary","type":"User"}`
const collaboratorProcessInvite = `{"id":7123,"repository":{"id":1,"full_name":"owner/current"},"invitee":` + collaboratorProcessUser + `,"permissions":"write","created_at":"2026-11-13T00:00:00Z"}`

type collaboratorProcessState struct {
	mu          sync.Mutex
	state, mode string
	puts        int
}

func collaboratorProcessFixture(t *testing.T, state *collaboratorProcessState) (*apiFixture, []string) {
	t.Helper()
	f, err := startHTTPFixture(func(w http.ResponseWriter, r *http.Request) {
		state.mu.Lock()
		defer state.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		if servePRAccessFixtureRead(w, r) || serveIssueCreationPolicyFixtureRead(w, r) || serveSponsorshipsFixtureRead(w, r) {
			return
		}
		if isMergeMethodsFixtureRead(r) {
			fmt.Fprint(w, mergeMethodsFixtureResponse)
			return
		}
		switch r.URL.Path {
		case "/repos/owner/current":
			if r.Method != "GET" {
				t.Error("unrelated mutation")
				w.WriteHeader(400)
				return
			}
			var base map[string]any
			json.Unmarshal([]byte(planResponse), &base)
			base["id"], base["full_name"], base["private"], base["archived"] = 1, "owner/current", false, false
			base["owner"] = map[string]any{"id": 2, "login": "owner", "type": "User"}
			json.NewEncoder(w).Encode(base)
		case "/repos/owner/current/collaborators":
			if state.state == "active" {
				fmt.Fprint(w, `[{"id":3123,"login":"recipient-canary","type":"User","role_name":"write"}]`)
			} else {
				fmt.Fprint(w, "[]")
			}
		case "/repos/owner/current/invitations":
			if state.mode == "unavailable" || state.mode == "readback failure" && state.puts > 0 {
				w.WriteHeader(403)
				fmt.Fprint(w, `{"message":"private-recipient-email@example.test"}`)
				return
			}
			if state.state == "pending" {
				fmt.Fprint(w, "["+collaboratorProcessInvite+"]")
			} else {
				fmt.Fprint(w, "[]")
			}
		case "/users/recipient-canary", "/user/3123":
			fmt.Fprint(w, collaboratorProcessUser)
		case "/repos/owner/current/collaborators/recipient-canary":
			var body []byte
			if r.Body != nil {
				body, _ = io.ReadAll(r.Body)
			}
			if r.Method != "PUT" || len(body) != 0 || r.ContentLength != 0 {
				t.Error("unexpected access write", r.Method, string(body))
				w.WriteHeader(400)
				return
			}
			state.puts++
			state.state = "pending"
			w.WriteHeader(201)
			fmt.Fprint(w, collaboratorProcessInvite)
		case "/repos/owner/current/topics":
			fmt.Fprint(w, `{"names":[]}`)
		case "/repos/owner/current/labels":
			fmt.Fprint(w, `[]`)
		default:
			t.Error("unexpected endpoint", r.Method, r.URL)
			w.WriteHeader(400)
		}
	}, false, "/collaborators", "/invitations", "/topics", "/labels")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(f.close)
	env := append(exportFixtureEnv(f.cfg), "SETTINGS_TARGET=owner/current")
	if err := setupOrdinaryOrigin(f.work, env); err != nil {
		t.Fatal(err)
	}
	return f, env
}

func TestCollaboratorsExportProcess(t *testing.T) {
	state := &collaboratorProcessState{state: "pending"}
	f, env := collaboratorProcessFixture(t, state)
	r, err := executeCLI(acceptanceBinary(t), f.work, f.root, env, []string{"export"}, false)
	if err != nil || r.runErr != nil || hasNonInfoDiagnostic(r.stderr.String()) {
		t.Fatal(err, r.runErr, r.stderr.String())
	}
	path := filepath.Join(f.work, ".github/settings.yml")
	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), "collaborators:") || !strings.Contains(string(data), "username: recipient-canary") || !strings.Contains(string(data), "id: 3123") || strings.Contains(string(data), "7123") {
		t.Fatal("export did not bind the pending user identity", string(data))
	}
	state.mu.Lock()
	state.mode = "unavailable"
	state.mu.Unlock()
	r, err = executeCLI(acceptanceBinary(t), f.work, f.root, env, []string{"export"}, false)
	after, _ := os.ReadFile(path)
	if err != nil || r.runErr != nil || !strings.Contains(r.stderr.String(), "info: export repository.settings.collaborators:") || strings.Contains(r.stderr.String(), "private-recipient-email") || string(after) != string(data) {
		t.Fatal("unavailable export did not retain complete prior family", err, r.runErr, r.stderr.String(), string(after))
	}
}

func TestCollaboratorsLifecycleProcess(t *testing.T) {
	state := &collaboratorProcessState{}
	f, env := collaboratorProcessFixture(t, state)
	binary := acceptanceBinary(t)
	input := repositoryInput("settings: {collaborators: [{username: recipient-canary}]}\n")
	if err := writeSettingsFile(f.work, input); err != nil {
		t.Fatal(err)
	}
	run := func(failed bool, args ...string) string {
		t.Helper()
		r, err := executeCLI(binary, f.work, f.root, env, args, false)
		if err != nil || (r.runErr != nil) != failed {
			t.Fatal(args, err, r.runErr, r.stdout.String(), r.stderr.String())
		}
		return r.stdout.String() + r.stderr.String()
	}
	if out := run(false, "plan"); !strings.Contains(out, "send initial invitation") || !strings.Contains(out, "record verified user ID locally") {
		t.Fatal(out)
	}
	if out := run(false, "apply"); !strings.Contains(out, "acceptance pending") || !strings.Contains(out, "user ID recorded locally") || strings.Contains(out, "Created") {
		t.Fatal(out)
	}
	path := filepath.Join(f.work, ".github/settings.yml")
	bound, _ := os.ReadFile(path)
	if !strings.Contains(string(bound), "id: 3123") {
		t.Fatal(string(bound))
	}
	for _, command := range []string{"plan", "apply", "apply"} {
		out := run(false, command)
		if !strings.Contains(out, "acceptance pending") {
			t.Fatal("pending hidden", out)
		}
	}
	state.mu.Lock()
	if state.puts != 1 {
		t.Fatal(state.puts)
	}
	state.state = "active"
	state.mu.Unlock()
	if out := run(false, "apply"); !strings.Contains(out, "active access") {
		t.Fatal(out)
	}
	state.mu.Lock()
	state.state = ""
	state.mu.Unlock()
	if out := run(true, "apply"); !strings.Contains(out, "refusing to reinvite") {
		t.Fatal(out)
	}
	state.mu.Lock()
	if state.puts != 1 {
		t.Fatal("reinvited bound actor")
	}
	state.mu.Unlock()
	if err := writeSettingsFile(f.work, input); err != nil {
		t.Fatal(err)
	}
	if out := run(false, "apply", "--no-writeback"); !strings.Contains(out, "record this verified user ID") {
		t.Fatal(out)
	}
	unchanged, _ := os.ReadFile(path)
	if string(unchanged) != input {
		t.Fatal("no-writeback changed file")
	}
	state.mu.Lock()
	state.state = ""
	state.mode = "readback failure"
	state.puts = 0
	state.mu.Unlock()
	if out := run(true, "apply"); !strings.Contains(out, "user id 3123") || !strings.Contains(out, "invitation id 7123") || strings.Contains(out, "user ID recorded") {
		t.Fatal("missing recovery", out)
	}
	unchanged, _ = os.ReadFile(path)
	if string(unchanged) != input {
		t.Fatal("failed readback bound identity")
	}
}

func TestCollaboratorsValidateBeforeRemoteProcess(t *testing.T) {
	for _, entries := range []string{"null", "[null]", "[{username: person, id: null}]", "[{username: person, id: 0}]", "[{username: person, id: 9223372036854775808}]", "[{username: person@example.test}]", "[{username: ../person}]", "[{username: person}, {username: PERSON}]", "[{username: person, id: 3}, {username: other, id: 3}]", "[{username: person, permission: write}]", "[{id: 3}]"} {
		t.Run(entries, func(t *testing.T) {
			state := &collaboratorProcessState{}
			f, env := collaboratorProcessFixture(t, state)
			if err := writeSettingsFile(f.work, repositoryInput("settings: {collaborators: "+entries+"}\n")); err != nil {
				t.Fatal(err)
			}
			r, err := executeCLI(acceptanceBinary(t), f.work, f.root, env, []string{"apply"}, false)
			if err != nil || r.runErr == nil || f.reads.Load() != 0 {
				t.Fatal("invalid declaration reached API", err, r.runErr, f.reads.Load())
			}
		})
	}
}
