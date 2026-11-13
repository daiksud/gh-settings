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

func TestOIDCImmutableLifecycle(t *testing.T) {
	for _, mode := range []string{"success", "missing-leaf", "readback-drift"} {
		t.Run(mode, func(t *testing.T) {
			var mu sync.Mutex
			state := map[string]any{"use_default": false, "include_claim_keys": []string{"context", "repo"}, "use_immutable_subject": true, "sub_claim_prefix": "PRIVATE_PREFIX"}
			if mode == "missing-leaf" {
				delete(state, "use_immutable_subject")
			}
			writes, calls := 0, 0
			f, err := startHTTPFixture(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				calls++
				w.Header().Set("Content-Type", "application/json")
				if r.Method == "GET" && r.URL.Path == "/repos/owner/current" {
					fmt.Fprint(w, `{"id":1,"full_name":"owner/current","visibility":"public","archived":false,"owner":{"id":2,"type":"User"}}`)
					return
				}
				if r.URL.Path != "/repos/owner/current/actions/oidc/customization/sub" {
					w.WriteHeader(403)
					fmt.Fprint(w, `{}`)
					return
				}
				if r.Method == "GET" {
					json.NewEncoder(w).Encode(state)
					return
				}
				if r.Method != "PUT" {
					t.Error(r.Method)
					w.WriteHeader(400)
					return
				}
				var payload map[string]any
				if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
					t.Error(err)
				}
				want := map[string]any{"use_default": false, "use_immutable_subject": false, "include_claim_keys": []any{"context", "repo"}}
				if !reflect.DeepEqual(payload, want) {
					t.Error("active companion replay", payload)
				}
				writes++
				state = payload
				state["sub_claim_prefix"] = "CHANGED_PRIVATE_PREFIX"
				if mode == "readback-drift" {
					state["include_claim_keys"] = []string{"repo", "context"}
				}
				w.WriteHeader(201)
				fmt.Fprint(w, `{}`)
			}, false, "/actions/oidc/customization/sub")
			if err != nil {
				t.Fatal(err)
			}
			defer f.close()
			env := append(labelIDFixtureEnv(f.cfg, false), "GH_DEBUG=api")
			if err := setupOrdinaryOrigin(f.work, env); err != nil {
				t.Fatal(err)
			}
			run := func(bad bool, args ...string) *executionResult {
				t.Helper()
				r, err := executeCLI(acceptanceBinary(t), f.work, f.root, env, args, false)
				if err != nil || (r.runErr != nil) != bad {
					t.Fatal(args, r.stderr.String(), r.runErr, err)
				}
				if strings.Contains(r.stdout.String()+r.stderr.String(), "PRIVATE_PREFIX") {
					t.Fatal("prefix leak")
				}
				return r
			}
			run(false, "export")
			path := filepath.Join(f.work, ".github/settings.yml")
			data, _ := os.ReadFile(path)
			if !strings.Contains(string(data), "subject_claim_template:") || strings.Contains(string(data), "use_immutable_subject_claim:") != (mode != "missing-leaf") {
				t.Fatal(string(data))
			}
			input := "version: daiksud/gh-settings@v0\nrepository: {settings: {actions: {oidc: {subject_claim: {use_immutable_subject_claim: false}}}}}\n"
			if err := os.WriteFile(path, []byte(input), 0600); err != nil {
				t.Fatal(err)
			}
			if mode == "missing-leaf" {
				r := run(false, "export")
				data, _ = os.ReadFile(path)
				if !strings.Contains(string(data), "use_immutable_subject_claim: false") || !strings.Contains(string(data), "subject_claim_template:") || !strings.Contains(r.stderr.String(), "use_immutable_subject_claim") {
					t.Fatal(string(data), r.stderr.String())
				}
				os.WriteFile(path, []byte(input), 0600)
			}
			mu.Lock()
			beforeCalls := calls
			mu.Unlock()
			run(false, "validate")
			mu.Lock()
			if beforeCalls != calls {
				t.Error("validate used remote")
			}
			mu.Unlock()
			plan := run(mode == "missing-leaf", "plan")
			if mode != "missing-leaf" && (!strings.Contains(plan.stdout.String(), "use_immutable_subject_claim") || strings.Contains(plan.stdout.String(), "use_default_template")) {
				t.Fatal("manufactured template change", plan.stdout.String())
			}
			result := run(mode != "success", "apply")
			if mode == "success" {
				run(false, "apply")
			} else if mode == "readback-drift" && !strings.Contains(result.stderr.String(), "configuration may have changed") {
				t.Fatal(result.stderr.String())
			}
			data, _ = os.ReadFile(path)
			if string(data) != input {
				t.Fatal("scalar apply rewrote file")
			}
			mu.Lock()
			defer mu.Unlock()
			want := 1
			if mode == "missing-leaf" {
				want = 0
			}
			if writes != want {
				t.Fatal("unexpected/repeated write", writes)
			}
		})
	}
}
