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

func labelIDFixtureEnv(config string, actions bool) []string {
	var env []string
	for _, entry := range exportFixtureEnv(config) {
		if !strings.HasPrefix(entry, "GITHUB_ACTIONS=") {
			env = append(env, entry)
		}
	}
	return append(env, fmt.Sprintf("GITHUB_ACTIONS=%t", actions))
}

func TestLabelsCreatedIDWritebackAndRepeatApply(t *testing.T) {
	for _, mode := range []string{"writeback", "no-writeback", "actions"} {
		t.Run(mode, func(t *testing.T) {
			var mu sync.Mutex
			creates, created := 0, false
			fixture, err := startHTTPFixture(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				w.Header().Set("Content-Type", "application/json")
				switch r.Method {
				case "GET":
					labels := []map[string]any{{"id": 9, "name": "unmanaged", "color": "123456", "description": "retain"}}
					if created {
						labels = append(labels, map[string]any{"id": 17, "name": "new", "color": "abcdef", "description": ""})
					}
					json.NewEncoder(w).Encode(labels)
				case "POST":
					creates++
					created = true
					w.WriteHeader(201)
					fmt.Fprint(w, `{"id":17,"name":"new","color":"abcdef","description":""}`)
				default:
					t.Error("unexpected update/delete of unlisted object", r.Method)
					w.WriteHeader(400)
				}
			}, false)
			if err != nil {
				t.Fatal(err)
			}
			defer fixture.close()
			env := labelIDFixtureEnv(fixture.cfg, mode == "actions")
			if err := setupOrdinaryOrigin(fixture.work, env); err != nil {
				t.Fatal(err)
			}
			input := "# original comment\nversion: daiksud/gh-settings@v0\nrepository:\n  labels:\n    - name: new # consumer intent\n      color: '#ABCDEF'\n"
			if err := writeSettingsFile(fixture.work, input); err != nil {
				t.Fatal(err)
			}
			args := []string{"apply"}
			if mode == "no-writeback" {
				args = append(args, "--no-writeback")
			}
			first, err := executeCLI(acceptanceBinary(t), fixture.work, fixture.root, env, args, false)
			if err != nil || first.runErr != nil || first.stderr.Len() != 0 || !strings.Contains(first.stdout.String(), `Created labels "new": id 17`) {
				t.Fatalf("first=%+v err=%v", first, err)
			}
			data, err := os.ReadFile(filepath.Join(fixture.work, ".github", "settings.yml"))
			if err != nil {
				t.Fatal(err)
			}
			if mode == "writeback" {
				for _, want := range []string{"id: 17", "# original comment", "# consumer intent"} {
					if !strings.Contains(string(data), want) {
						t.Fatalf("lost %q: %s", want, data)
					}
				}
			} else if string(data) != input {
				t.Fatalf("disabled writeback modified YAML: %s", data)
			}
			second, err := executeCLI(acceptanceBinary(t), fixture.work, fixture.root, env, args, false)
			if err != nil {
				t.Fatal(err)
			}
			if mode == "writeback" {
				if second.runErr != nil || second.stderr.Len() != 0 || second.stdout.String() != "Applied repository settings.\n" {
					t.Fatalf("second=%+v", second)
				}
			} else if second.runErr == nil || !strings.Contains(second.stderr.String(), "specify id: 17") {
				t.Fatalf("collision was adopted: %+v", second)
			}
			mu.Lock()
			defer mu.Unlock()
			if creates != 1 {
				t.Fatalf("created %d times", creates)
			}
		})
	}
}

func TestLabelsPartialFailureRetainsRecoverableCreatedIDs(t *testing.T) {
	for _, mode := range []string{"writeback conflict", "later remote failure"} {
		t.Run(mode, func(t *testing.T) {
			var mu sync.Mutex
			var created []string
			var path, input string
			fixture, err := startHTTPFixture(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				w.Header().Set("Content-Type", "application/json")
				if r.Method == "GET" {
					fmt.Fprint(w, `[]`)
					return
				}
				if r.Method != "POST" {
					t.Error("unexpected mutation", r.Method)
					w.WriteHeader(400)
					return
				}
				var payload struct{ Name string }
				if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
					t.Error(err)
				}
				if payload.Name == "b" {
					w.WriteHeader(403)
					fmt.Fprint(w, `{"message":"fixture denied"}`)
					return
				}
				created = append(created, payload.Name)
				if mode == "writeback conflict" {
					if err := os.WriteFile(path, []byte(input+"# concurrent edit\n"), 0600); err != nil {
						t.Error(err)
					}
				}
				w.WriteHeader(201)
				fmt.Fprint(w, `{"id":17,"name":"a","color":"abcdef","description":""}`)
			}, false)
			if err != nil {
				t.Fatal(err)
			}
			defer fixture.close()
			env := labelIDFixtureEnv(fixture.cfg, false)
			if err := setupOrdinaryOrigin(fixture.work, env); err != nil {
				t.Fatal(err)
			}
			input = "version: daiksud/gh-settings@v0\nrepository:\n  labels:\n    - {name: a, color: '#abcdef'}\n    - {name: b, color: '#abcdef'}\n"
			path = filepath.Join(fixture.work, ".github", "settings.yml")
			if err := writeSettingsFile(fixture.work, input); err != nil {
				t.Fatal(err)
			}
			result, err := executeCLI(acceptanceBinary(t), fixture.work, fixture.root, env, []string{"apply"}, false)
			if err != nil || result.runErr == nil || !strings.Contains(result.stderr.String(), "id 17") {
				t.Fatalf("result=%+v err=%v", result, err)
			}
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if mode == "writeback conflict" {
				if string(data) != input+"# concurrent edit\n" || !strings.Contains(result.stderr.String(), "writeback failed") {
					t.Fatal(string(data), result.stderr.String())
				}
			} else if !strings.Contains(string(data), "id: 17") || !strings.Contains(result.stderr.String(), "403") {
				t.Fatal(string(data), result.stderr.String())
			}
			mu.Lock()
			defer mu.Unlock()
			if len(created) != 1 || created[0] != "a" {
				t.Fatal(created)
			}
		})
	}
}
