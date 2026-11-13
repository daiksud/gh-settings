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

func TestOIDCSubjectTemplateLifecycle(t *testing.T) {
	binary := acceptanceBinary(t)
	var mu sync.Mutex
	current := `{"use_default":true,"use_immutable_subject":true,"sub_claim_prefix":"private-prefix-canary"}`
	unavailable, drift := false, false
	puts := 0
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
				t.Error("repository mutation")
				w.WriteHeader(400)
				return
			}
			var v map[string]any
			json.Unmarshal([]byte(planResponse), &v)
			v["visibility"] = "public"
			v["owner"] = map[string]any{"type": "User", "id": 2}
			v["id"], v["full_name"], v["archived"] = 1, "owner/current", false
			json.NewEncoder(w).Encode(v)
		case "/repos/owner/current/topics":
			fmt.Fprint(w, `{"names":["acquired"]}`)
		case "/repos/owner/current/labels":
			fmt.Fprint(w, `[]`)
		case "/repos/owner/current/actions/oidc/customization/sub":
			if r.Method == "GET" {
				if unavailable {
					w.WriteHeader(403)
					fmt.Fprint(w, `{"message":"private-error-canary"}`)
					return
				}
				fmt.Fprint(w, current)
				return
			}
			if r.Method != "PUT" {
				t.Error(r.Method)
				w.WriteHeader(400)
				return
			}
			var p map[string]any
			json.NewDecoder(r.Body).Decode(&p)
			if len(p) != 3 || p["use_default"] != false || p["use_immutable_subject"] != true || !reflect.DeepEqual(p["include_claim_keys"], []any{"repo", "context"}) {
				t.Error("payload", p)
				w.WriteHeader(400)
				return
			}
			b, _ := json.Marshal(p)
			current = string(b)
			puts++
			if drift {
				current = strings.Replace(current, `"use_immutable_subject":true`, `"use_immutable_subject":false`, 1)
			}
			w.WriteHeader(201)
			fmt.Fprint(w, `{}`)
		default:
			t.Error("unexpected endpoint", r.Method, r.URL)
			w.WriteHeader(400)
		}
	}, false, "/actions/oidc/customization/sub", "/topics", "/labels")
	if err != nil {
		t.Fatal(err)
	}
	defer f.close()
	env := exportFixtureEnv(f.cfg)

	if err := setupOrdinaryOrigin(f.work, env); err != nil {
		t.Fatal(err)
	}
	run := func(args ...string) string {
		t.Helper()
		r, e := executeCLI(binary, f.work, f.root, env, args, false)
		if e != nil || r.runErr != nil || hasNonInfoDiagnostic(r.stderr.String()) {
			t.Fatal(args, e, r.runErr, r.stderr.String())
		}
		if strings.Contains(r.stderr.String(), "private-prefix-canary") || strings.Contains(r.stderr.String(), "private-error-canary") {
			t.Fatal("raw debug leak")
		}
		return r.stdout.String()
	}
	run("export")
	debug, debugErr := executeCLI(binary, f.work, f.root, append(append([]string{}, env...), "GH_DEBUG=api"), []string{"export"}, false)
	if debugErr != nil || debug.runErr != nil || strings.Contains(debug.stderr.String(), "private-prefix-canary") || strings.Contains(debug.stderr.String(), "/actions/oidc/customization/sub") {
		t.Fatal("OIDC debug privacy", debugErr, debug.runErr)
	}
	path := filepath.Join(f.work, ".github/settings.yml")
	b, _ := os.ReadFile(path)
	if !strings.Contains(string(b), "use_default_template: true") || strings.Contains(string(b), "private-prefix-canary") {
		t.Fatal(string(b))
	}
	input := "version: daiksud/gh-settings@v0\nrepository: {settings: {actions: {oidc: {subject_claim: {use_default_template: false, subject_claim_template: [repo, context]}}}}}\n"
	os.WriteFile(path, []byte(input), 0600)
	if out := run("plan"); !strings.Contains(out, "subject_claim") {
		t.Fatal(out)
	}
	run("apply")
	run("apply")
	mu.Lock()
	n := puts
	mu.Unlock()
	if n != 1 {
		t.Fatal("no-op wrote", n)
	}
	b, _ = os.ReadFile(path)
	if string(b) != input {
		t.Fatal("configuration changed")
	}
	mu.Lock()
	unavailable = true
	mu.Unlock()
	run("export")
	b, _ = os.ReadFile(path)
	if !strings.Contains(string(b), "subject_claim_template:") || !strings.Contains(string(b), "acquired") {
		t.Fatal("retention or sibling failed", string(b))
	}
	os.WriteFile(path, []byte(input), 0600)
	mu.Lock()
	unavailable = false
	drift = true
	current = `{"use_default":true,"use_immutable_subject":true}`
	mu.Unlock()
	r, e := executeCLI(binary, f.work, f.root, env, []string{"apply"}, false)
	if e != nil || r.runErr == nil || !strings.Contains(r.stderr.String(), "configuration may have changed") {
		t.Fatal(e, r.runErr, r.stderr.String())
	}
}
