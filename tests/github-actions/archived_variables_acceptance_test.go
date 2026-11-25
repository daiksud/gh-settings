//go:build acceptance

package githubactions

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestBuiltCompositeArchivedVariables(t *testing.T) {
	const desired = "FIXTURE_ARCHIVED_DESIRED_CANARY"
	const remote = "FIXTURE_ARCHIVED_REMOTE_CANARY"
	product := suppliedCompositeProduct(t)
	for _, scenario := range []string{"matching", "different", "absent", "denied", "wrong name", "duplicate value", "value alias", "duplicate name", "name alias"} {
		t.Run(scenario, func(t *testing.T) {
			f := product.newFixture(t)
			f.base["GITHUB_ACTIONS"] = "true"

			input := "version: daiksud/gh-settings@v0\nrepository: {settings: {general: {danger_zone: {archive_this_repository: true}}, secrets_and_variables: {actions: {variables: [{name: BUILD_CHANNEL, value: " + desired + "}]}, agents: {variables: [{name: MODEL_FLAG, value: " + desired + "}]}}}}\n"
			config := filepath.Join(f.caller, ".github/settings.yml")
			f.write(config, input)
			dir, err := os.MkdirTemp("/tmp", "ghs-archived-vars-")
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = os.RemoveAll(dir) })
			listener, err := net.Listen("unix", filepath.Join(dir, "api.sock"))
			if err != nil {
				t.Fatal(err)
			}
			var mu sync.Mutex
			reads, writes, variableReads := 0, 0, 0
			server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				w.Header().Set("Content-Type", "application/json")
				if r.Method != http.MethodGet {
					writes++
					w.WriteHeader(400)
					return
				}
				if r.Host != "api.github.com" || r.Header.Get("Authorization") != "token archive-vars-fixture" {
					t.Error("unexpected target or credential")
					w.WriteHeader(400)
					return
				}
				switch r.URL.Path {
				case "/repos/owner/config":
					reads++
					fmt.Fprint(w, `{"id":42,"full_name":"owner/config","archived":true}`)
				case "/repos/owner/config/actions/variables/BUILD_CHANNEL", "/repos/owner/config/agents/variables/MODEL_FLAG":
					variableReads++
					if scenario == "denied" {
						w.WriteHeader(403)
						fmt.Fprintf(w, `{"message":%q}`, remote)
						return
					}
					if scenario == "absent" {
						w.WriteHeader(404)
						return
					}
					name := "BUILD_CHANNEL"
					if strings.Contains(r.URL.Path, "/agents/") {
						name = "MODEL_FLAG"
					}
					if scenario == "wrong name" {
						name = "OTHER"
					}
					if scenario == "duplicate value" {
						fmt.Fprintf(w, `{"name":%q,"value":%q,"value":%q}`, name, remote, desired)
						return
					}
					if scenario == "value alias" {
						fmt.Fprintf(w, `{"name":%q,"Value":%q,"value":%q}`, name, remote, desired)
						return
					}
					if scenario == "duplicate name" {
						fmt.Fprintf(w, `{"name":"OTHER","name":%q,"value":%q}`, name, desired)
						return
					}
					if scenario == "name alias" {
						fmt.Fprintf(w, `{"Name":"OTHER","name":%q,"value":%q}`, name, desired)
						return
					}
					value := desired
					if scenario == "different" {
						value = remote
					}
					fmt.Fprintf(w, `{"name":%q,"value":%q}`, name, value)
				default:
					t.Error("unexpected read")
					w.WriteHeader(400)
				}
			})}
			go func() { _ = server.Serve(listener) }()
			t.Cleanup(func() { _ = server.Close() })
			f.write(filepath.Join(dir, "config.yml"), "http_unix_socket: "+filepath.Join(dir, "api.sock")+"\n")
			f.base["GH_CONFIG_DIR"], f.base["GH_PATH_DISABLED"], f.base["GH_DEBUG"] = dir, "1", "api"
			for _, key := range []string{"HTTP_PROXY", "HTTPS_PROXY", "http_proxy", "https_proxy"} {
				f.base[key] = "http://127.0.0.1:1"
			}
			f.base["NO_PROXY"], f.base["no_proxy"] = "", ""
			if scenario == "matching" {
				plan := f.run(t, "plan", "archive-vars-fixture", map[string]string{"inputs.plan-reporting": "status-only"})
				if plan.status != 0 {
					t.Fatal("status-only plan failed", plan.status)
				}
				for _, output := range []string{plan.log, plan.summary, plan.api, plan.calls, fmt.Sprint(plan.outputs)} {
					if strings.Contains(output, desired) || strings.Contains(output, remote) {
						t.Fatal("variable value leaked into status-only plan")
					}
				}
				mu.Lock()
				reads, writes, variableReads = 0, 0, 0
				mu.Unlock()
			}
			result := f.run(t, "apply", "archive-vars-fixture", nil)
			if (result.status == 0) != (scenario == "matching") {
				redacted := strings.ReplaceAll(strings.ReplaceAll(result.log, desired, "[redacted]"), remote, "[redacted]")
				t.Fatal("unexpected Action outcome", result.status, redacted)
			}
			for _, output := range []string{result.log, result.summary, result.api, result.calls, fmt.Sprint(result.outputs)} {
				if strings.Contains(output, desired) || strings.Contains(output, remote) {
					t.Fatal("variable value leaked into Action output")
				}
			}
			mu.Lock()
			defer mu.Unlock()
			if writes != 0 || variableReads == 0 || scenario == "matching" && (reads != 2 || variableReads != 2) {
				t.Fatal("unexpected request counts", writes, reads, variableReads)
			}
			data, err := os.ReadFile(config)
			if err != nil || string(data) != input {
				t.Fatal("declaration changed", err)
			}
		})
	}
}
