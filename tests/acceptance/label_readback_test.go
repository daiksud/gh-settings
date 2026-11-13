//go:build acceptance

package acceptance

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
)

func TestLabelCreateReadbackFailureRetainsRecoveryID(t *testing.T) {
	for _, failure := range []string{"stale", "denied"} {
		for _, earlier := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/earlier=%t", failure, earlier), func(t *testing.T) {
				var mu sync.Mutex
				var requests []string
				var source string
				failedName, failedID := "a", 17
				if earlier {
					failedName, failedID = "b", 18
				}
				remote := map[string]map[string]any{}
				fixture, err := startHTTPFixture(func(w http.ResponseWriter, r *http.Request) {
					mu.Lock()
					defer mu.Unlock()
					requests = append(requests, r.Method+" "+r.URL.EscapedPath())
					w.Header().Set("Content-Type", "application/json")
					switch {
					case r.Method == "GET" && r.URL.Path == "/repos/owner/current/labels":
						fmt.Fprint(w, `[]`)
					case r.Method == "POST" && r.URL.Path == "/repos/owner/current/labels":
						var payload struct{ Name, Color, Description string }
						if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
							t.Error(err)
						}
						id := 17 + len(remote)
						actual := map[string]any{"id": id, "name": payload.Name, "color": payload.Color, "description": payload.Description}
						w.WriteHeader(201)
						json.NewEncoder(w).Encode(actual) // Acknowledgment always claims success.
						if payload.Name == failedName {
							actual["color"] = "000000"
						}
						remote[payload.Name] = actual
					case r.Method == "GET" && strings.HasPrefix(r.URL.Path, "/repos/owner/current/labels/"):
						name := strings.TrimPrefix(r.URL.Path, "/repos/owner/current/labels/")
						data, err := os.ReadFile(source)
						if err != nil {
							t.Error(err)
						}
						if strings.Contains(string(data), fmt.Sprintf("id: %v", remote[name]["id"])) {
							t.Error("ID was written before independent readback", string(data))
						}
						if name == failedName && failure == "denied" {
							w.WriteHeader(http.StatusForbidden)
							fmt.Fprint(w, `{"message":"fixture readback denied"}`)
							return
						}
						json.NewEncoder(w).Encode(remote[name])
					default:
						t.Error("unexpected request", r.Method, r.URL)
						w.WriteHeader(400)
					}
				}, false)
				if err != nil {
					t.Fatal(err)
				}
				defer fixture.close()
				env := labelIDFixtureEnv(fixture.cfg, false)
				if err := setupOrdinaryOrigin(fixture.work, env); err != nil {
					t.Fatal(err)
				}
				input := "version: daiksud/gh-settings@v0\nrepository:\n  labels:\n    - {name: a, color: '#abcdef'}\n    - {name: b, color: '#abcdef'}\n    - {name: c, color: '#abcdef'}\n"
				if err := writeSettingsFile(fixture.work, input); err != nil {
					t.Fatal(err)
				}
				source = filepath.Join(fixture.work, ".github", "settings.yml")
				result, err := executeCLI(acceptanceBinary(t), fixture.work, fixture.root, env, []string{"apply"}, false)
				if err != nil || result.runErr == nil || result.stdout.Len() != 0 {
					t.Fatalf("expected failed readback, result=%+v err=%v", result, err)
				}
				for _, want := range []string{"read back label", "write may have applied", fmt.Sprintf(`reported repository.labels %q id %d`, failedName, failedID), "retain this ID before retrying"} {
					if !strings.Contains(result.stderr.String(), want) {
						t.Fatalf("missing %q: %s", want, result.stderr.String())
					}
				}
				data, err := os.ReadFile(source)
				if err != nil {
					t.Fatal(err)
				}
				if strings.Contains(string(data), fmt.Sprintf("id: %d", failedID)) || (!earlier && string(data) != input) || (earlier && !strings.Contains(string(data), "id: 17")) {
					t.Fatal("incorrect ID writeback", string(data))
				}
				want := []string{"GET /repos/owner/current/labels", "POST /repos/owner/current/labels", "GET /repos/owner/current/labels/a"}
				if earlier {
					want = append(want, "POST /repos/owner/current/labels", "GET /repos/owner/current/labels/b")
				}
				mu.Lock()
				defer mu.Unlock()
				if !reflect.DeepEqual(requests, want) {
					t.Fatalf("requests=%v want=%v", requests, want)
				}
			})
		}
	}
}

func TestLabelRenameReadbackPrecedesPruneAndRepeatIsUnchanged(t *testing.T) {
	for _, stale := range []bool{false, true} {
		t.Run(fmt.Sprintf("stale=%t", stale), func(t *testing.T) {
			var mu sync.Mutex
			var requests []string
			currentName, currentColor, removed := "old", "123456", false
			fixture, err := startHTTPFixture(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				requests = append(requests, r.Method+" "+r.URL.EscapedPath())
				w.Header().Set("Content-Type", "application/json")
				current := map[string]any{"id": 7, "name": currentName, "color": currentColor, "description": ""}
				remove := map[string]any{"id": 9, "name": "remove", "color": "abcdef", "description": ""}
				switch {
				case r.Method == "GET" && r.URL.Path == "/repos/owner/current/labels":
					labels := []map[string]any{current}
					if !removed {
						labels = append(labels, remove)
					}
					json.NewEncoder(w).Encode(labels)
				case r.Method == "GET" && r.URL.Path == "/repos/owner/current/labels/"+currentName:
					json.NewEncoder(w).Encode(current)
				case r.Method == "GET" && r.URL.Path == "/repos/owner/current/labels/remove" && !removed:
					json.NewEncoder(w).Encode(remove)
				case r.Method == "PATCH" && r.URL.Path == "/repos/owner/current/labels/old":
					var payload map[string]any
					if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
						t.Error(err)
					}
					if !reflect.DeepEqual(payload, map[string]any{"new_name": "renamed/docs ✨", "color": "abcdef", "description": ""}) {
						t.Error("unexpected PATCH", payload)
					}
					currentName = "renamed/docs ✨"
					if !stale {
						currentColor = "abcdef"
					}
					fmt.Fprint(w, `{"id":7,"name":"renamed/docs ✨","color":"abcdef","description":""}`)
				case r.Method == "DELETE" && r.URL.Path == "/repos/owner/current/labels/remove":
					removed = true
					w.WriteHeader(http.StatusNoContent)
				default:
					t.Error("unexpected request", r.Method, r.URL)
					w.WriteHeader(400)
				}
			}, false)
			if err != nil {
				t.Fatal(err)
			}
			defer fixture.close()
			env := labelIDFixtureEnv(fixture.cfg, false)
			if err := setupOrdinaryOrigin(fixture.work, env); err != nil {
				t.Fatal(err)
			}
			input := "version: daiksud/gh-settings@v0\nrepository:\n  labels:\n    - {id: 7, name: 'renamed/docs ✨', color: '#abcdef'}\n"
			if err := writeSettingsFile(fixture.work, input); err != nil {
				t.Fatal(err)
			}
			args := []string{"apply", "--prune", "labels"}
			result, err := executeCLI(acceptanceBinary(t), fixture.work, fixture.root, env, args, false)
			if err != nil || (result.runErr != nil) != stale {
				t.Fatalf("result=%+v err=%v", result, err)
			}
			want := []string{"GET /repos/owner/current/labels", "GET /repos/owner/current/labels", "GET /repos/owner/current/labels/old", "PATCH /repos/owner/current/labels/old", "GET /repos/owner/current/labels/renamed%2Fdocs%20%E2%9C%A8"}
			if stale {
				if result.stdout.Len() != 0 || !strings.Contains(result.stderr.String(), "read back label") || strings.Contains(result.stderr.String(), "reported repository.labels") {
					t.Fatal(result)
				}
			} else {
				want = append(want, "GET /repos/owner/current/labels/remove", "DELETE /repos/owner/current/labels/remove", "GET /repos/owner/current/labels")
				second, err := executeCLI(acceptanceBinary(t), fixture.work, fixture.root, env, args, false)
				if err != nil || second.runErr != nil || hasNonInfoDiagnostic(second.stderr.String()) {
					t.Fatalf("repeat=%+v err=%v", second, err)
				}
				want = append(want, "GET /repos/owner/current/labels", "GET /repos/owner/current/labels")
			}
			if err := checkSettingsUnchanged(fixture.work, input); err != nil {
				t.Fatal(err)
			}
			mu.Lock()
			defer mu.Unlock()
			if !reflect.DeepEqual(requests, want) || removed == stale {
				t.Fatalf("requests=%v want=%v removed=%t", requests, want, removed)
			}
		})
	}
}
