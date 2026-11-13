//go:build acceptance

package githubactions

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
)

func TestBuiltCompositeRulesetCoverage(t *testing.T) {
	product, version := suppliedCompositeProduct(t)
	for _, scenario := range []string{"matching", "changed", "readback drift"} {
		t.Run(scenario, func(t *testing.T) {
			f := newCompositeFixture(t)
			f.digest = fmt.Sprintf("%x", sha256.Sum256(product))
			f.write(f.base["MOCK_ASSET"], string(product))
			f.write(f.base["MOCK_CHECKSUMS"], f.digest+"  gh-settings-linux-amd64\n")
			f.write(filepath.Join(f.packagePath, "action/cli-version"), version+"\n")
			const input = "version: daiksud/gh-settings@v0\nrepository:\n  settings:\n    rulesets:\n      - id: 7\n        ruleset_name: PRIVATE_COVERAGE_RULE\n        enforcement_status: active\n        target_branches: {include_default_branch: true}\n        rules:\n          restrict_code_coverage: {minimum_line_coverage_percentage: 90.25}\n"
			path := filepath.Join(f.caller, ".github/settings.yml")
			f.write(path, input)
			directory, err := os.MkdirTemp("/tmp", "ghs-coverage-")
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = os.RemoveAll(directory) })
			listener, err := net.Listen("unix", filepath.Join(directory, "api.sock"))
			if err != nil {
				t.Fatal(err)
			}
			var mu sync.Mutex
			minimum := 80.5
			if scenario == "matching" {
				minimum = 90.25
			}
			state := map[string]any{"id": 7, "name": "PRIVATE_COVERAGE_RULE", "source_type": "Repository", "source": "owner/config", "target": "branch", "enforcement": "active",
				"conditions": map[string]any{"ref_name": map[string]any{"include": []any{"~DEFAULT_BRANCH"}, "exclude": []any{}}}, "bypass_actors": []any{},
				"rules": []any{map[string]any{"type": "code_coverage", "parameters": map[string]any{"minimum_coverage": minimum, "max_coverage_drop": float64(0)}}}}
			var operations []string
			writes := 0
			server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				const prefix = "/repos/owner/config/rulesets"
				if req.Host != "api.github.com" || req.Header.Get("Authorization") != "token coverage-fixture-token" || (req.URL.Path != prefix && req.URL.Path != prefix+"/7") {
					t.Error("escaped target/credential", req.Method, req.URL)
					w.WriteHeader(400)
					return
				}
				operations = append(operations, req.Method+strings.TrimPrefix(req.URL.Path, prefix))
				w.Header().Set("Content-Type", "application/json")
				if req.Method == "GET" {
					if req.URL.Query().Get("includes_parents") != "false" {
						t.Error("inherited Rulesets requested")
					}
					if req.URL.Path == prefix {
						fmt.Fprint(w, `[{"id":7,"source_type":"Repository","source":"owner/config"}]`)
						return
					}
					if writes > 0 && scenario == "readback drift" {
						state["rules"].([]any)[0].(map[string]any)["parameters"].(map[string]any)["minimum_coverage"] = 90.5
					}
					json.NewEncoder(w).Encode(state)
					return
				}
				if req.Method != "PUT" || req.URL.Path != prefix+"/7" {
					t.Error(req.Method, req.URL)
					w.WriteHeader(400)
					return
				}
				writes++
				var payload map[string]any
				if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
					t.Error(err)
				}
				wantRules := []any{map[string]any{"type": "code_coverage", "parameters": map[string]any{"minimum_coverage": 90.25, "max_coverage_drop": float64(0)}}}
				if _, hasID := payload["id"]; hasID || !reflect.DeepEqual(payload["rules"], wantRules) {
					t.Error("incomplete threshold ownership", payload)
				}
				payload["id"], payload["source"], payload["source_type"] = 7, "owner/config", "Repository"
				state = payload
				json.NewEncoder(w).Encode(state)
			})}
			go func() { _ = server.Serve(listener) }()
			t.Cleanup(func() { _ = server.Close() })
			f.write(filepath.Join(directory, "config.yml"), "http_unix_socket: "+filepath.Join(directory, "api.sock")+"\n")
			f.base["GH_CONFIG_DIR"], f.base["GH_PATH_DISABLED"] = directory, "1"
			for _, key := range []string{"HTTP_PROXY", "HTTPS_PROXY", "http_proxy", "https_proxy"} {
				f.base[key] = "http://127.0.0.1:1"
			}
			f.base["NO_PROXY"], f.base["no_proxy"] = "", ""
			for _, mode := range []string{"detailed", "status-only"} {
				result := f.run(t, "plan", "coverage-fixture-token", map[string]string{"github.action_ref": version, "inputs.plan-reporting": mode, "GH_DEBUG": "api"})
				if result.status != 0 || len(result.remainingReports) != 0 {
					t.Fatal(result)
				}
				if mode == "status-only" && (strings.Contains(result.log+result.summary, "PRIVATE_COVERAGE_RULE") || strings.Contains(result.log+result.summary, "90.25")) {
					t.Fatal("status-only leaked policy", result)
				}
				if mode == "detailed" && scenario != "matching" && !strings.Contains(result.summary, "minimum_line_coverage_percentage") {
					t.Fatal(result.summary)
				}
			}
			mu.Lock()
			if writes != 0 {
				t.Error("plan mutated policy")
			}
			operations = nil
			mu.Unlock()
			result := f.run(t, "apply", "coverage-fixture-token", map[string]string{"github.action_ref": version})
			success := scenario != "readback drift"
			if (result.status == 0) != success || success && len(result.outputs) != 4 || !success && len(result.outputs) != 0 {
				t.Fatal(result)
			}
			if !success && !strings.Contains(result.log, "may already have applied") {
				t.Fatal("readback uncertainty lost", result)
			}
			if success {
				again := f.run(t, "apply", "coverage-fixture-token", map[string]string{"github.action_ref": version})
				if again.status != 0 {
					t.Fatal(again)
				}
			}
			mu.Lock()
			defer mu.Unlock()
			want := []string{"GET", "GET/7"}
			if scenario != "matching" {
				want = append(want, "GET/7", "PUT/7", "GET/7")
			}
			if success {
				want = append(want, "GET", "GET/7")
			}
			wantWrites := 1
			if scenario == "matching" {
				wantWrites = 0
			}
			if !reflect.DeepEqual(operations, want) || writes != wantWrites {
				t.Fatal(operations, want, writes)
			}
			data, err := os.ReadFile(path)
			if err != nil || string(data) != input || f.git("rev-parse", "HEAD") != f.sha {
				t.Fatal("Action changed source", err)
			}
		})
	}
}
