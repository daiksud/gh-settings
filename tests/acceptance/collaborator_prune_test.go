//go:build acceptance

package acceptance

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestCollaboratorPruneOmittedProcess(t *testing.T) {
	state := &collaboratorProcessState{state: "pending"}
	f, env := collaboratorProcessFixture(t, state)
	path := filepath.Join(f.work, ".github/settings.yml")
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(repositoryInput("settings: {}\n")), 0600); err != nil {
		t.Fatal(err)
	}
	for _, command := range []string{"plan", "apply"} {
		r, err := executeCLI(acceptanceBinary(t), f.work, f.root, env, []string{command, "--prune", "collaborators"}, false)
		if err != nil || r.runErr != nil {
			t.Fatal("omitted collaborators must remain unmanaged", err, r.runErr, r.stderr.String())
		}
	}
	if state.state != "pending" || state.puts != 0 {
		t.Fatal("unmanaged access changed")
	}
}

func TestCollaboratorPruneLifecycleProcess(t *testing.T) {
	for _, mode := range []string{"retained", "empty", "all", "ordinary", "guard", "readback", "second failure", "own unarchive"} {
		t.Run(mode, func(t *testing.T) {
			var mu sync.Mutex
			active := map[int]string{3: "removed", 5: "kept"}
			pending := map[int]string{4: "cancelled"}
			events := []string{}
			archived := mode == "own unarchive"
			person := func(id int, name string) map[string]any {
				return map[string]any{"id": id, "login": name, "type": "User"}
			}
			invite := func(id int, name string) map[string]any {
				inviter := person(2, "owner")
				if mode == "guard" {
					inviter = person(3, "removed")
				}
				return map[string]any{"id": id + 100, "repository": map[string]any{"id": 1, "full_name": "owner/current"}, "invitee": person(id, name), "permissions": "write", "created_at": "2026-11-14T00:00:00Z", "inviter": inviter, "email": "PRIVATE_INVITEE_EMAIL_CANARY"}
			}
			f, err := startHTTPFixture(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				w.Header().Set("Content-Type", "application/json")
				path := r.URL.Path
				if r.Method == "PATCH" {
					events = append(events, "unarchive")
					if path != "/repos/owner/current" || mode != "own unarchive" {
						t.Error("unrelated PATCH after unsafe preflight")
					}
					archived = false
					json.NewEncoder(w).Encode(map[string]any{"id": 1, "full_name": "owner/current", "archived": false})
					return
				}
				if r.Method == "PUT" {
					events = append(events, "invite")
					if path != "/repos/owner/current/collaborators/new" {
						t.Error("unexpected invite", path)
					}
					pending[6] = "new"
					w.WriteHeader(201)
					json.NewEncoder(w).Encode(invite(6, "new"))
					return
				}
				if r.Method == "DELETE" {
					if r.ContentLength != 0 {
						t.Error("DELETE payload")
					}
					events = append(events, path)
					if mode == "second failure" && strings.Contains(path, "/collaborators/") {
						w.WriteHeader(403)
						fmt.Fprint(w, `{"message":"PRIVATE_INVITER_EMAIL_CANARY"}`)
						return
					}
					switch path {
					case "/repos/owner/current/invitations/104":
						delete(pending, 4)
					case "/repos/owner/current/collaborators/removed":
						delete(active, 3)
					case "/repos/owner/current/collaborators/kept":
						delete(active, 5)
					default:
						t.Error("unselected deletion", path)
					}
					w.WriteHeader(204)
					return
				}
				if r.Method != "GET" {
					t.Error("unexpected method", r.Method)
					w.WriteHeader(400)
					return
				}
				switch path {
				case "/repos/owner/current":
					json.NewEncoder(w).Encode(map[string]any{"id": 1, "full_name": "owner/current", "private": false, "archived": archived, "owner": person(2, "owner")})
				case "/user":
					json.NewEncoder(w).Encode(person(2, "owner"))
				case "/installation/repositories":
					w.WriteHeader(403)
					fmt.Fprint(w, `{}`)
				case "/repos/owner/current/collaborators":
					rows := []any{}
					for id, name := range active {
						row := person(id, name)
						row["role_name"] = "write"
						rows = append(rows, row)
					}
					json.NewEncoder(w).Encode(rows)
				case "/repos/owner/current/invitations":
					deleted := false
					for _, event := range events {
						deleted = deleted || strings.Contains(event, "/invitations/")
					}
					if mode == "readback" && deleted {
						w.WriteHeader(403)
						fmt.Fprint(w, `{"message":"PRIVATE_INVITER_EMAIL_CANARY"}`)
						return
					}
					rows := []any{}
					for id, name := range pending {
						rows = append(rows, invite(id, name))
					}
					json.NewEncoder(w).Encode(rows)
				case "/users/kept", "/user/5":
					json.NewEncoder(w).Encode(person(5, "kept"))
				case "/users/new", "/user/6":
					json.NewEncoder(w).Encode(person(6, "new"))
				case "/users/removed", "/user/3":
					json.NewEncoder(w).Encode(person(3, "removed"))
				case "/users/cancelled", "/user/4":
					json.NewEncoder(w).Encode(person(4, "cancelled"))
				default:
					t.Error("unexpected endpoint", r.Method, r.URL)
					w.WriteHeader(400)
				}
			}, false, "/collaborators", "/invitations")
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(f.close)
			env := append(labelIDFixtureEnv(f.cfg, false), "SETTINGS_TARGET=owner/current", "GH_DEBUG=api")
			if err := setupOrdinaryOrigin(f.work, env); err != nil {
				t.Fatal(err)
			}
			collaborators := "[{username: kept}, {username: new}]"
			if mode == "empty" || mode == "all" {
				collaborators = "[]"
			}
			if mode == "guard" {
				collaborators = "[{username: cancelled}]"
			}
			settings := "settings: {collaborators: " + collaborators + "}"
			if mode == "own unarchive" || mode == "guard" {
				settings = "settings: {collaborators: " + collaborators + ", general: {danger_zone: {archive_this_repository: false}}}"
			}
			input := repositoryInput(settings + "\n")
			path := filepath.Join(f.work, ".github/settings.yml")
			os.MkdirAll(filepath.Dir(path), 0700)
			os.WriteFile(path, []byte(input), 0600)
			scope := "collaborators"
			if mode == "all" {
				scope = "all"
			}
			flags := []string{"--prune", scope}
			if mode == "ordinary" {
				flags = nil
			}
			if mode != "guard" && mode != "own unarchive" {
				r, err := executeCLI(acceptanceBinary(t), f.work, f.root, env, append([]string{"plan"}, flags...), false)
				if err != nil || r.runErr != nil {
					t.Fatal(err, r.runErr, r.stderr.String())
				}
				if mode != "ordinary" && (!strings.Contains(r.stdout.String(), "cancel pending invitation id 104") || !strings.Contains(r.stdout.String(), "remove collaborator grant")) {
					t.Fatal(r.stdout.String())
				}
				if len(events) != 0 {
					t.Fatal("plan mutated", events)
				}
			}
			r, err := executeCLI(acceptanceBinary(t), f.work, f.root, env, append([]string{"apply"}, flags...), false)
			failed := mode == "guard" || mode == "readback" || mode == "second failure"
			if err != nil || (r.runErr != nil) != failed {
				t.Fatal(err, r.runErr, r.stderr.String())
			}
			public := r.stdout.String() + r.stderr.String()
			if strings.Contains(public, "PRIVATE_") {
				t.Fatal("private evidence leaked", public)
			}
			if mode == "guard" {
				if len(events) != 0 || !strings.Contains(public, "before any writes") {
					t.Fatal(events, public)
				}
				return
			}
			if failed {
				want := "deletion acknowledged; resulting access unverified"
				if mode == "second failure" {
					want = "invitation cancelled"
				}
				if !strings.Contains(public, want) {
					t.Fatal(public)
				}
				return
			}
			mu.Lock()
			eventCount := len(events)
			if mode == "ordinary" {
				if len(active) != 2 || len(pending) != 2 {
					t.Fatal("ordinary apply deleted unlisted people", events)
				}
			} else if active[3] != "" || pending[4] != "" {
				t.Fatal("unlisted access retained", events)
			}
			mu.Unlock()
			data, _ := os.ReadFile(path)
			if mode == "retained" && (!strings.Contains(string(data), "id: 5") || !strings.Contains(string(data), "id: 6")) {
				t.Fatal("verified user IDs missing", string(data))
			}
			r, err = executeCLI(acceptanceBinary(t), f.work, f.root, env, append([]string{"apply"}, flags...), false)
			if err != nil || r.runErr != nil || len(events) != eventCount {
				t.Fatal("repeat mutated", err, r.runErr, r.stderr.String(), events)
			}
		})
	}
}
