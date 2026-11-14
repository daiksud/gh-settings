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

func TestRunnerLabelsLifecycle(t *testing.T) {
	binary := acceptanceBinary(t)
	var mu sync.Mutex
	labels := []string{"old"}
	puts, inventory := 0, 0
	unavailable, drift := false, false
	rawLabels := func() []any {
		v := []any{map[string]any{"id": 1, "name": "self-hosted-private", "type": "read-only"}}
		for _, n := range labels {
			v = append(v, map[string]any{"name": n, "type": "custom"})
		}
		return v
	}
	f, e := startHTTPFixture(func(w http.ResponseWriter, r *http.Request) {
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
				t.Error("unexpected repository write")
				w.WriteHeader(400)
				return
			}
			var v map[string]any
			json.Unmarshal([]byte(planResponse), &v)
			v["id"], v["full_name"], v["private"], v["archived"] = 1, "owner/current", false, false
			v["owner"] = map[string]any{"id": 2, "login": "owner", "type": "User"}
			json.NewEncoder(w).Encode(v)
		case "/repos/owner/current/topics":
			fmt.Fprint(w, `{"names":["acquired"]}`)
		case "/repos/owner/current/labels":
			fmt.Fprint(w, `[]`)
		case "/repos/owner/current/actions/runners":
			inventory++
			if unavailable {
				w.WriteHeader(403)
				fmt.Fprint(w, `{"message":"private-upstream"}`)
				return
			}
			v := map[string]any{"id": 23, "name": "private-runner-name", "os": "private-os", "status": "offline", "busy": true, "labels": rawLabels()}
			json.NewEncoder(w).Encode(map[string]any{"total_count": 1, "runners": []any{v}})
		case "/repos/owner/current/actions/runners/23":
			if r.Method != "GET" {
				t.Error("lifecycle write")
				w.WriteHeader(400)
				return
			}
			name := "private-runner-name"
			if drift && puts > 2 {
				name = "changed"
			}
			json.NewEncoder(w).Encode(map[string]any{"id": 23, "name": name, "os": "private-os", "status": "offline", "busy": true, "labels": rawLabels()})
		case "/repos/owner/current/actions/runners/23/labels":
			if r.Method == "PUT" {
				var p map[string]json.RawMessage
				if json.NewDecoder(r.Body).Decode(&p) != nil || len(p) != 1 || json.Unmarshal(p["labels"], &labels) != nil || labels == nil {
					t.Error("invalid payload")
				}
				puts++
			} else if r.Method != "GET" {
				t.Error("unexpected method", r.Method)
				w.WriteHeader(400)
				return
			}
			v := rawLabels()
			json.NewEncoder(w).Encode(map[string]any{"total_count": len(v), "labels": v})
		default:
			if r.Method != "GET" && r.URL.Path != "/graphql" {
				t.Error("unexpected mutation", r.Method, r.URL.Path)
			}
			w.WriteHeader(403)
			fmt.Fprint(w, `{"message":"unavailable"}`)
		}
	}, false, "/actions/runners", "/actions/runners/23", "/actions/runners/23/labels", "/topics", "/labels")
	if e != nil {
		t.Fatal(e)
	}
	defer f.close()
	env := exportFixtureEnv(f.cfg)
	if e = setupOrdinaryOrigin(f.work, env); e != nil {
		t.Fatal(e)
	}
	run := func(args ...string) string {
		t.Helper()
		r, e := executeCLI(binary, f.work, f.root, env, args, false)
		if e != nil || r.runErr != nil || hasNonInfoDiagnostic(r.stderr.String()) {
			t.Fatal(args, e, r.runErr, r.stderr.String())
		}
		if strings.Contains(r.stderr.String(), "private-") {
			t.Fatal("private observation leaked", r.stderr.String())
		}
		return r.stdout.String()
	}
	run("export")
	path := filepath.Join(f.work, ".github/settings.yml")
	b, _ := os.ReadFile(path)
	if !strings.Contains(string(b), "custom_labels:") || strings.Contains(string(b), "private-") {
		t.Fatal(string(b))
	}
	input := func(values string) string {
		return "version: daiksud/gh-settings@v0\nrepository: {settings: {actions: {runners: [{id: 23, custom_labels: " + values + "}]}}}\n"
	}
	desired := input("[GPU, accelerated]")
	os.WriteFile(path, []byte(desired), 0600)
	beforeInventory := func() int { mu.Lock(); defer mu.Unlock(); return inventory }()
	if out := run("plan"); !strings.Contains(out, "GPU") || !strings.Contains(out, "23") || strings.Contains(out, "private-") {
		t.Fatal(out)
	}
	run("apply")
	run("apply")
	mu.Lock()
	if puts != 1 || inventory != beforeInventory || !reflect.DeepEqual(labels, []string{"GPU", "accelerated"}) {
		t.Fatal(puts, inventory, labels)
	}
	mu.Unlock()
	b, _ = os.ReadFile(path)
	if string(b) != desired {
		t.Fatal("source changed")
	}
	os.WriteFile(path, []byte(input("[]")), 0600)
	run("apply")
	run("apply")
	mu.Lock()
	if puts != 2 || len(labels) != 0 {
		t.Fatal(puts, labels)
	}
	unavailable = true
	mu.Unlock()
	run("export")
	b, _ = os.ReadFile(path)
	if !strings.Contains(string(b), "custom_labels: []") || !strings.Contains(string(b), "acquired") {
		t.Fatal(string(b))
	}
	mu.Lock()
	unavailable = false
	drift = true
	mu.Unlock()
	os.WriteFile(path, []byte(desired), 0600)
	r, e := executeCLI(binary, f.work, f.root, append(append([]string{}, env...), "GH_DEBUG=api"), []string{"apply"}, false)
	if e != nil || r.runErr == nil || !strings.Contains(r.stderr.String(), "effects unknown") || strings.Contains(r.stderr.String(), "private-") || strings.Contains(r.stderr.String(), "/actions/runners") {
		t.Fatal(e, r.runErr, r.stderr.String())
	}
	b, _ = os.ReadFile(path)
	if string(b) != desired {
		t.Fatal("failed write changed source")
	}
}
