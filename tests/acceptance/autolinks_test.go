//go:build acceptance

package acceptance

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"testing"
)

func TestAutolinksExecutableLifecycle(t *testing.T) {
	for _, mode := range []string{"create-prune", "ordinary", "no-writeback", "all", "omitted", "immutable", "collision", "readback-failure", "own-unarchive"} {
		t.Run(mode, func(t *testing.T) {
			var mu sync.Mutex
			archived := mode == "own-unarchive"
			events := []string{}
			next := 8
			row := func(id int, prefix string) map[string]any {
				return map[string]any{"id": id, "key_prefix": prefix, "url_template": "https://target.example/<num>?Case=A%2FB", "is_alphanumeric": false}
			}
			rows := map[int]map[string]any{7: row(7, "OLD-")}
			fixture, err := startHTTPFixture(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				w.Header().Set("Content-Type", "application/json")
				path := strings.TrimPrefix(r.URL.Path, "/repos/owner/current")
				switch {
				case r.Method == "GET" && path == "":
					json.NewEncoder(w).Encode(map[string]any{"id": 1, "full_name": "owner/current", "private": false, "archived": archived, "owner": map[string]any{"id": 2, "login": "owner", "type": "User"}})
				case r.Method == "PATCH" && path == "":
					if mode != "own-unarchive" {
						t.Error("unexpected PATCH")
					}
					var input map[string]any
					json.NewDecoder(r.Body).Decode(&input)
					if !reflect.DeepEqual(input, map[string]any{"archived": false}) {
						t.Error(input)
					}
					events = append(events, "unarchive")
					archived = false
					json.NewEncoder(w).Encode(map[string]any{"id": 1, "full_name": "owner/current", "archived": false})
				case r.Method == "GET" && path == "/autolinks":
					values := []any{}
					for _, v := range rows {
						values = append(values, v)
					}
					json.NewEncoder(w).Encode(values)
				case r.Method == "POST" && path == "/autolinks":
					var input map[string]any
					json.NewDecoder(r.Body).Decode(&input)
					if !reflect.DeepEqual(input, map[string]any{"key_prefix": "NEW-", "url_template": "https://target.example/<num>?Case=A%2FB", "is_alphanumeric": false}) {
						t.Error(input)
					}
					events = append(events, "create")
					rows[next] = row(next, "NEW-")
					w.WriteHeader(201)
					json.NewEncoder(w).Encode(rows[next])
					next++
				case strings.HasPrefix(path, "/autolinks/"):
					id, err := strconv.Atoi(strings.TrimPrefix(path, "/autolinks/"))
					if err != nil {
						t.Error(path)
					}
					if r.Method == "DELETE" {
						events = append(events, "delete")
						if id != 7 {
							t.Error("candidate expanded", id)
						}
						delete(rows, id)
						w.WriteHeader(204)
					} else if r.Method == "GET" {
						if mode == "readback-failure" && id == 8 {
							w.WriteHeader(403)
							fmt.Fprint(w, `{"message":"UPSTREAM_PRIVATE_URL"}`)
						} else {
							json.NewEncoder(w).Encode(rows[id])
						}
					} else {
						t.Error(r.Method, path)
					}
				default:
					w.WriteHeader(403)
					fmt.Fprint(w, `{"message":"unavailable"}`)
				}
			}, false, "/autolinks")
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(fixture.close)
			env := append(labelIDFixtureEnv(fixture.cfg, false), "GH_DEBUG=api")
			if err = setupOrdinaryOrigin(fixture.work, env); err != nil {
				t.Fatal(err)
			}
			entry := "{reference_prefix: NEW-, target_url: 'https://target.example/<num>?Case=A%2FB', reference_format: numeric}"
			if mode == "immutable" {
				entry = "{id: 7, reference_prefix: OLD-, target_url: 'https://different/<num>', reference_format: numeric}"
			}
			if mode == "collision" {
				entry = strings.Replace(entry, "NEW-", "OLD-", 1)
			}
			input := "settings: {autolinks: [" + entry + "]}"
			if mode == "all" {
				input = "settings: {autolinks: []}"
			}
			if mode == "omitted" {
				input = "settings: {}"
			}
			if mode == "own-unarchive" {
				input = "settings: {autolinks: [" + entry + "], general: {danger_zone: {archive_this_repository: false}}}"
			}
			original := repositoryInput(input + "\n")
			path := filepath.Join(fixture.work, ".github/settings.yml")
			os.MkdirAll(filepath.Dir(path), 0700)
			os.WriteFile(path, []byte(original), 0600)
			flags := []string{"--prune", "autolinks"}
			if mode == "ordinary" || mode == "no-writeback" {
				flags = nil
			}
			if mode == "all" {
				flags = []string{"--prune", "all"}
			}
			plan, err := executeCLI(acceptanceBinary(t), fixture.work, fixture.root, env, append([]string{"plan"}, flags...), false)
			bad := mode == "immutable" || mode == "collision"
			if err != nil || (plan.runErr != nil) != bad {
				t.Fatal(err, plan.runErr, plan.stderr.String())
			}
			if len(events) != 0 {
				t.Fatal("plan wrote", events)
			}
			if bad {
				apply, e := executeCLI(acceptanceBinary(t), fixture.work, fixture.root, env, append([]string{"apply"}, flags...), false)
				if e != nil || apply.runErr == nil || len(events) != 0 {
					t.Fatal(e, apply.runErr, events)
				}
				return
			}
			if mode == "no-writeback" {
				flags = append(flags, "--no-writeback")
			}
			applied, err := executeCLI(acceptanceBinary(t), fixture.work, fixture.root, env, append([]string{"apply"}, flags...), false)
			failed := mode == "readback-failure"
			if err != nil || (applied.runErr != nil) != failed {
				t.Fatal(err, applied.runErr, applied.stderr.String())
			}
			if strings.Contains(applied.stdout.String()+applied.stderr.String(), "UPSTREAM_PRIVATE_URL") {
				t.Fatal("debug leaked")
			}
			data, _ := os.ReadFile(path)
			if failed {
				if string(data) != original || !reflect.DeepEqual(events, []string{"create"}) || !strings.Contains(applied.stderr.String(), "id 8") {
					t.Fatal(events, string(data), applied.stderr.String())
				}
				return
			}
			if mode == "no-writeback" {
				if string(data) != original {
					t.Fatal("changed suppressed YAML")
				}
				return
			}
			if mode != "all" && mode != "omitted" && !strings.Contains(string(data), "id: 8") {
				t.Fatal("creation not bound", string(data))
			}
			count := len(events)
			repeat, err := executeCLI(acceptanceBinary(t), fixture.work, fixture.root, env, append([]string{"apply"}, flags...), false)
			if err != nil || repeat.runErr != nil || len(events) != count {
				t.Fatal("repeat wrote", err, repeat.runErr, repeat.stderr.String(), events)
			}
			if mode == "ordinary" && rows[7] == nil {
				t.Fatal("ordinary apply deleted old")
			}
			if mode == "own-unarchive" && (len(events) != 3 || events[0] != "unarchive" || events[1] != "create" || events[2] != "delete") {
				t.Fatal(events)
			}
		})
	}
}

func TestAutolinksExecutableExportAndRetention(t *testing.T) {
	for _, mode := range []string{"complete", "empty", "unavailable-initial", "unavailable-prior"} {
		t.Run(mode, func(t *testing.T) {
			fixture, err := startHTTPFixture(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if r.Method != "GET" {
					w.WriteHeader(403)
					return
				}
				switch r.URL.Path {
				case "/repos/owner/current":
					fmt.Fprint(w, `{"id":1,"full_name":"owner/current","private":false,"archived":false,"owner":{"id":2,"login":"owner","type":"User"}}`)
				case "/repos/owner/current/autolinks":
					if strings.HasPrefix(mode, "unavailable") {
						w.WriteHeader(403)
						fmt.Fprint(w, `{"message":"UPSTREAM_PRIVATE_URL"}`)
					} else if mode == "empty" {
						fmt.Fprint(w, `[]`)
					} else {
						fmt.Fprint(w, `[{"id":7,"key_prefix":"TICKET-","url_template":"https://target/<num>?q=A%2FB","is_alphanumeric":false}]`)
					}
				default:
					w.WriteHeader(403)
				}
			}, false, "/autolinks")
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(fixture.close)
			env := append(labelIDFixtureEnv(fixture.cfg, false), "GH_DEBUG=api")
			if err = setupOrdinaryOrigin(fixture.work, env); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(fixture.work, ".github/settings.yml")
			if mode == "unavailable-prior" {
				os.MkdirAll(filepath.Dir(path), 0700)
				os.WriteFile(path, []byte(repositoryInput("settings: {autolinks: [{id: 9, reference_prefix: PRIOR-, target_url: 'https://old/<num>', reference_format: alphanumeric}]}\n")), 0600)
			}
			result, err := executeCLI(acceptanceBinary(t), fixture.work, fixture.root, env, []string{"export"}, false)
			if err != nil || result.runErr != nil {
				t.Fatal(err, result.runErr, result.stderr.String())
			}
			data, _ := os.ReadFile(path)
			want := "autolinks: []"
			switch mode {
			case "complete":
				want = "reference_format: numeric"
			case "unavailable-prior":
				want = "reference_prefix: PRIOR-"
			case "unavailable-initial":
				want = "version:"
			}
			if !strings.Contains(string(data), want) {
				t.Fatal(string(data))
			}
			if mode == "unavailable-initial" && strings.Contains(string(data), "autolinks:") {
				t.Fatal("fabricated unavailable")
			}
			if strings.Contains(result.stderr.String(), "UPSTREAM_PRIVATE_URL") {
				t.Fatal(result.stderr.String())
			}
			valid, err := executeCLI(acceptanceBinary(t), fixture.work, fixture.root, env, []string{"validate"}, true)
			if err != nil || valid.runErr != nil {
				t.Fatal(err, valid.runErr, valid.stderr.String())
			}
		})
	}
}
