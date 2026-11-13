//go:build acceptance

package acceptance

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
)

const deployKeyOne = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIAEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEB"
const deployKeyTwo = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIAICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgIC"

func TestDeployKeysLifecycleProcess(t *testing.T) {
	binary := acceptanceBinary(t)
	var mu sync.Mutex
	keys := map[int]map[string]any{}
	next, creates, deletes := 10, 0, 0
	f, err := startHTTPFixture(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
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
				t.Error("unrelated repository mutation")
				w.WriteHeader(400)
				return
			}
			var value map[string]any
			json.Unmarshal([]byte(planResponse), &value)
			value["id"], value["full_name"], value["archived"], value["visibility"] = 1, "owner/current", false, "public"
			value["owner"] = map[string]any{"id": 2, "login": "owner", "type": "User"}
			json.NewEncoder(w).Encode(value)
		case "/repos/owner/current/topics":
			fmt.Fprint(w, `{"names":[]}`)
		case "/repos/owner/current/labels":
			fmt.Fprint(w, `[]`)
		case "/repos/owner/current/keys":
			if r.Method == "GET" {
				values := []map[string]any{}
				for id := 10; id < next; id++ {
					if value, ok := keys[id]; ok {
						values = append(values, value)
					}
				}
				json.NewEncoder(w).Encode(values)
				return
			}
			if r.Method != "POST" {
				t.Error("unexpected method", r.Method)
				w.WriteHeader(400)
				return
			}
			var input map[string]any
			if json.NewDecoder(r.Body).Decode(&input) != nil || len(input) != 3 || input["title"] != "shared-title" {
				t.Error("invalid create payload")
				w.WriteHeader(400)
				return
			}
			wantKey, wantReadOnly := deployKeyOne, true
			if creates == 1 {
				wantKey, wantReadOnly = deployKeyTwo, false
			}
			if input["key"] != wantKey || input["read_only"] != wantReadOnly {
				t.Error("key normalization or grant polarity")
				w.WriteHeader(400)
				return
			}
			id := next
			next++
			creates++
			value := map[string]any{"id": id, "title": input["title"], "key": input["key"], "read_only": input["read_only"], "url": fmt.Sprintf("https://api.github.com/repos/owner/current/keys/%d", id), "created_at": "2026-11-14T00:00:00Z", "verified": false, "enabled": true, "added_by": "PRIVATE_ATTRIBUTION_CANARY", "last_used": nil}
			keys[id] = value
			w.WriteHeader(201)
			json.NewEncoder(w).Encode(value)
		default:
			if strings.HasPrefix(r.URL.Path, "/repos/owner/current/keys/") {
				id, e := strconv.Atoi(strings.TrimPrefix(r.URL.Path, "/repos/owner/current/keys/"))
				value, exists := keys[id]
				if e != nil || !exists {
					w.WriteHeader(404)
					return
				}
				if r.Method == "GET" {
					json.NewEncoder(w).Encode(value)
					return
				}
				if r.Method == "DELETE" {
					delete(keys, id)
					deletes++
					w.WriteHeader(204)
					return
				}
			}
			t.Error("unexpected endpoint", r.Method, r.URL.Path)
			w.WriteHeader(400)
		}
	}, false, "/keys", "/topics", "/labels")
	if err != nil {
		t.Fatal(err)
	}
	defer f.close()
	env := exportFixtureEnv(f.cfg)
	// This lifecycle explicitly exercises guarded local ID binding under a host
	// that may be running Actions. The actual composite separately forbids binding.
	for i, value := range env {
		if strings.HasPrefix(value, "GITHUB_ACTIONS=") {
			env[i] = "GITHUB_ACTIONS=false"
		}
	}
	env = append(env, "GITHUB_ACTIONS=false")
	if err := setupOrdinaryOrigin(f.work, env); err != nil {
		t.Fatal(err)
	}
	run := func(args ...string) string {
		t.Helper()
		r, e := executeCLI(binary, f.work, f.root, env, args, false)
		if e != nil || r.runErr != nil || hasNonInfoDiagnostic(r.stderr.String()) {
			t.Fatal(args, e, r.runErr, r.stderr.String())
		}
		if strings.Contains(r.stdout.String()+r.stderr.String(), "PRIVATE_ATTRIBUTION_CANARY") {
			t.Fatal("private metadata leaked")
		}
		return r.stdout.String()
	}
	run("export")
	path := filepath.Join(f.work, ".github/settings.yml")
	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), "deploy_keys: []") {
		t.Fatal("complete empty Deploy keys were not exported")
	}
	input := "version: daiksud/gh-settings@v0\nrepository:\n  settings:\n    deploy_keys:\n      - title: shared-title\n        key: " + deployKeyOne + " source-comment\n        allow_write_access: false\n      - title: shared-title\n        key: " + deployKeyTwo + "\n        allow_write_access: true\n"
	if err := os.WriteFile(path, []byte(input), 0600); err != nil {
		t.Fatal(err)
	}
	out := run("plan")
	if !strings.Contains(out, "deploy_keys") || !strings.Contains(out, "SHA256:") || strings.Contains(out, deployKeyOne) {
		t.Fatal("safe typed create plan", out)
	}
	run("apply")
	run("apply")
	data, _ = os.ReadFile(path)
	if !strings.Contains(string(data), "id: 10") || !strings.Contains(string(data), "id: 11") || !strings.Contains(string(data), "source-comment") {
		t.Fatal("canonical-key binding lost ID or original comment")
	}
	mu.Lock()
	n := creates
	mu.Unlock()
	if n != 2 {
		t.Fatal("matching apply created keys", n)
	}
	if err := os.WriteFile(path, []byte("version: daiksud/gh-settings@v0\nrepository: {settings: {deploy_keys: []}}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	run("apply")
	mu.Lock()
	n = deletes
	mu.Unlock()
	if n != 0 {
		t.Fatal("ordinary empty list deleted keys")
	}
	out = run("plan", "--prune", "deploy-keys")
	if !strings.Contains(out, "delete") || strings.Contains(out, deployKeyOne) {
		t.Fatal("safe deletion plan", out)
	}
	run("apply", "--prune", "deploy-keys")
	run("apply", "--prune", "all")
	mu.Lock()
	defer mu.Unlock()
	if deletes != 2 || len(keys) != 0 {
		t.Fatal("explicit pruning incomplete", deletes, len(keys))
	}
}

func TestDeployKeysPrivateInputAndUnverifiedCreationProcess(t *testing.T) {
	binary := acceptanceBinary(t)
	var calls, posts atomic.Int64
	f, err := startHTTPFixture(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		switch r.Method + " " + r.URL.Path {
		case "GET /repos/owner/current":
			fmt.Fprint(w, `{"id":1,"full_name":"owner/current","archived":false,"visibility":"public","owner":{"id":2,"login":"owner","type":"User"}}`)
		case "GET /repos/owner/current/keys":
			fmt.Fprint(w, "[]")
		case "POST /repos/owner/current/keys":
			posts.Add(1)
			w.WriteHeader(201)
			fmt.Fprint(w, `{"id":7,"title":"private-attribution","added_by":"PRIVATE_RESPONSE_CANARY"}`)
		default:
			t.Error("unexpected operation", r.Method, r.URL)
			w.WriteHeader(400)
		}
	}, false, "/keys")
	if err != nil {
		t.Fatal(err)
	}
	defer f.close()
	env := append(exportFixtureEnv(f.cfg), "GH_DEBUG=api", "GITHUB_ACTIONS=false")
	if err := setupOrdinaryOrigin(f.work, env); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(f.work, ".github/settings.yml")
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	for _, material := range []string{"PRIVATE_INPUT_CANARY", "-----BEGIN OPENSSH PRIVATE KEY-----\nPRIVATE_INPUT_CANARY\n-----END OPENSSH PRIVATE KEY-----", "command=\"PRIVATE_INPUT_CANARY\" " + deployKeyOne, "PRIVATE_INPUT_CANARY\n" + deployKeyOne} {
		quoted, _ := json.Marshal(material)
		input := "version: daiksud/gh-settings@v0\nrepository: {settings: {deploy_keys: [{title: build, key: " + string(quoted) + ", allow_write_access: false}]}}\n"
		os.WriteFile(path, []byte(input), 0600)
		for _, command := range []string{"validate", "plan", "apply"} {
			r, e := executeCLI(binary, f.work, f.root, env, []string{command}, false)
			if e != nil || r.runErr == nil || strings.Contains(r.stdout.String()+r.stderr.String(), "PRIVATE_INPUT_CANARY") {
				t.Fatal(command, e, r.runErr, r.stderr.String())
			}
		}
	}
	if calls.Load() != 0 {
		t.Fatal("invalid public material reached remote", calls.Load())
	}
	input := "version: daiksud/gh-settings@v0\nrepository: {settings: {deploy_keys: [{title: build, key: " + deployKeyOne + ", allow_write_access: false}]}}\n"
	os.WriteFile(path, []byte(input), 0600)
	r, e := executeCLI(binary, f.work, f.root, env, []string{"apply"}, false)
	data, _ := os.ReadFile(path)
	if e != nil || r.runErr == nil || posts.Load() != 1 || string(data) != input || strings.Contains(r.stderr.String(), "PRIVATE_RESPONSE_CANARY") || !strings.Contains(r.stderr.String(), "id 7") || !strings.Contains(r.stderr.String(), "SHA256:") {
		t.Fatal(e, r.runErr, posts.Load(), r.stderr.String(), string(data))
	}
}
